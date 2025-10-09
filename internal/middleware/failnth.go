package middleware

import (
	"net/http"
	"sync/atomic"
)

// FailNthConfig is the configuration for the FailNth middleware.
type FailNthConfig struct {
	N      int    `yaml:"n"`
	Status int    `yaml:"status"`
	Body   string `yaml:"body"`
}

// FailNthMiddleware returns a middleware that fails every Nth request.
func FailNthMiddleware(conf FailNthConfig) func(http.Handler) http.Handler {
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
			if int(c)%conf.N == 0 {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(body))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
