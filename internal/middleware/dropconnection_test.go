package middleware

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDropConnectionMiddlewareProbability(t *testing.T) {
	config := DropConnectionConfig{Prob: 0.5}
	localRand := rand.New(rand.NewSource(42))
	dropConnectionRNG = localRand
	defer func() { dropConnectionRNG = nil }()

	dropped := 0
	notDropped := 0
	total := 1000
	for i := 0; i < total; i++ {
		called := false
		mw := DropConnectionMiddleware(config)
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if called {
			notDropped++
		} else {
			dropped++
		}
	}
	if dropped < 400 || dropped > 600 {
		t.Errorf("expected ~500 drops, got %d", dropped)
	}
	if notDropped < 400 || notDropped > 600 {
		t.Errorf("expected ~500 passes, got %d", notDropped)
	}
}

func TestDropConnectionMiddlewareDoesNotCallNext(t *testing.T) {
	mw := DropConnectionMiddleware(DropConnectionConfig{})
	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called {
		t.Error("next handler should not be called when connection is dropped")
	}
}
