package gwopts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
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
		if s.Code() == codes.Internal {
			// remove internal error from http response, but send to logs
			log.Error(s.Message())

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, err = w.Write([]byte(`{"code": 500, "message": "Internal Server Error"}`))
			if err != nil {
				log.Error(err)
			}
			return
		}
		if s.Code() == codes.InvalidArgument {
			errorsMap := map[string][]string{}
			var violations *validate.Violations

			for _, detail := range s.Details() {
				if violations, ok = detail.(*validate.Violations); ok {
					for _, violation := range violations.GetViolations() {
						field := fieldPath(violation)
						if field == "" {
							field = "unknown"
						}
						errorsMap[field] = append(errorsMap[field], violation.GetMessage())
					}
				}
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			err = json.NewEncoder(w).Encode(map[string]any{
				"code":    400,
				"message": "Bad Request",
				"errors":  errorsMap,
			})
			if err != nil {
				log.Error(err)
			}
			return
		}
		// TODO: Добавить остальные ошибки из responser.go:
		// Not found
		// Forbidden
		// Unauthorized
	}

	runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)
}

func fieldPath(v *validate.Violation) string {
	if v.GetField() == nil || len(v.GetField().GetElements()) == 0 {
		return ""
	}

	var parts []string
	for _, el := range v.GetField().GetElements() {
		name := el.GetFieldName()
		if name == "" {
			continue
		}

		if el.GetSubscript() != nil { // e.g. repeated
			name = fmt.Sprintf("%s[%d]", name, el.GetIndex())
		}
		parts = append(parts, name)
	}

	return strings.Join(parts, ".")
}
