package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHeaderTransform_SetExistingHeader(t *testing.T) {
	cfg := HeaderTransformConfig{
		Request: &HeaderOps{Set: map[string]string{"X-Foo": "new"}},
	}
	mw := HeaderTransformMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Foo", r.Header.Get("X-Foo"))
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Foo", "old")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	resp := rec.Result()
	if resp.Header.Get("X-Foo") != "new" {
		t.Errorf("expected X-Foo=new (overwritten), got %v", resp.Header.Get("X-Foo"))
	}
}

func TestHeaderTransform_SetNonExistentHeader(t *testing.T) {
	cfg := HeaderTransformConfig{
		Request: &HeaderOps{Set: map[string]string{"X-Bar": "baz"}},
	}
	mw := HeaderTransformMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Bar", r.Header.Get("X-Bar"))
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	resp := rec.Result()
	if resp.Header.Get("X-Bar") != "baz" {
		t.Errorf("expected X-Bar=baz, got %v", resp.Header.Get("X-Bar"))
	}
}

func TestHeaderTransform_DeleteNonExistentHeader(t *testing.T) {
	cfg := HeaderTransformConfig{
		Request: &HeaderOps{Delete: []string{"X-NotThere"}},
	}
	mw := HeaderTransformMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	resp := rec.Result()
	// Should not panic or error, nothing to check
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHeaderTransform_EmptyHeaders(t *testing.T) {
	cfg := HeaderTransformConfig{
		Request: &HeaderOps{Set: map[string]string{"X-Foo": "bar"}},
	}
	mw := HeaderTransformMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Foo", r.Header.Get("X-Foo"))
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	resp := rec.Result()
	if resp.Header.Get("X-Foo") != "bar" {
		t.Errorf("expected X-Foo=bar, got %v", resp.Header.Get("X-Foo"))
	}
}

func TestHeaderTransform_CaseInsensitiveDelete(t *testing.T) {
	cfg := HeaderTransformConfig{
		Request: &HeaderOps{Delete: []string{"x-foo"}},
	}
	mw := HeaderTransformMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range r.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Foo", "bar")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	resp := rec.Result()
	if resp.Header.Get("X-Foo") != "" {
		t.Errorf("expected X-Foo to be deleted (case-insensitive)")
	}
}

func TestHeaderTransform_MultiValueHeader(t *testing.T) {
	cfg := HeaderTransformConfig{
		Request: &HeaderOps{Set: map[string]string{"X-Multi": "one"}},
	}
	mw := HeaderTransformMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, v := range r.Header["X-Multi"] {
			w.Header().Add("X-Multi", v)
		}
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Add("X-Multi", "zero")
	req.Header.Add("X-Multi", "two")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	resp := rec.Result()
	vals := resp.Header["X-Multi"]
	if len(vals) != 1 || vals[0] != "one" {
		t.Errorf("expected X-Multi=[one], got %v", vals)
	}
}

func TestHeaderTransform_RequestSetDelete(t *testing.T) {
	cfg := HeaderTransformConfig{
		Request: &HeaderOps{
			Set:    map[string]string{"X-Foo": "bar", "X-Bar": "baz"},
			Delete: []string{"X-RemoveMe"},
		},
	}
	mw := HeaderTransformMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range r.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Bar", "old")
	req.Header.Set("X-RemoveMe", "bye")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	resp := rec.Result()
	if resp.Header.Get("X-Foo") != "bar" {
		t.Errorf("expected X-Foo=bar, got %v", resp.Header.Get("X-Foo"))
	}
	if resp.Header.Get("X-Bar") != "baz" {
		t.Errorf("expected X-Bar=baz, got %v", resp.Header.Get("X-Bar"))
	}
	if resp.Header.Get("X-RemoveMe") != "" {
		t.Errorf("expected X-RemoveMe to be deleted")
	}
}

func TestHeaderTransform_ResponseSetDelete(t *testing.T) {
	cfg := HeaderTransformConfig{
		Response: &HeaderOps{
			Set:    map[string]string{"X-Foo": "resp", "X-Bar": "resp2"},
			Delete: []string{"X-RemoveMe"},
		},
	}
	mw := HeaderTransformMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Bar", "oldresp")
		w.Header().Set("X-RemoveMe", "bye")
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	resp := rec.Result()
	if resp.Header.Get("X-Foo") != "resp" {
		t.Errorf("expected X-Foo=resp, got %v", resp.Header.Get("X-Foo"))
	}
	if resp.Header.Get("X-Bar") != "resp2" {
		t.Errorf("expected X-Bar=resp2, got %v", resp.Header.Get("X-Bar"))
	}
	if resp.Header.Get("X-RemoveMe") != "" {
		t.Errorf("expected X-RemoveMe to be deleted in response")
	}
}

func TestHeaderTransform_NoOpConfig(t *testing.T) {
	cfg := HeaderTransformConfig{}
	mw := HeaderTransformMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Foo", "bar")
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	resp := rec.Result()
	if resp.Header.Get("X-Foo") != "bar" {
		t.Errorf("expected X-Foo=bar, got %v", resp.Header.Get("X-Foo"))
	}
}
