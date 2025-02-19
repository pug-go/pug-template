package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/pug-go/pug-template/pkg/middleware"
)

type InitHttpRoutesFn func(mux *runtime.ServeMux, conn *grpc.ClientConn) error

type HttpServer struct {
	server           *http.Server
	initHttpRoutesFn InitHttpRoutesFn
	middlewares      []func(next http.Handler) http.Handler
	gwmux            *runtime.ServeMux
}

func NewHttpServer(initHttpRoutesFn InitHttpRoutesFn) *HttpServer {
	middlewares := []func(next http.Handler) http.Handler{
		// put your http middlewares here
		middleware.Prometheus,
		middleware.Recovery,
	}

	gwmux := runtime.NewServeMux(
		// put your opts here
		runtime.WithErrorHandler(handleHttpError),
		runtime.WithMetadata(func(ctx context.Context, req *http.Request) metadata.MD {
			return metadata.Pairs("x-from-grpc-gateway", "true")
		}),
	)

	return &HttpServer{
		server: &http.Server{
			ReadTimeout:  60 * time.Second,
			WriteTimeout: 60 * time.Second,
		},
		initHttpRoutesFn: initHttpRoutesFn,
		middlewares:      middlewares,
		gwmux:            gwmux,
	}
}

func (s *HttpServer) Run(grpcPort, httpPort int16) error {
	// create grpc client conn for internal http proxy
	conn, err := grpc.NewClient(
		fmt.Sprintf("0.0.0.0:%d", grpcPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to dial internal grpc conn: %s", err)
	}

	err = s.initHttpRoutesFn(s.gwmux, conn)
	if err != nil {
		return err
	}
	s.server.Handler = s.applyMiddlewares(s.gwmux)

	s.server.Addr = fmt.Sprintf(":%d", httpPort)
	log.Info("http server listening on: ", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *HttpServer) Stop(ctx context.Context) error {
	s.server.SetKeepAlivesEnabled(false)
	return s.server.Shutdown(ctx)
}

func (s *HttpServer) Use(middleware func(next http.Handler) http.Handler) {
	s.middlewares = append(s.middlewares, middleware)
}

func (s *HttpServer) applyMiddlewares(handler http.Handler) http.Handler {
	for _, m := range s.middlewares {
		handler = m(handler)
	}

	return handler
}

func handleHttpError(
	ctx context.Context,
	mux *runtime.ServeMux,
	marshaler runtime.Marshaler,
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	if s, ok := status.FromError(err); ok {
		// remove internal error from http response
		if s.Code() == codes.Internal {
			log.Error(s.Message())

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, err = w.Write([]byte(`{"code": 13, "message": "Internal"}`))
			if err != nil {
				log.Error(err)
			}
			return
		}
	}

	runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)
}
