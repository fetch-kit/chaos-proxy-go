package middleware

import (
	"math/rand"
	"net/http"
	"sync"
)

// dropConnectionRNG is the RNG used by DropConnectionMiddleware. Tests can override it.
var dropConnectionRNG *rand.Rand

// DropConnectionConfig is the configuration for the DropConnection middleware.
type DropConnectionConfig struct {
	Prob float64 `yaml:"prob"`
	Seed *int64  `yaml:"seed"`
}

// DropConnectionMiddleware returns a middleware that randomly drops connections.
func DropConnectionMiddleware(config DropConnectionConfig) func(http.Handler) http.Handler {
	prob := config.Prob
	if prob == 0 {
		prob = 1.0
	}
	var rng *rand.Rand
	if config.Seed != nil {
		rng = rand.New(rand.NewSource(*config.Seed))
	} else {
		rng = dropConnectionRNG
	}
	var rngMu sync.Mutex
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var v float64
			if rng != nil {
				rngMu.Lock()
				v = rng.Float64()
				rngMu.Unlock()
			} else {
				v = rand.Float64()
			}
			if v < prob {
				if hj, ok := w.(http.Hijacker); ok {
					conn, _, err := hj.Hijack()
					if err == nil {
						_ = conn.Close()
						return
					}
				}
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
