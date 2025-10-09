package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBodyTransformJSONMiddleware_RequestSetDelete(t *testing.T) {
	cfg := BodyTransformJSONConfig{
		Request: &BodyOps{
			Set:    map[string]interface{}{"foo": 123, "bar": "baz"},
			Delete: []string{"removeMe"},
		},
	}
	mw := BodyTransformJSONMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			t.Errorf("decode: %v", err)
		}
		if err := json.NewEncoder(w).Encode(m); err != nil {
			t.Errorf("encode: %v", err)
		}
	}))
	input := map[string]interface{}{
		"bar":      "old",
		"removeMe": true,
	}
	body, _ := json.Marshal(input)
	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var out map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["foo"] != float64(123) {
		t.Errorf("expected foo=123, got %v", out["foo"])
	}
	if out["bar"] != "baz" {
		t.Errorf("expected bar='baz', got %v", out["bar"])
	}
	if _, ok := out["removeMe"]; ok {
		t.Errorf("expected removeMe to be removed")
	}
}

func TestBodyTransformJSONMiddleware_EmptyBody(t *testing.T) {
	cfg := BodyTransformJSONConfig{
		Request: &BodyOps{Set: map[string]interface{}{"foo": 1}},
	}
	mw := BodyTransformJSONMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			t.Errorf("decode: %v", err)
		}
		if err := json.NewEncoder(w).Encode(m); err != nil {
			t.Errorf("encode: %v", err)
		}
	}))
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var out map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["foo"] != float64(1) {
		t.Errorf("expected foo=1, got %v", out["foo"])
	}
}

func TestBodyTransformJSONMiddleware_InvalidJSON(t *testing.T) {
	cfg := BodyTransformJSONConfig{
		Request: &BodyOps{Set: map[string]interface{}{"foo": 1}},
	}
	mw := BodyTransformJSONMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Errorf("expected 400 for invalid JSON, got %d", rec.Code)
	}
}

func TestBodyTransformJSONMiddleware_NonJSONContentType(t *testing.T) {
	cfg := BodyTransformJSONConfig{
		Request: &BodyOps{Set: map[string]interface{}{"foo": 1}},
	}
	mw := BodyTransformJSONMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	}))
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Errorf("expected 201 for non-JSON, got %d", rec.Code)
	}
}

func TestBodyTransformJSONMiddleware_NoOpConfig(t *testing.T) {
	cfg := BodyTransformJSONConfig{}
	mw := BodyTransformJSONMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			t.Errorf("decode: %v", err)
		}
		if err := json.NewEncoder(w).Encode(m); err != nil {
			t.Errorf("encode: %v", err)
		}
	}))
	input := map[string]interface{}{"foo": 1}
	body, _ := json.Marshal(input)
	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var out map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["foo"] != float64(1) {
		t.Errorf("expected foo=1, got %v", out["foo"])
	}
}

func TestBodyTransformJSONMiddleware_RemoveNonExistentField(t *testing.T) {
	cfg := BodyTransformJSONConfig{
		Request: &BodyOps{Delete: []string{"notThere"}},
	}
	mw := BodyTransformJSONMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			t.Errorf("decode: %v", err)
		}
		if err := json.NewEncoder(w).Encode(m); err != nil {
			t.Errorf("encode: %v", err)
		}
	}))
	input := map[string]interface{}{"foo": 1}
	body, _ := json.Marshal(input)
	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var out map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["foo"] != float64(1) {
		t.Errorf("expected foo=1, got %v", out["foo"])
	}
}

func TestBodyTransformJSONMiddleware_SetNonExistentField(t *testing.T) {
	cfg := BodyTransformJSONConfig{
		Request: &BodyOps{Set: map[string]interface{}{"bar": 2}},
	}
	mw := BodyTransformJSONMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			t.Errorf("decode: %v", err)
		}
		if err := json.NewEncoder(w).Encode(m); err != nil {
			t.Errorf("encode: %v", err)
		}
	}))
	input := map[string]interface{}{"foo": 1}
	body, _ := json.Marshal(input)
	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var out map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["bar"] != float64(2) {
		t.Errorf("expected bar=2, got %v", out["bar"])
	}
}

func TestBodyTransformJSONMiddleware_SetExistingField(t *testing.T) {
	cfg := BodyTransformJSONConfig{
		Request: &BodyOps{Set: map[string]interface{}{"foo": 2}},
	}
	mw := BodyTransformJSONMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			t.Errorf("decode: %v", err)
		}
		if err := json.NewEncoder(w).Encode(m); err != nil {
			t.Errorf("encode: %v", err)
		}
	}))
	input := map[string]interface{}{"foo": 1}
	body, _ := json.Marshal(input)
	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var out map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["foo"] != float64(2) {
		t.Errorf("expected foo=2 (overwritten), got %v", out["foo"])
	}
}

func TestBodyTransformJSONMiddleware_ResponseSetDelete(t *testing.T) {
	cfg := BodyTransformJSONConfig{
		Response: &BodyOps{
			Set:    map[string]interface{}{"foo": 42},
			Delete: []string{"removeMe"},
		},
	}
	mw := BodyTransformJSONMiddleware(cfg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"bar":1, "removeMe":true}`)); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var out map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["foo"] != float64(42) {
		t.Errorf("expected foo=42 in response, got %v", out["foo"])
	}
	if _, ok := out["removeMe"]; ok {
		t.Errorf("expected removeMe to be deleted in response")
	}
}
