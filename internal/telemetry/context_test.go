package telemetry

import (
	"net/http"
	"testing"
)

func TestParseTraceparent_Valid(t *testing.T) {
	val := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	tc := ParseTraceparent(val)
	if tc == nil {
		t.Fatal("expected non-nil TraceContext")
	}
	if tc.TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("unexpected traceID: %s", tc.TraceID)
	}
	if tc.SpanID != "00f067aa0ba902b7" {
		t.Errorf("unexpected spanID: %s", tc.SpanID)
	}
	if tc.TraceFlags != "01" {
		t.Errorf("unexpected flags: %s", tc.TraceFlags)
	}
	if tc.ParentSpanID != "00f067aa0ba902b7" {
		t.Errorf("parentSpanID should equal incoming spanID")
	}
}

func TestParseTraceparent_InvalidFormat(t *testing.T) {
	cases := []string{
		"",
		"not-valid",
		"01-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", // unsupported version
		"00-00000000000000000000000000000000-00f067aa0ba902b7-01", // all-zero traceID
		"00-4bf92f3577b34da6a3ce929d0e0e4736-0000000000000000-01", // all-zero spanID
		"00-short-00f067aa0ba902b7-01",
	}
	for _, c := range cases {
		if ParseTraceparent(c) != nil {
			t.Errorf("expected nil for input %q", c)
		}
	}
}

func TestFormatTraceparent(t *testing.T) {
	tc := &TraceContext{TraceID: "abc123" + "00000000000000000000000000", SpanID: "deadbeef00000000", TraceFlags: "01"}
	got := FormatTraceparent(tc)
	want := "00-abc12300000000000000000000000000-deadbeef00000000-01"
	if got != want {
		t.Errorf("got %s want %s", got, want)
	}
}

func TestExtractTraceContext_Present(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("Traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	tc := ExtractTraceContext(r)
	if tc == nil {
		t.Fatal("expected trace context")
	}
	if tc.TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("unexpected traceID: %s", tc.TraceID)
	}
}

func TestExtractTraceContext_Absent(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	if ExtractTraceContext(r) != nil {
		t.Error("expected nil when no traceparent header")
	}
}

func TestInjectTraceContext(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	tc := &TraceContext{TraceID: "4bf92f3577b34da6a3ce929d0e0e4736", SpanID: "00f067aa0ba902b7", TraceFlags: "01"}
	InjectTraceContext(r, tc)
	got := r.Header.Get("Traceparent")
	want := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	if got != want {
		t.Errorf("got %s want %s", got, want)
	}
}

func TestNewTraceContext(t *testing.T) {
	tc := NewTraceContext()
	if len(tc.TraceID) != 32 {
		t.Errorf("traceID should be 32 hex chars, got %d", len(tc.TraceID))
	}
	if tc.TraceFlags != "01" {
		t.Errorf("expected flags=01")
	}
}

func TestGenerateSpanID(t *testing.T) {
	a := GenerateSpanID()
	b := GenerateSpanID()
	if len(a) != 16 {
		t.Errorf("spanID should be 16 hex chars, got %d", len(a))
	}
	if a == b {
		t.Error("two generated span IDs should not be equal")
	}
}
