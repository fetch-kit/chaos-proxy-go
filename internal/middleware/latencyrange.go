package middleware

import (
	"math/rand"
	"net/http"
	"time"
)

// latencyRangeRNG is the RNG used by LatencyRangeMiddleware. Tests can override it.
var latencyRangeRNG *rand.Rand

// LatencyRangeConfig is the configuration for the LatencyRange middleware.
type LatencyRangeConfig struct {
	MinMs int `yaml:"minMs"`
	MaxMs int `yaml:"maxMs"`
}

// LatencyRangeMiddleware returns a middleware that adds a random delay in a range.
func LatencyRangeMiddleware(conf LatencyRangeConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			delay := conf.MinMs
			var n int
			if conf.MaxMs > conf.MinMs {
				if latencyRangeRNG != nil {
					n = latencyRangeRNG.Intn(conf.MaxMs - conf.MinMs + 1)
				} else {
					n = rand.Intn(conf.MaxMs - conf.MinMs + 1)
				}
				delay += n
			}
			time.Sleep(time.Duration(delay) * time.Millisecond)
			next.ServeHTTP(w, r)
		})
	}
}
