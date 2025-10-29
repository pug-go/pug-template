package interceptor

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/pug-go/pug-template/pkg/promlib"
)

func UnaryServerPrometheus() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		started := time.Now()

		resp, err := handler(ctx, req)

		// ignore internal grpc-gateway http requests
		grpcGateway := isFromGrpcGateway(ctx)
		if grpcGateway {
			return resp, err
		}

		method := promlib.GetGrpcHandlerName(info.FullMethod)
		status := promlib.GrpcErrorToStatus(err)

		handleMetrics(started, method, status)

		return resp, err
	}
}

func StreamServerPrometheus() func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		started := time.Now()
		ctx := ss.Context()

		// ignore internal grpc-gateway http requests
		grpcGateway := isFromGrpcGateway(ctx)
		if grpcGateway {
			return nil
		}

		err := handler(srv, ss)

		method := promlib.GetGrpcHandlerName(info.FullMethod)
		status := promlib.GrpcErrorToStatus(err)

		handleMetrics(started, method, status)

		return err
	}
}

func handleMetrics(started time.Time, method, status string) {
	// pug_requests_total
	promlib.RequestsTotal.WithLabelValues(
		method,
		"grpc",
		status,
	).Inc()

	// pug_response_time_seconds
	promlib.ResponseTime.WithLabelValues(
		method,
		"grpc",
		status,
	).Observe(promlib.CalculateObservation(started))
}

func isFromGrpcGateway(ctx context.Context) bool {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		_, isGateway := md["x-from-grpc-gateway"]
		if isGateway {
			return true
		}
	}
	return false
}
