package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type InitHttpRoutesFn func(mux *runtime.ServeMux, conn *grpc.ClientConn) error

type HttpServer struct {
	server           *http.Server
	initHttpRoutesFn InitHttpRoutesFn
	gwmux            *runtime.ServeMux
	cors             *cors.Cors
}

func NewHttpServer(initHttpRoutesFn InitHttpRoutesFn) *HttpServer {
	gwmux := runtime.NewServeMux()

	return &HttpServer{
		server: &http.Server{
			Handler:      gwmux,
			ReadTimeout:  60 * time.Second,
			WriteTimeout: 60 * time.Second,
		},
		initHttpRoutesFn: initHttpRoutesFn,
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
	s.server.Handler = s.cors.Handler(s.gwmux)

	s.server.Addr = fmt.Sprintf(":%d", httpPort)
	log.Info("http server listening on: ", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *HttpServer) Stop(ctx context.Context) error {
	s.server.SetKeepAlivesEnabled(false)
	return s.server.Shutdown(ctx)
}

func (s *HttpServer) SetCors(opts cors.Options) {
	s.cors = cors.New(opts)
}
