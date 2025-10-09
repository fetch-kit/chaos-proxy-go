package middleware

import (
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
