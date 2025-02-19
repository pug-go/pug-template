package middleware

import (
	"encoding/json"
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec != nil {
				err := errors.New("panic: " + rec.(string))
				log.Error(err)

				resp := struct {
					Code    uint32   `json:"code"`
					Message string   `json:"message"`
					Details []string `json:"details"`
				}{
					Code:    uint32(codes.Internal),
					Message: codes.Internal.String(),
					Details: []string{},
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
