package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type dummyConfig struct {
	Value string
}

func dummyMiddleware(cfg dummyConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Dummy", cfg.Value)
			next.ServeHTTP(w, r)
		})
	}
}

func TestRegistry_RegisterAndCreate(t *testing.T) {
	// Register dummy middleware on DefaultRegistry for test
	DefaultRegistry.factories["dummy"] = func(cfg any) (func(http.Handler) http.Handler, error) {
		c, ok := cfg.(dummyConfig)
		if !ok {
			return nil, errors.New("bad config type")
		}
		return dummyMiddleware(c), nil
	}

	mw, err := DefaultRegistry.Create("dummy", dummyConfig{Value: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if req.Header.Get("X-Dummy") != "test" {
		t.Errorf("expected X-Dummy header to be set, got %v", req.Header.Get("X-Dummy"))
	}
}

func TestRegistry_CreateUnknown(t *testing.T) {
	_, err := DefaultRegistry.Create("notfound", struct{}{})
	if err == nil {
		t.Error("expected error for unknown middleware name")
	}
}

func TestRegistry_CreateBadConfigType(t *testing.T) {
	DefaultRegistry.factories["dummy"] = func(cfg any) (func(http.Handler) http.Handler, error) {
		c, ok := cfg.(dummyConfig)
		if !ok {
			return nil, errors.New("bad config type")
		}
		return dummyMiddleware(c), nil
	}
	_, err := DefaultRegistry.Create("dummy", struct{}{})
	if err == nil {
		t.Error("expected error for bad config type")
	}
}
