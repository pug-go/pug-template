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
	"github.com/pug-go/pug-template/pkg/closer"
	"github.com/pug-go/pug-template/pkg/healthcheck"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
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

func (a *App) startDebugServer() {
	// TODO
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
