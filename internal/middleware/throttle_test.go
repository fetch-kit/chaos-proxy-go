package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestThrottleMiddleware_BurstAndRate(t *testing.T) {
	config := ThrottleConfig{Rate: 1024, Burst: 2048, ChunkSize: 1024}
	mw := ThrottleMiddleware(config)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(bytes.Repeat([]byte("a"), 4096)); err != nil {
			t.Errorf("write: %v", err)
		}
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	start := time.Now()
	handler.ServeHTTP(rec, req)
	elapsed := time.Since(start)
	// 4096 bytes, burst 2048, rate 1024/sec, so expect at least 2s for last 2048 bytes
	if elapsed < 2*time.Second {
		t.Errorf("expected throttling delay >= 2s, got %v", elapsed)
	}
}

func TestThrottleMiddleware_BurstResetsPerRequest(t *testing.T) {
	config := ThrottleConfig{Rate: 1024, Burst: 2048, ChunkSize: 1024}
	mw := ThrottleMiddleware(config)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(bytes.Repeat([]byte("a"), 4096)); err != nil {
			t.Errorf("write: %v", err)
		}
	}))

	// Run two sequential requests — each should get a fresh burst allowance
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		start := time.Now()
		handler.ServeHTTP(rec, req)
		elapsed := time.Since(start)
		// Each request: 4096 bytes, burst 2048 free, 2048 throttled at 1024/sec = ~2s
		if elapsed < 2*time.Second {
			t.Errorf("request %d: expected throttling delay >= 2s, got %v (burst not reset?)", i+1, elapsed)
		}
		// Should not take more than ~3s (burst is being applied correctly)
		if elapsed > 3*time.Second {
			t.Errorf("request %d: took too long (%v), burst may not be applying", i+1, elapsed)
		}
	}
}

func TestThrottleMiddleware_NoRatePassthrough(t *testing.T) {
	config := ThrottleConfig{Rate: 0}
	mw := ThrottleMiddleware(config)
	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if !called {
		t.Error("expected handler to be called when rate=0")
	}
}
