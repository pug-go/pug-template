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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
	swagger "github.com/swaggo/http-swagger"
	"google.golang.org/grpc"

	"github.com/pug-go/pug-template/pkg/closer"
	"github.com/pug-go/pug-template/pkg/healthcheck"
	"github.com/pug-go/pug-template/pkg/middleware"
)

const gracefulTimeout = 10 * time.Second
const gracefulDelay = 3 * time.Second

type Handler interface {
	RegisterGrpcServices(server *grpc.Server)
	InitHttpRoutes(mux *runtime.ServeMux, conn *grpc.ClientConn) error
}

type GrpcServer interface {
	Run(port int16) error
	Stop(ctx context.Context) error
}

type HttpServer interface {
	Run(grpcPort, httpPort int16) error
	Stop(ctx context.Context) error
	Use(middleware func(next http.Handler) http.Handler)
}

type App struct {
	publicCloser *closer.Closer
	debugCloser  *closer.Closer
	hc           healthcheck.Handler
	config       Config
}

type Config struct {
	ServiceName string
	Domain      string
	GrpcPort    int16
	HttpPort    int16
	DebugPort   int16
}

func NewApp(config Config) (*App, error) {
	return &App{
		publicCloser: closer.NewCloser(),
		debugCloser:  closer.NewCloser(),
		hc:           healthcheck.NewHandler(),
		config:       config,
	}, nil
}

func (a *App) Run(
	grpcServer GrpcServer,
	httpServer HttpServer,
) {
	// starting servers
	go a.startGrpcServer(grpcServer)
	go a.startHttpServer(httpServer)
	go a.startDebugServer()

	// gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit
	a.shutdown()
}

func (a *App) startGrpcServer(grpcServer GrpcServer) {
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

	if err := grpcServer.Run(a.config.GrpcPort); err != nil {
		log.Fatalf("grpc: error occurred while running server: %s", err.Error())
	}
}

func (a *App) startHttpServer(httpServer HttpServer) {
	swaggerDomain := fmt.Sprintf("://%s:%d", a.config.Domain, a.config.DebugPort)
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http" + swaggerDomain, "https" + swaggerDomain},
		AllowedMethods:   []string{http.MethodHead, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	a.publicCloser.Add(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), gracefulTimeout)
		defer cancel()

		if err := httpServer.Stop(ctx); err != nil {
			return fmt.Errorf("http.public: error during shutdown: %w", err)
		}
		log.Info("http.public: gracefully stopped")

		return nil
	})

	httpServer.Use(c.Handler)

	log.Info("debug server listening on: ", a.config.DebugPort)
	if err := httpServer.Run(a.config.GrpcPort, a.config.HttpPort); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("http.public: error occurred while running server: %s", err.Error())
	}
}

func (a *App) startDebugServer() {
	mux := http.NewServeMux()

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
		swagger.UIConfig(map[string]string{
			"onComplete": fmt.Sprintf(
				`() => {
					window.ui.setHost('%s:%d');
				}`, a.config.Domain, a.config.HttpPort),
			"requestInterceptor": fmt.Sprintf(
				`(req) => {
					req.headers["X-Source"] = "%s";
					return req;
			  	}`, a.config.ServiceName),
		}),
	))
	mux.HandleFunc(healthcheck.CheckHandlerPathReadiness, a.hc.ReadyEndpointHandlerFunc)
	mux.HandleFunc(healthcheck.CheckHandlerPathLiveness, a.hc.LiveEndpointHandlerFunc)
	mux.HandleFunc("/metrics", promhttp.Handler().ServeHTTP)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", a.config.DebugPort),
		Handler: middleware.Recovery(mux),
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
