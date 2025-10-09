package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFailMiddlewareAlwaysFails(t *testing.T) {
	config := FailConfig{Status: 503, Body: "fail"}
	mw := FailMiddleware(config)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if _, err := w.Write([]byte("should not reach here")); err != nil {
			t.Errorf("write: %v", err)
		}
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != 503 {
		t.Errorf("expected status 503, got %d", rec.Code)
	}
	if rec.Body.String() != "fail" {
		t.Errorf("expected body 'fail', got '%s'", rec.Body.String())
	}
}

func TestFailMiddlewareNeverFails(t *testing.T) {
	config := FailConfig{Status: 200, Body: "ok"}
	mw := FailMiddleware(config)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(418)
		if _, err := w.Write([]byte("should not reach here")); err != nil {
			t.Errorf("write: %v", err)
		}
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("expected body 'ok', got '%s'", rec.Body.String())
	}
}
