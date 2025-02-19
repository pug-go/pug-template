package promlib

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//goland:noinspection GoSnakeCaseUsage
const (
	Status_Unknown       = "unknown"
	Status_InternalError = "internal_error"
	Status_ClientError   = "client_error"
	Status_Redirection   = "redirection"
	Status_OK            = "ok"
)

var (
	ResponseTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "pug",
		Name:      "response_time_seconds",
		Help:      "Histogram of application RT for any kind of requests: HTTP, gRPC (seconds).",
	}, []string{"handler", "protocol", "status"})
	RequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "pug",
		Name:      "requests_total",
		Help:      "Counter of application requests for any kind of requests: HTTP, gRPC.",
	}, []string{"handler", "protocol", "status"})
)

func HttpCodeToStatus(code int) string {
	switch {
	case code >= 500:
		return Status_InternalError
	case code >= 400:
		return Status_ClientError
	case code >= 300:
		return Status_Redirection
	default:
		return Status_OK
	}
}

func GrpcErrorToStatus(err error) string {
	if err == nil {
		return Status_OK
	}

	st, ok := status.FromError(err)
	if !ok {
		return Status_Unknown
	}

	switch st.Code() {
	// internal_error
	case codes.Unimplemented:
		fallthrough
	case codes.Internal:
		fallthrough
	case codes.Unavailable:
		fallthrough
	case codes.Unknown:
		fallthrough
	case codes.DataLoss:
		fallthrough
	case codes.DeadlineExceeded:
		return Status_InternalError
	// client_error
	case codes.Canceled:
		fallthrough
	case codes.InvalidArgument:
		fallthrough
	case codes.NotFound:
		fallthrough
	case codes.AlreadyExists:
		fallthrough
	case codes.PermissionDenied:
		fallthrough
	case codes.Unauthenticated:
		fallthrough
	case codes.ResourceExhausted:
		fallthrough
	case codes.FailedPrecondition:
		fallthrough
	case codes.Aborted:
		fallthrough
	case codes.OutOfRange:
		return Status_ClientError
	// ok
	case codes.OK:
		return Status_OK
	}

	return Status_Unknown
}

func GetGrpcHandlerName(fullMethod string) string {
	fmSl := strings.Split(fullMethod, "/")
	return fmSl[len(fmSl)-1]
}

func CalculateObservation(started time.Time) float64 {
	return float64(time.Since(started)) / float64(time.Second)
}
