package telemetry

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"chaos-proxy-go/internal/config"
)

func minimalOtelCfg(endpoint string) config.OtelConfig {
	return config.OtelConfig{
		ServiceName:     "test-svc",
		Endpoint:        endpoint,
		FlushIntervalMs: 60000, // very long — prevent auto-flush during test
		MaxBatchSize:    100,
		MaxQueueSize:    1000,
	}
}

func TestExporter_ShutdownFlushesSpans(t *testing.T) {
	var mu sync.Mutex
	var received []map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("invalid JSON: %v", err)
			return
		}
		mu.Lock()
		received = append(received, payload)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := minimalOtelCfg(srv.URL)
	e := NewExporter(cfg)

	span := &Span{TraceID: "t1", SpanID: "s1", StartTime: time.Now(), EndTime: time.Now(), ServiceName: "test-svc"}
	e.AddSpan(span)
	e.Shutdown()

	mu.Lock()
	n := len(received)
	mu.Unlock()

	if n == 0 {
		t.Error("expected at least one export on shutdown")
	}
}

func TestExporter_FlushOnBatchSize(t *testing.T) {
	flushed := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case flushed <- struct{}{}:
		default:
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := minimalOtelCfg(srv.URL)
	cfg.MaxBatchSize = 3
	e := NewExporter(cfg)
	defer e.Shutdown()

	for i := 0; i < 3; i++ {
		e.AddSpan(&Span{TraceID: "t", SpanID: "s", StartTime: time.Now(), EndTime: time.Now(), ServiceName: "svc"})
	}

	select {
	case <-flushed:
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for batch flush")
	}
}

func TestExporter_QueueOverflowDropsOldest(t *testing.T) {
	cfg := minimalOtelCfg("http://127.0.0.1:0") // unreachable; we won't actually flush
	cfg.MaxQueueSize = 2
	cfg.MaxBatchSize = 100
	e := NewExporter(cfg)
	defer e.Shutdown()

	span := func(id string) *Span {
		return &Span{TraceID: id, SpanID: id, StartTime: time.Now(), EndTime: time.Now(), ServiceName: "svc"}
	}

	e.AddSpan(span("first"))
	e.AddSpan(span("second"))
	e.AddSpan(span("third")) // should drop "first"

	e.mu.Lock()
	q := make([]*Span, len(e.queue))
	copy(q, e.queue)
	e.mu.Unlock()

	if len(q) != 2 {
		t.Fatalf("expected 2 in queue, got %d", len(q))
	}
	if q[0].TraceID != "second" {
		t.Errorf("expected second to be oldest remaining, got %s", q[0].TraceID)
	}
}

func TestExporter_DrainBatchCapsAllocationWhenConfigIsCorrupted(t *testing.T) {
	queue := make([]*Span, maxExportBatchSize+1)
	e := &OtlpExporter{
		cfg: config.OtelConfig{
			MaxBatchSize: int(^uint(0) >> 1),
		},
		queue: queue,
	}

	e.mu.Lock()
	batch := e.drainBatch()
	e.mu.Unlock()

	if len(batch) != maxExportBatchSize {
		t.Fatalf("expected batch to be capped at %d spans, got %d", maxExportBatchSize, len(batch))
	}
	if len(e.queue) != 1 {
		t.Fatalf("expected one span to remain queued, got %d", len(e.queue))
	}
}

func TestExporter_DrainBatchRejectsNonPositiveSize(t *testing.T) {
	e := &OtlpExporter{
		cfg:   config.OtelConfig{MaxBatchSize: -1},
		queue: []*Span{{TraceID: "still-queued"}},
	}

	e.mu.Lock()
	batch := e.drainBatch()
	e.mu.Unlock()

	if batch != nil {
		t.Fatalf("expected no batch, got %d spans", len(batch))
	}
	if len(e.queue) != 1 {
		t.Fatalf("expected the queue to remain unchanged, got %d spans", len(e.queue))
	}
}
