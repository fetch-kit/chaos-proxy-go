package middleware

import (
	"net/http"
	"time"
)

// LatencyConfig is the configuration for the Latency middleware.
type LatencyConfig struct {
	Ms int `yaml:"ms"`
}

// LatencyMiddleware returns a middleware that adds a fixed delay to requests.
func LatencyMiddleware(config LatencyConfig) func(http.Handler) http.Handler {
	delay := time.Duration(config.Ms) * time.Millisecond

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-time.After(delay):
			case <-r.Context().Done():
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
