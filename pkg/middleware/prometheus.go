package middleware

import (
	"net/http"
	"time"

	"github.com/pug-go/pug-template/pkg/promlib"
)

func Prometheus(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()

		rw := &rwWrapper{ResponseWriter: w}
		next.ServeHTTP(rw, r)

		status := promlib.HttpCodeToStatus(rw.status)
		pattern := popParam("pattern", w.Header())
		handler := "HTTP " + r.Method + ": " + pattern

		// pug_requests_total
		promlib.RequestsTotal.WithLabelValues(
			handler,
			"http",
			status,
		).Inc()

		// pug_response_time_seconds
		promlib.ResponseTime.WithLabelValues(
			handler,
			"http",
			status,
		).Observe(promlib.CalculateObservation(started))
	})
}

type rwWrapper struct {
	http.ResponseWriter
	status int
}

func (r *rwWrapper) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *rwWrapper) Write(b []byte) (int, error) {
	return r.ResponseWriter.Write(b)
}
