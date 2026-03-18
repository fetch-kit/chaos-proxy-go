package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLatencyMiddlewareAddsDelay(t *testing.T) {
	config := LatencyConfig{Ms: 100}
	mw := LatencyMiddleware(config)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	start := time.Now()
	handler.ServeHTTP(rec, req)
	elapsed := time.Since(start)

	if elapsed < 100*time.Millisecond {
		t.Errorf("expected at least 100ms delay, got %v", elapsed)
	}
}

func TestLatencyMiddlewareClientAbort(t *testing.T) {
	config := LatencyConfig{Ms: 500}
	mw := LatencyMiddleware(config)

	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	// Cancel the context after a short time to simulate client abort
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	handler.ServeHTTP(rec, req)
	elapsed := time.Since(start)

	if called {
		t.Error("expected upstream handler NOT to be called after client abort")
	}
	if elapsed >= 500*time.Millisecond {
		t.Errorf("expected early exit on client abort, but took %v", elapsed)
	}
}
