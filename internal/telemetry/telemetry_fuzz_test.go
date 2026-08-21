package telemetry

import (
	"reflect"
	"testing"

	"chaos-proxy-go/internal/config"
)

func FuzzTraceparentRoundTrip(f *testing.F) {
	for _, seed := range []string{
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		"00-4BF92F3577B34DA6A3CE929D0E0E4736-00F067AA0BA902B7-01",
		"00-00000000000000000000000000000000-00f067aa0ba902b7-01",
		"not-a-traceparent",
		"",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		parsed := ParseTraceparent(value)
		if parsed == nil {
			return
		}
		reparsed := ParseTraceparent(FormatTraceparent(parsed))
		if reparsed == nil {
			t.Fatalf("formatted parsed context became invalid: %#v", parsed)
		}
		if !reflect.DeepEqual(parsed, reparsed) {
			t.Fatalf("trace context changed after round trip: before=%#v after=%#v", parsed, reparsed)
		}
	})
}

func FuzzExporterDrainBatch(f *testing.F) {
	f.Add(uint16(0), int32(1))
	f.Add(uint16(1001), int32(1000))
	f.Add(uint16(1001), int32(1<<30))
	f.Add(uint16(1), int32(-1))

	f.Fuzz(func(t *testing.T, rawQueueSize uint16, rawBatchSize int32) {
		queueSize := int(rawQueueSize % 2001)
		queue := make([]*Span, queueSize)
		for i := range queue {
			queue[i] = &Span{TraceID: string(rune(i + 1))}
		}
		original := append([]*Span(nil), queue...)
		e := &OtlpExporter{
			cfg:   config.OtelConfig{MaxBatchSize: int(rawBatchSize)},
			queue: queue,
		}

		batch := e.drainBatch()
		if len(batch) > maxExportBatchSize {
			t.Fatalf("batch exceeded hard limit: %d", len(batch))
		}
		if len(batch)+len(e.queue) != len(original) {
			t.Fatalf("drain lost or duplicated spans: batch=%d remaining=%d original=%d", len(batch), len(e.queue), len(original))
		}
		for i, span := range batch {
			if span != original[i] {
				t.Fatalf("exported batch changed FIFO order at index %d", i)
			}
		}
		for i, span := range e.queue {
			if span != original[len(batch)+i] {
				t.Fatalf("remaining queue changed FIFO order at index %d", i)
			}
		}
	})
}
