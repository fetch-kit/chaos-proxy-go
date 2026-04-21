package telemetry

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"chaos-proxy-go/internal/config"
)

func testOtelCfg() config.OtelConfig {
	return config.OtelConfig{
		ServiceName:     "test-svc",
		Endpoint:        "http://localhost:4318",
		FlushIntervalMs: 60000,
		MaxBatchSize:    100,
		MaxQueueSize:    1000,
	}
}

func TestMiddleware_PropagatesExistingTraceparent(t *testing.T) {
	cfg := testOtelCfg()
	exporter := NewExporter(cfg)
	defer exporter.Shutdown()

	mw := NewMiddleware(cfg, exporter)

	var capturedTraceID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tc := ExtractTraceContext(r)
		if tc != nil {
			capturedTraceID = tc.TraceID
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := mw(inner)
	r := httptest.NewRequest("GET", "/api/test", nil)
	r.Header.Set("Traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if capturedTraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("expected trace ID to be propagated, got %q", capturedTraceID)
	}
}

func TestMiddleware_GeneratesTraceWhenAbsent(t *testing.T) {
	cfg := testOtelCfg()
	exporter := NewExporter(cfg)
	defer exporter.Shutdown()

	mw := NewMiddleware(cfg, exporter)

	var capturedTraceID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tc := ExtractTraceContext(r)
		if tc != nil {
			capturedTraceID = tc.TraceID
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := mw(inner)
	r := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if len(capturedTraceID) != 32 {
		t.Errorf("expected a generated trace ID (32 chars), got %q", capturedTraceID)
	}
}

func TestMiddleware_CapturesStatusCode(t *testing.T) {
	cfg := testOtelCfg()
	exporter := NewExporter(cfg)
	defer exporter.Shutdown()

	mw := NewMiddleware(cfg, exporter)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	handler := mw(inner)
	r := httptest.NewRequest("GET", "/missing", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	// Give the exporter a moment to receive the span (AddSpan is synchronous)
	exporter.mu.Lock()
	q := exporter.queue
	exporter.mu.Unlock()

	// Span should be in queue with correct status
	if len(q) == 0 {
		t.Fatal("expected span in queue")
	}
	span := q[len(q)-1]
	if span.Status != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", span.Status)
	}
	if !span.Error {
		t.Error("expected error=true for 4xx status")
	}
}

func TestMiddleware_MarksErrorFor5xx(t *testing.T) {
	cfg := testOtelCfg()
	exporter := NewExporter(cfg)
	defer exporter.Shutdown()

	mw := NewMiddleware(cfg, exporter)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	handler := mw(inner)
	r := httptest.NewRequest("POST", "/checkout", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	exporter.mu.Lock()
	q := exporter.queue
	exporter.mu.Unlock()

	if len(q) == 0 {
		t.Fatal("expected span in queue")
	}
	span := q[len(q)-1]
	if !span.Error {
		t.Error("expected error=true for 503")
	}
}
