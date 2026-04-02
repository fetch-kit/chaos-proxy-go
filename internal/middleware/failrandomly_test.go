package middleware

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFailRandomlyMiddlewareDeterministic(t *testing.T) {
	config := FailRandomlyConfig{Rate: 0.5, Status: 504, Body: "failrandomly"}
	localRand := rand.New(rand.NewSource(42))
	failRandomlyRNG = localRand
	defer func() { failRandomlyRNG = nil }()
	mw := FailRandomlyMiddleware(config)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if _, err := w.Write([]byte("ok")); err != nil {
			t.Errorf("write: %v", err)
		}
	}))

	failures := 0
	successes := 0
	total := 1000
	for i := 0; i < total; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code == 504 && rec.Body.String() == "failrandomly" {
			failures++
		} else if rec.Code == 200 && rec.Body.String() == "ok" {
			successes++
		} else {
			t.Errorf("unexpected response: code=%d, body=%q", rec.Code, rec.Body.String())
		}
	}
	if failures < 400 || failures > 600 {
		t.Errorf("expected ~500 failures, got %d", failures)
	}
	if successes < 400 || successes > 600 {
		t.Errorf("expected ~500 successes, got %d", successes)
	}
}

func TestFailRandomlyMiddlewareSeedDeterministic(t *testing.T) {
	seed := int64(123)
	config := FailRandomlyConfig{Rate: 0.5, Status: 504, Body: "failrandomly", Seed: &seed}

	mwA := FailRandomlyMiddleware(config)
	mwB := FailRandomlyMiddleware(config)

	handlerA := mwA(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))
	handlerB := mwB(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))

	for i := 0; i < 100; i++ {
		reqA := httptest.NewRequest("GET", "/", nil)
		reqB := httptest.NewRequest("GET", "/", nil)
		recA := httptest.NewRecorder()
		recB := httptest.NewRecorder()

		handlerA.ServeHTTP(recA, reqA)
		handlerB.ServeHTTP(recB, reqB)

		if recA.Code != recB.Code || recA.Body.String() != recB.Body.String() {
			t.Fatalf("seeded middleware diverged at request %d: A=(%d,%q) B=(%d,%q)", i, recA.Code, recA.Body.String(), recB.Code, recB.Body.String())
		}
	}
}
