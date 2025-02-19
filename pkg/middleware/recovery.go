package middleware

import (
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec != nil {
				err := errors.New("panic: " + rec.(string))
				log.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("internal server error"))
			}
		}()

		next.ServeHTTP(w, r)
	})
}
