package middleware

import "net/http"

var Default = []func(http.Handler) http.Handler{
	Prometheus,
	Recovery,
}

func New(middlewares ...func(http.Handler) http.Handler) []func(http.Handler) http.Handler {
	return middlewares
}

func popParam(key string, header http.Header) string {
	result := header.Get(key)
	header.Del(key)
	return result
}
