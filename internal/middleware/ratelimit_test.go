package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimitMiddleware_BasicLimit(t *testing.T) {
	config := RateLimitConfig{Limit: 3, WindowMs: 1000}
	mw := RateLimitMiddleware(config)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Errorf("expected 200, got %d on request %d", rec.Code, i+1)
		}
	}
	// 4th request should be rate-limited
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 429 {
		t.Errorf("expected 429, got %d on 4th request", rec.Code)
	}
}

func TestRateLimitMiddleware_WindowReset(t *testing.T) {
	config := RateLimitConfig{Limit: 2, WindowMs: 100}
	mw := RateLimitMiddleware(config)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "5.6.7.8:1234"
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Errorf("expected 200, got %d on request %d", rec.Code, i+1)
		}
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 429 {
		t.Errorf("expected 429 before window reset, got %d", rec.Code)
	}
	time.Sleep(120 * time.Millisecond)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("expected 200 after window reset, got %d", rec.Code)
	}
}

func TestRateLimitMiddleware_KeyHeader(t *testing.T) {
	config := RateLimitConfig{Limit: 2, WindowMs: 1000, Key: "X-API-Key"}
	mw := RateLimitMiddleware(config)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	// Two different keys should have independent limits
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.Header.Set("X-API-Key", "abc")
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.Header.Set("X-API-Key", "def")
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req1)
		if rec.Code != 200 {
			t.Errorf("expected 200 for key abc, got %d on request %d", rec.Code, i+1)
		}
		rec2 := httptest.NewRecorder()
		handler.ServeHTTP(rec2, req2)
		if rec2.Code != 200 {
			t.Errorf("expected 200 for key def, got %d on request %d", rec2.Code, i+1)
		}
	}
	// 3rd request for key abc should be rate-limited
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req1)
	if rec.Code != 429 {
		t.Errorf("expected 429 for key abc, got %d", rec.Code)
	}
	// 3rd request for key def should be rate-limited
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != 429 {
		t.Errorf("expected 429 for key def, got %d", rec2.Code)
	}
}

func TestRateLimitMiddleware_KeyFallbackToRemoteAddr(t *testing.T) {
	config := RateLimitConfig{Limit: 1, WindowMs: 1000, Key: "X-API-Key"}
	mw := RateLimitMiddleware(config)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "9.8.7.6:4321"
	// No X-API-Key header, should fallback to RemoteAddr
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("expected 200 for fallback, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 429 {
		t.Errorf("expected 429 for fallback, got %d", rec.Code)
	}
}
