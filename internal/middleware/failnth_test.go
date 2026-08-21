package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFailNthMiddlewareFailsOnNthRequest(t *testing.T) {
	config := FailNthConfig{N: 3, Status: 502, Body: "failnth"}
	mw := FailNthMiddleware(config)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if _, err := w.Write([]byte("ok")); err != nil {
			t.Errorf("write: %v", err)
		}
	}))

	for i := 1; i <= 6; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if i%3 == 0 {
			if rec.Code != 502 {
				t.Errorf("expected status 502 on request %d, got %d", i, rec.Code)
			}
			if rec.Body.String() != "failnth" {
				t.Errorf("expected body 'failnth' on request %d, got '%s'", i, rec.Body.String())
			}
		} else {
			if rec.Code != 200 {
				t.Errorf("expected status 200 on request %d, got %d", i, rec.Code)
			}
			if rec.Body.String() != "ok" {
				t.Errorf("expected body 'ok' on request %d, got '%s'", i, rec.Body.String())
			}
		}
	}
}

func TestFailNthMiddlewareNonPositiveNPassesThrough(t *testing.T) {
	for _, n := range []int{0, -1} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			calls := 0
			handler := FailNthMiddleware(FailNthConfig{N: n})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++
				w.WriteHeader(http.StatusNoContent)
			}))

			for range 2 {
				recorder := httptest.NewRecorder()
				handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
				if recorder.Code != http.StatusNoContent {
					t.Fatalf("got status %d, want %d", recorder.Code, http.StatusNoContent)
				}
			}
			if calls != 2 {
				t.Fatalf("downstream called %d times, want 2", calls)
			}
		})
	}
}
