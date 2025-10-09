package middleware

import (
	"math/rand"
	"net/http"
)

// FailRandomlyConfig is the configuration for the FailRandomly middleware.
type FailRandomlyConfig struct {
	Rate   float64 `yaml:"rate"`
	Status int     `yaml:"status"`
	Body   string  `yaml:"body"`
}

// failRandomlyRNG is the RNG used by FailRandomlyMiddleware. Tests can override it.
var failRandomlyRNG *rand.Rand

// FailRandomlyMiddleware returns a middleware that fails requests randomly.
func FailRandomlyMiddleware(conf FailRandomlyConfig) func(http.Handler) http.Handler {
	status := conf.Status
	if status == 0 {
		status = 503
	}
	body := conf.Body
	if body == "" {
		body = "failed by chaos-proxy-go"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			var v float64
			if failRandomlyRNG != nil {
				v = failRandomlyRNG.Float64()
			} else {
				v = rand.Float64()
			}
			if v < conf.Rate {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(body))
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}
