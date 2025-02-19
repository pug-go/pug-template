package middleware

import (
	"encoding/json"
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

				// TODO: should be grpc-like response
				resp := struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				}{
					Code:    http.StatusInternalServerError,
					Message: "internal server error",
				}

				w.WriteHeader(http.StatusInternalServerError)
				if err = json.NewEncoder(w).Encode(resp); err != nil {
					log.Error(err)
				}
			}
		}()

		next.ServeHTTP(w, r)
	})
}
