package middleware

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLatencyRangeMiddlewareDeterministic(t *testing.T) {
	config := LatencyRangeConfig{MinMs: 50, MaxMs: 150}
	localRand := rand.New(rand.NewSource(42))
	latencyRangeRNG = localRand
	defer func() { latencyRangeRNG = nil }()
	mw := LatencyRangeMiddleware(config)

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		start := time.Now()
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))
		handler.ServeHTTP(rec, req)
		elapsed := time.Since(start)
		if elapsed < 50*time.Millisecond {
			t.Errorf("delay too short: %v", elapsed)
		}
		if elapsed > 160*time.Millisecond {
			t.Errorf("delay too long: %v", elapsed)
		}
	}
}
