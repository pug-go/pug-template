package handler

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	pugv1pb "github.com/pug-go/pug-template/gen/pug/v1"
	"github.com/pug-go/pug-template/internal/handler/pugv1"
	"google.golang.org/grpc"
)

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterGrpcServices(server *grpc.Server) {
	pugv1pb.RegisterPugServiceServer(server, pugv1.NewPugService())
}

func (h *Handler) InitHttpRoutes(mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	// register here your grpc services for http handlers availability

	err := pugv1pb.RegisterPugServiceHandler(context.Background(), mux, conn)
	if err != nil {
		return fmt.Errorf("register pug grpc server: %w", err)
	}

	// register here your custom http routes
	// you can create custom handlers dir in internal for clean architecture
	// documentation: https://grpc-ecosystem.github.io/grpc-gateway/docs/operations/inject_router/

	//err = mux.HandlePath(http.MethodGet, "/custom", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	//	_, _ = w.Write([]byte("hello " + pathParams["name"]))
	//})
	//if err != nil {
	//	return err
	//}

	return nil
}
