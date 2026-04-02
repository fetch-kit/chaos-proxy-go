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

func TestDropConnectionMiddlewareSeedDeterministic(t *testing.T) {
	seed := int64(456)
	config := DropConnectionConfig{Prob: 0.5, Seed: &seed}

	mwA := DropConnectionMiddleware(config)
	mwB := DropConnectionMiddleware(config)

	for i := 0; i < 100; i++ {
		calledA := false
		calledB := false

		handlerA := mwA(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calledA = true
		}))
		handlerB := mwB(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calledB = true
		}))

		reqA := httptest.NewRequest("GET", "/", nil)
		reqB := httptest.NewRequest("GET", "/", nil)
		recA := httptest.NewRecorder()
		recB := httptest.NewRecorder()

		handlerA.ServeHTTP(recA, reqA)
		handlerB.ServeHTTP(recB, reqB)

		if calledA != calledB {
			t.Fatalf("seeded middleware diverged at request %d: A called=%v, B called=%v", i, calledA, calledB)
		}
	}
}
