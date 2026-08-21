package telemetry

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"chaos-proxy-go/internal/config"
)

const (
	initialSpanQueueCapacity = 100
	maxExportBatchSize       = 1000
)

// OtlpExporter batches spans and ships them to an OTLP HTTP endpoint.
type OtlpExporter struct {
	cfg          config.OtelConfig
	mu           sync.Mutex
	queue        []*Span
	ticker       *time.Ticker
	done         chan struct{}
	closed       bool
	shutdownOnce sync.Once
}

// NewExporter creates and starts an OtlpExporter.
func NewExporter(cfg config.OtelConfig) *OtlpExporter {
	e := &OtlpExporter{
		cfg:   cfg,
		queue: make([]*Span, 0, initialSpanQueueCapacity),
		done:  make(chan struct{}),
	}
	e.ticker = time.NewTicker(time.Duration(cfg.FlushIntervalMs) * time.Millisecond)
	go e.flushLoop()
	return e
}

// AddSpan enqueues a span. If the queue is full, the oldest span is dropped (FIFO).
func (e *OtlpExporter) AddSpan(span *Span) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return
	}

	if len(e.queue) >= e.cfg.MaxQueueSize {
		e.queue = e.queue[1:] // drop oldest
	}
	e.queue = append(e.queue, span)

	if len(e.queue) >= e.cfg.MaxBatchSize {
		batch := e.drainBatch()
		go e.exportBatch(batch)
	}
}

// Shutdown stops the flush timer and flushes remaining spans synchronously.
func (e *OtlpExporter) Shutdown() {
	e.shutdownOnce.Do(func() {
		e.mu.Lock()
		e.closed = true
		e.mu.Unlock()

		e.ticker.Stop()
		close(e.done)
		e.mu.Lock()
		batch := e.drainBatch()
		e.mu.Unlock()
		if len(batch) > 0 {
			e.exportBatch(batch)
		}
	})
}

func (e *OtlpExporter) flushLoop() {
	for {
		select {
		case <-e.ticker.C:
			e.mu.Lock()
			batch := e.drainBatch()
			e.mu.Unlock()
			if len(batch) > 0 {
				e.exportBatch(batch)
			}
		case <-e.done:
			return
		}
	}
}

// drainBatch takes up to MaxBatchSize spans from the queue. Must be called with lock held.
func (e *OtlpExporter) drainBatch() []*Span {
	n := e.cfg.MaxBatchSize
	if n > maxExportBatchSize {
		n = maxExportBatchSize
	}
	if n <= 0 {
		return nil
	}
	if len(e.queue) < n {
		n = len(e.queue)
	}
	batch := make([]*Span, n)
	copy(batch, e.queue[:n])
	e.queue = e.queue[n:]
	return batch
}

func (e *OtlpExporter) exportBatch(spans []*Span) {
	otlpSpans := make([]map[string]any, len(spans))
	for i, s := range spans {
		otlpSpans[i] = SpanToOtlpJSON(s)
	}

	payload := map[string]any{
		"resourceSpans": []map[string]any{
			{
				"resource": map[string]any{
					"attributes": []map[string]any{
						{"key": "service.name", "value": map[string]any{"stringValue": e.cfg.ServiceName}},
					},
				},
				"scopeSpans": []map[string]any{
					{
						"scope": map[string]any{
							"name":    "chaos-proxy-go",
							"version": "0.1.0",
						},
						"spans": otlpSpans,
					},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[otel] failed to marshal spans: %v", err)
		return
	}

	endpoint, err := url.JoinPath(e.cfg.Endpoint, "v1/traces")
	if err != nil {
		log.Printf("[otel] failed to build endpoint: %v", err)
		return
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		log.Printf("[otel] failed to build request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range e.cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[otel] export failed: %v", err)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("[otel] failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode >= 300 {
		log.Printf("[otel] export returned %d", resp.StatusCode)
	}
}
