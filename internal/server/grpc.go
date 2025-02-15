package server

import (
	"fmt"
	"net"

	"google.golang.org/grpc"
)

type GrpcServer struct {
	server  *grpc.Server
	address string
}

type RegisterGrpcServicesFn func(server *grpc.Server)

func NewGrpcServer(port int16) *GrpcServer {
	return &GrpcServer{
		address: fmt.Sprintf(":%d", port),
		server:  grpc.NewServer(),
	}
}

func (s *GrpcServer) Run(registerServicesFn RegisterGrpcServicesFn) error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to start gRPC server on %s: %w", s.address, err)
	}

	registerServicesFn(s.server)

	return s.server.Serve(listener)
}

func (s *GrpcServer) Stop() {
	s.server.Stop()
}

func (s *GrpcServer) GracefulStop() {
	s.server.GracefulStop()
}
