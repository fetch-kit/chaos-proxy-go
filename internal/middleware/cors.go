package middleware

import (
	"net/http"
)

// CorsConfig is the configuration for the CORS middleware.
type CorsConfig struct {
	Origin  string `yaml:"origin"`
	Methods string `yaml:"methods"`
	Headers string `yaml:"headers"`
}

// CorsMiddleware returns a middleware that sets CORS headers.
func CorsMiddleware(conf CorsConfig) func(http.Handler) http.Handler {
	origin := conf.Origin
	if origin == "" {
		origin = "*"
	}
	methods := conf.Methods
	if methods == "" {
		methods = "GET,POST,PUT,DELETE,OPTIONS"
	}
	headers := conf.Headers
	if headers == "" {
		headers = "Content-Type,Authorization"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", methods)
			w.Header().Set("Access-Control-Allow-Headers", headers)
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
