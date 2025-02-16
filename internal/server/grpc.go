package server

import (
	"context"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type GrpcServer struct {
	address            string
	registerServicesFn func(server *grpc.Server)
	server             *grpc.Server
}

// TODO: Сюда хендлеры?

func NewGrpcServer(
	port int16,
	registerServicesFn func(server *grpc.Server),
) *GrpcServer {
	return &GrpcServer{
		address:            fmt.Sprintf(":%d", port),
		server:             grpc.NewServer(),
		registerServicesFn: registerServicesFn,
	}
}

func (s *GrpcServer) Run() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to start gRPC server on %s: %w", s.address, err)
	}

	s.registerServicesFn(s.server)

	log.Info("gRPC server listening on " + s.address)
	return s.server.Serve(listener)
}

func (s *GrpcServer) Stop(_ context.Context) error {
	s.server.GracefulStop()
	return nil
}
