package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFailFirstNMiddlewareFailsFirstN(t *testing.T) {
	config := FailFirstNConfig{N: 2, Status: 429, Body: "too early"}
	mw := FailFirstNMiddleware(config)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if _, err := w.Write([]byte("ok")); err != nil {
			t.Errorf("write: %v", err)
		}
	}))

	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if i <= 2 {
			if rec.Code != 429 {
				t.Errorf("request %d: expected status 429, got %d", i, rec.Code)
			}
			if rec.Body.String() != "too early" {
				t.Errorf("request %d: expected body 'too early', got '%s'", i, rec.Body.String())
			}
		} else {
			if rec.Code != 200 {
				t.Errorf("request %d: expected status 200, got %d", i, rec.Code)
			}
			if rec.Body.String() != "ok" {
				t.Errorf("request %d: expected body 'ok', got '%s'", i, rec.Body.String())
			}
		}
	}
}

func TestFailFirstNMiddlewareDefaults(t *testing.T) {
	config := FailFirstNConfig{N: 1}
	mw := FailFirstNMiddleware(config)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if _, err := w.Write([]byte("ok")); err != nil {
			t.Errorf("write: %v", err)
		}
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 503 {
		t.Errorf("expected status 503, got %d", rec.Code)
	}
	if rec.Body.String() != "failed by chaos-proxy-go" {
		t.Errorf("expected body 'failed by chaos-proxy-go', got '%s'", rec.Body.String())
	}
}

func TestFailFirstNMiddlewareInstanceIsolation(t *testing.T) {
	configA := FailFirstNConfig{N: 1, Status: 418, Body: "a"}
	configB := FailFirstNConfig{N: 1, Status: 409, Body: "b"}

	mwA := FailFirstNMiddleware(configA)
	mwB := FailFirstNMiddleware(configB)

	handlerA := mwA(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if _, err := w.Write([]byte("a ok")); err != nil {
			t.Errorf("write: %v", err)
		}
	}))

	handlerB := mwB(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if _, err := w.Write([]byte("b ok")); err != nil {
			t.Errorf("write: %v", err)
		}
	}))

	// First request to A should fail
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handlerA.ServeHTTP(rec, req)
	if rec.Code != 418 {
		t.Errorf("A request 1: expected 418, got %d", rec.Code)
	}

	// First request to B should independently fail
	req = httptest.NewRequest("GET", "/", nil)
	rec = httptest.NewRecorder()
	handlerB.ServeHTTP(rec, req)
	if rec.Code != 409 {
		t.Errorf("B request 1: expected 409, got %d", rec.Code)
	}

	// Second request to A should pass through
	req = httptest.NewRequest("GET", "/", nil)
	rec = httptest.NewRecorder()
	handlerA.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("A request 2: expected 200, got %d", rec.Code)
	}

	// Second request to B should pass through
	req = httptest.NewRequest("GET", "/", nil)
	rec = httptest.NewRecorder()
	handlerB.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("B request 2: expected 200, got %d", rec.Code)
	}
}
