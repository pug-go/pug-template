package server

import (
	"context"
	"fmt"
	"net"

	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcRecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/pug-go/pug-template/pkg/interceptor"
)

type GrpcServer struct {
	server             *grpc.Server
	registerServicesFn func(server *grpc.Server)
}

func NewGrpcServer(registerServicesFn func(server *grpc.Server)) *GrpcServer {
	return &GrpcServer{
		server: grpc.NewServer(
			grpc.UnaryInterceptor(grpcMiddleware.ChainUnaryServer(
				interceptor.UnaryServerPrometheus(),
				// put your interceptors here
				grpcRecovery.UnaryServerInterceptor(), // should be last
			)),
			grpc.StreamInterceptor(grpcMiddleware.ChainStreamServer(
				interceptor.StreamServerPrometheus(),
				// put your interceptors here
				grpcRecovery.StreamServerInterceptor(), // should be last
			)),
		),
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
