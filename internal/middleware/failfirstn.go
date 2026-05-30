package middleware

import (
	"net/http"
	"sync/atomic"
)

// FailFirstNConfig is the configuration for the FailFirstN middleware.
type FailFirstNConfig struct {
	N      int    `yaml:"n"`
	Status int    `yaml:"status"`
	Body   string `yaml:"body"`
}

// FailFirstNMiddleware returns a middleware that fails the first N requests, then always passes through.
func FailFirstNMiddleware(conf FailFirstNConfig) func(http.Handler) http.Handler {
	var count int64
	status := conf.Status
	if status == 0 {
		status = 503
	}
	body := conf.Body
	if body == "" {
		body = "failed by chaos-proxy-go"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := atomic.AddInt64(&count, 1)
			if c <= int64(conf.N) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(body))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
