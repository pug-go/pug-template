package server

import (
	"context"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type GrpcServer struct {
	server             *grpc.Server
	registerServicesFn func(server *grpc.Server)
}

func NewGrpcServer(registerServicesFn func(server *grpc.Server)) *GrpcServer {
	return &GrpcServer{
		server:             grpc.NewServer(),
		registerServicesFn: registerServicesFn,
	}
}

func (s *GrpcServer) Run(port int16) error {
	address := fmt.Sprintf(":%d", port)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to start gRPC server on %s: %w", address, err)
	}

	s.registerServicesFn(s.server)

	log.Info("gRPC server listening on " + address)
	return s.server.Serve(listener)
}

func (s *GrpcServer) Stop(_ context.Context) error {
	s.server.GracefulStop()
	return nil
}
