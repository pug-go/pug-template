package pug

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	log "github.com/sirupsen/logrus"
	swagger "github.com/swaggo/http-swagger"
	"google.golang.org/grpc"

	"github.com/pug-go/pug-template/pkg/closer"
	"github.com/pug-go/pug-template/pkg/healthcheck"
)

const gracefulTimeout = 10 * time.Second
const gracefulDelay = 3 * time.Second

type Handler interface {
	RegisterGrpcServices(server *grpc.Server)
	InitHttpRoutes(mux *runtime.ServeMux, conn *grpc.ClientConn) error
}

type Server interface {
	Run() error
	Stop(ctx context.Context) error
}

type App struct {
	publicCloser *closer.Closer
	debugCloser  *closer.Closer
	hc           healthcheck.Handler
}

func NewApp() (*App, error) {
	return &App{
		publicCloser: closer.NewCloser(),
		debugCloser:  closer.NewCloser(),
		hc:           healthcheck.NewHandler(),
	}, nil
}

func (a *App) Run(
	grpcServer Server,
	httpServer Server,
	debugPort int16,
) {
	// starting servers
	go a.startGrpcServer(grpcServer)
	go a.startHttpServer(httpServer)
	go a.startDebugServer(debugPort)

	// gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit
	a.shutdown()
}

func (a *App) startGrpcServer(grpcServer Server) {
	a.publicCloser.Add(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), gracefulTimeout)
		defer cancel()

		done := make(chan struct{})
		go func() {
			_ = grpcServer.Stop(ctx)
			close(done)
		}()
		select {
		case <-done:
			log.Info("grpc: gracefully stopped")
		case <-ctx.Done():
			err := fmt.Errorf("error during shutdown server: %w", ctx.Err())
			_ = grpcServer.Stop(ctx)
			return fmt.Errorf("grpc: force stopped: %w", err)
		}

		return nil
	})

	if err := grpcServer.Run(); err != nil {
		log.Fatalf("grpc: error occurred while running server: %s", err.Error())
	}
}

func (a *App) startHttpServer(httpServer Server) {
	a.publicCloser.Add(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), gracefulTimeout)
		defer cancel()

		if err := httpServer.Stop(ctx); err != nil {
			return fmt.Errorf("http.public: error during shutdown: %w", err)
		}
		log.Info("http.public: gracefully stopped")

		return nil
	})

	if err := httpServer.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("http.public: error occurred while running server: %s", err.Error())
	}
}

func (a *App) startDebugServer(debugPort int16) {
	mux := http.NewServeMux()

	// TODO: Pass data for proxy to http server

	swaggerRoute := "docs"
	swaggerJsonPath := fmt.Sprintf("/%s/swagger.json", swaggerRoute)
	swaggerPath := fmt.Sprintf("/%s/", swaggerRoute)

	mux.HandleFunc(swaggerJsonPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		http.ServeFile(w, r, "swagger.json")
	})
	mux.HandleFunc(swaggerPath, swagger.Handler(
		swagger.URL("swagger.json"),
		swagger.BeforeScript(
			`const UrlMutatorPlugin = (system) => ({
			  rootInjects: {
				setHost: (host) => {
				  const jsonSpec = system.getState().toJSON().spec.json;
				  const newJsonSpec = Object.assign({}, jsonSpec, { host });
		
				  return system.specActions.updateJsonSpec(newJsonSpec);
				},
				setTitle: (title) => {
				  const jsonSpec = system.getState().toJSON().spec.json;
				  const newJsonSpec = Object.assign({}, jsonSpec, { info: { title } });
		
				  return system.specActions.updateJsonSpec(newJsonSpec);
				},
				setBasePath: (basePath) => {
				  const jsonSpec = system.getState().toJSON().spec.json;
				  const newJsonSpec = Object.assign({}, jsonSpec, { basePath });
		
				  return system.specActions.updateJsonSpec(newJsonSpec);
				}
			  }
			});`),
		swagger.Plugins([]string{"UrlMutatorPlugin"}),
		//swagger.UIConfig(map[string]string{
		//	"onComplete": fmt.Sprintf(
		//		`() => {
		//			window.ui.setHost(''); // current host
		//			window.ui.setTitle('%s');
		//			window.ui.setBasePath('%s');
		//		}`, appName, basePath),
		//	"requestInterceptor": fmt.Sprintf(
		//		`(req) => {
		//			req.headers["X-Source"] = "%s";
		//			return req;
		//	  	}`, appName),
		//}),
	))
	mux.HandleFunc(healthcheck.CheckHandlerPathReadiness, a.hc.ReadyEndpointHandlerFunc)
	mux.HandleFunc(healthcheck.CheckHandlerPathLiveness, a.hc.LiveEndpointHandlerFunc)
	// mux.HandleFunc("/metrics", promhttp.Handler())

	// s.Use(middleware.Recovery)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", debugPort),
		Handler: mux,
	}

	a.debugCloser.Add(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		srv.SetKeepAlivesEnabled(false)
		if err := srv.Shutdown(ctx); err != nil {
			return fmt.Errorf("http.debug: error during shutdown: %w", err)
		}
		log.Info("http.debug: gracefully stopped")

		return nil
	})

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("http.debug: error occurred while running server: %s", err.Error())
	}
}

func (a *App) shutdown() {
	log.Info("shutdown process initiated")

	log.Info("waiting stop of traffic")
	time.Sleep(gracefulDelay)
	log.Info("shutting down")

	// stop http and grpc servers
	a.publicCloser.CloseAll()

	// stop debug server (swagger and so on)
	a.debugCloser.CloseAll()

	// stop other resources
	closer.CloseAll()
}
