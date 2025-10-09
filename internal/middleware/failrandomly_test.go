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
