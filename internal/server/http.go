package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type HttpServer struct {
	server   *http.Server
	grpcPort int16
	gwmux    *runtime.ServeMux
}

type InitHttpRoutesFn func(mux *runtime.ServeMux, conn *grpc.ClientConn) error

func NewHttpServer(httpPort int16, grpcPort int16) *HttpServer {
	gwmux := runtime.NewServeMux()

	return &HttpServer{
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", httpPort),
			Handler:      gwmux,
			ReadTimeout:  60 * time.Second,
			WriteTimeout: 60 * time.Second,
		},
		grpcPort: grpcPort,
		gwmux:    gwmux,
	}
}

func (s *HttpServer) Run(initHttpRoutesFn InitHttpRoutesFn) error {
	// create grpc client conn for internal http proxy
	conn, err := grpc.NewClient(
		fmt.Sprintf("0.0.0.0:%d", s.grpcPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to dial internal grpc conn: %s", err)
	}

	err = initHttpRoutesFn(s.gwmux, conn)
	if err != nil {
		return err
	}

	return s.server.ListenAndServe()
}

func (s *HttpServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *HttpServer) SetKeepAlivesEnabled(v bool) {
	s.server.SetKeepAlivesEnabled(v)
}
