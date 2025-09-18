package gwopts

import (
	"context"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var Default = []runtime.ServeMuxOption{
	runtime.WithErrorHandler(handleHttpError),
	runtime.WithMetadata(func(ctx context.Context, req *http.Request) metadata.MD {
		return metadata.Pairs("x-from-grpc-gateway", "true")
	}),
	runtime.WithForwardResponseOption(func(ctx context.Context, writer http.ResponseWriter, message proto.Message) error {
		pattern, ok := runtime.HTTPPathPattern(ctx)
		if ok {
			writer.Header().Add("pattern", pattern)
		}
		return nil
	}),
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
