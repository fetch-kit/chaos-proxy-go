package telemetry

import (
	"fmt"
	"net/url"
	"time"
)

// Span holds all data for a single proxy request span.
type Span struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	Name         string
	StartTime    time.Time
	EndTime      time.Time
	Method       string
	URL          string
	Path         string
	Status       int
	Error        bool
	ErrorMessage string
	ServiceName  string
}

// msToNanos converts a Unix millisecond timestamp to a nanosecond string for OTLP.
func msToNanos(ms int64) string {
	return fmt.Sprintf("%d", ms*1_000_000)
}

// SpanToOtlpJSON serializes a Span to the OTLP JSON representation.
func SpanToOtlpJSON(span *Span) map[string]any {
	statusCode := "STATUS_CODE_UNSET"
	if span.Error {
		statusCode = "STATUS_CODE_ERROR"
	} else if span.Status > 0 {
		statusCode = "STATUS_CODE_OK"
	}

	attributes := []map[string]any{
		{"key": "http.method", "value": map[string]any{"stringValue": span.Method}},
		{"key": "http.url", "value": map[string]any{"stringValue": span.URL}},
		{"key": "http.target", "value": map[string]any{"stringValue": span.Path}},
		{"key": "service.name", "value": map[string]any{"stringValue": span.ServiceName}},
	}
	if span.Status > 0 {
		attributes = append(attributes, map[string]any{
			"key":   "http.status_code",
			"value": map[string]any{"intValue": span.Status},
		})
	}

	otlpSpan := map[string]any{
		"traceId":           span.TraceID,
		"spanId":            span.SpanID,
		"name":              span.Name,
		"kind":              "SPAN_KIND_SERVER",
		"startTimeUnixNano": msToNanos(span.StartTime.UnixMilli()),
		"endTimeUnixNano":   msToNanos(span.EndTime.UnixMilli()),
		"attributes":        attributes,
		"status": map[string]any{
			"code":    statusCode,
			"message": span.ErrorMessage,
		},
	}

	if span.ParentSpanID != "" {
		otlpSpan["parentSpanId"] = span.ParentSpanID
	}

	return otlpSpan
}

// NewSpan creates a new Span for the given request.
func NewSpan(tc *TraceContext, method, rawURL, serviceName string) *Span {
	spanID := GenerateSpanID()
	path := rawURL
	if u, err := url.Parse(rawURL); err == nil {
		path = u.Path
		if u.RawQuery != "" {
			path += "?" + u.RawQuery
		}
	}

	parentSpanID := ""
	if tc.ParentSpanID != "" && tc.ParentSpanID != "0000000000000000" {
		parentSpanID = tc.ParentSpanID
	}

	return &Span{
		TraceID:      tc.TraceID,
		SpanID:       spanID,
		ParentSpanID: parentSpanID,
		Name:         fmt.Sprintf("%s %s", method, path),
		StartTime:    time.Now(),
		Method:       method,
		URL:          rawURL,
		Path:         path,
		ServiceName:  serviceName,
	}
}
