package middleware

import (
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// latencyRangeRNG is the RNG used by LatencyRangeMiddleware. Tests can override it.
var latencyRangeRNG *rand.Rand

// LatencyRangeConfig is the configuration for the LatencyRange middleware.
type LatencyRangeConfig struct {
	MinMs int    `yaml:"minMs"`
	MaxMs int    `yaml:"maxMs"`
	Seed  *int64 `yaml:"seed"`
}

func sampleLatencyDelayMs(conf LatencyRangeConfig, rng *rand.Rand, rngMu *sync.Mutex) int {
	delay := conf.MinMs
	if conf.MaxMs > conf.MinMs {
		var n int
		if rng != nil {
			rngMu.Lock()
			n = rng.Intn(conf.MaxMs - conf.MinMs + 1)
			rngMu.Unlock()
		} else {
			n = rand.Intn(conf.MaxMs - conf.MinMs + 1)
		}
		delay += n
	}
	return delay
}

// LatencyRangeMiddleware returns a middleware that adds a random delay in a range.
func LatencyRangeMiddleware(conf LatencyRangeConfig) func(http.Handler) http.Handler {
	var rng *rand.Rand
	if conf.Seed != nil {
		rng = rand.New(rand.NewSource(*conf.Seed))
	} else {
		rng = latencyRangeRNG
	}
	var rngMu sync.Mutex
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			delay := sampleLatencyDelayMs(conf, rng, &rngMu)
			select {
			case <-time.After(time.Duration(delay) * time.Millisecond):
			case <-r.Context().Done():
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
