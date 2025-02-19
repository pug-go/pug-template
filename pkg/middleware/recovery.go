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
				err := errors.New(rec.(string))
				log.Error(err)

				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_, err = w.Write([]byte(`{"code": 13, "message": "Internal"}`))
				if err != nil {
					log.Error(err)
				}
			}
		}()

		next.ServeHTTP(w, r)
	})
}
