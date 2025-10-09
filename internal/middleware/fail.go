package middleware

import (
	"net/http"
)

// FailConfig is the configuration for the Fail middleware.
type FailConfig struct {
	Status int    `yaml:"status"`
	Body   string `yaml:"body"`
}

// FailMiddleware returns a middleware that always fails with the given status and body.
func FailMiddleware(config FailConfig) func(http.Handler) http.Handler {
	status := config.Status
	if status == 0 {
		status = 503
	}
	body := config.Body
	if body == "" {
		body = "failed by chaos-proxy-go"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		})
	}
}
