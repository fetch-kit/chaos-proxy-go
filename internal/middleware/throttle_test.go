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
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()
	start := time.Now()
	handler.ServeHTTP(rec, req)
	elapsed := time.Since(start)
	// 4096 bytes, burst 2048, rate 1024/sec, so expect at least 2s for last 2048 bytes
	if elapsed < 2*time.Second {
		t.Errorf("expected throttling delay >= 2s, got %v", elapsed)
	}
}

func TestThrottleMiddleware_KeyHeader(t *testing.T) {
	config := ThrottleConfig{Rate: 1024, Burst: 1024, ChunkSize: 1024, Key: "X-API-Key"}
	mw := ThrottleMiddleware(config)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(bytes.Repeat([]byte("b"), 2048)); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	// Two different keys should have independent throttling
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.Header.Set("X-API-Key", "abc")
	rec1 := httptest.NewRecorder()
	start1 := time.Now()
	handler.ServeHTTP(rec1, req1)
	elapsed1 := time.Since(start1)

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.Header.Set("X-API-Key", "def")
	rec2 := httptest.NewRecorder()
	start2 := time.Now()
	handler.ServeHTTP(rec2, req2)
	elapsed2 := time.Since(start2)

	if elapsed1 < 1*time.Second {
		t.Errorf("expected throttling delay >= 1s for key abc, got %v", elapsed1)
	}
	if elapsed2 < 1*time.Second {
		t.Errorf("expected throttling delay >= 1s for key def, got %v", elapsed2)
	}
}

func TestThrottleMiddleware_KeyFallbackToRemoteAddr(t *testing.T) {
	config := ThrottleConfig{Rate: 1024, Burst: 1024, ChunkSize: 1024, Key: "X-API-Key"}
	mw := ThrottleMiddleware(config)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(bytes.Repeat([]byte("c"), 2048)); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "9.8.7.6:4321"
	// No X-API-Key header, should fallback to RemoteAddr
	rec := httptest.NewRecorder()
	start := time.Now()
	handler.ServeHTTP(rec, req)
	elapsed := time.Since(start)
	if elapsed < 1*time.Second {
		t.Errorf("expected throttling delay >= 1s for fallback, got %v", elapsed)
	}
}
