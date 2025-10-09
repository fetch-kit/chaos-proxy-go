package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware_Defaults(t *testing.T) {
	mw := CorsMiddleware(CorsConfig{})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected default origin '*', got '%s'", rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if rec.Header().Get("Access-Control-Allow-Methods") != "GET,POST,PUT,DELETE,OPTIONS" {
		t.Errorf("expected default methods, got '%s'", rec.Header().Get("Access-Control-Allow-Methods"))
	}
	if rec.Header().Get("Access-Control-Allow-Headers") != "Content-Type,Authorization" {
		t.Errorf("expected default headers, got '%s'", rec.Header().Get("Access-Control-Allow-Headers"))
	}
}

func TestCORSMiddleware_CustomConfig(t *testing.T) {
	mw := CorsMiddleware(CorsConfig{
		Origin:  "https://example.com",
		Methods: "GET,POST",
		Headers: "X-Custom,Authorization",
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("POST", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("expected custom origin, got '%s'", rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if rec.Header().Get("Access-Control-Allow-Methods") != "GET,POST" {
		t.Errorf("expected custom methods, got '%s'", rec.Header().Get("Access-Control-Allow-Methods"))
	}
	if rec.Header().Get("Access-Control-Allow-Headers") != "X-Custom,Authorization" {
		t.Errorf("expected custom headers, got '%s'", rec.Header().Get("Access-Control-Allow-Headers"))
	}
}

func TestCORSMiddleware_OptionsRequest(t *testing.T) {
	mw := CorsMiddleware(CorsConfig{Origin: "https://foo.bar"})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for OPTIONS request")
	}))

	req := httptest.NewRequest("OPTIONS", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 No Content, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://foo.bar" {
		t.Errorf("expected origin header for OPTIONS, got '%s'", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSMiddleware_NoOriginHeader(t *testing.T) {
	mw := CorsMiddleware(CorsConfig{Origin: ""})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected '*' for empty origin config, got '%s'", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}
