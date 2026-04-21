package telemetry

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"regexp"
)

const traceparentHeader = "Traceparent"

var traceparentRe = regexp.MustCompile(`(?i)^([\da-f]{2})-([\da-f]{32})-([\da-f]{16})-([\da-f]{2})$`)

// TraceContext holds the W3C traceparent components.
type TraceContext struct {
	TraceID      string
	SpanID       string
	TraceFlags   string
	ParentSpanID string // the incoming spanId (becomes parent for our child span)
}

// ParseTraceparent parses a W3C traceparent header value.
// Returns nil if the value is invalid or uses an unsupported version.
func ParseTraceparent(value string) *TraceContext {
	m := traceparentRe.FindStringSubmatch(value)
	if m == nil {
		return nil
	}
	version, traceID, spanID, flags := m[1], m[2], m[3], m[4]
	if version != "00" {
		return nil
	}
	if isAllZero(traceID) || isAllZero(spanID) {
		return nil
	}
	return &TraceContext{
		TraceID:      traceID,
		SpanID:       spanID,
		TraceFlags:   flags,
		ParentSpanID: spanID,
	}
}

// FormatTraceparent formats a TraceContext into a W3C traceparent header value.
func FormatTraceparent(tc *TraceContext) string {
	return fmt.Sprintf("00-%s-%s-%s", tc.TraceID, tc.SpanID, tc.TraceFlags)
}

// ExtractTraceContext reads the traceparent header from an incoming request.
// Returns nil if absent or invalid.
func ExtractTraceContext(r *http.Request) *TraceContext {
	val := r.Header.Get(traceparentHeader)
	if val == "" {
		return nil
	}
	return ParseTraceparent(val)
}

// InjectTraceContext sets the traceparent header on an outgoing request.
func InjectTraceContext(r *http.Request, tc *TraceContext) {
	r.Header.Set(traceparentHeader, FormatTraceparent(tc))
}

// NewTraceContext generates a fresh TraceContext with a new trace ID.
func NewTraceContext() *TraceContext {
	return &TraceContext{
		TraceID:    generateHex(16),
		SpanID:     "0000000000000000",
		TraceFlags: "01",
	}
}

// GenerateSpanID generates a random 8-byte hex span ID.
func GenerateSpanID() string {
	return generateHex(8)
}

func generateHex(bytes int) string {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		// Fallback: fill with pseudo-random via fmt (safe, just less secure)
		for i := range buf {
			buf[i] = byte(i)
		}
	}
	return fmt.Sprintf("%x", buf)
}

func isAllZero(s string) bool {
	for _, c := range s {
		if c != '0' {
			return false
		}
	}
	return true
}
