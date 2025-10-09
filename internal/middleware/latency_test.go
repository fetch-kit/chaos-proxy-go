package middleware

import (
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
