package telemetry

import (
	"testing"
	"time"
)

func TestSpanToOtlpJSON_Basic(t *testing.T) {
	span := &Span{
		TraceID:     "abc123abc123abc123abc123abc12345",
		SpanID:      "deadbeef12345678",
		Name:        "GET /api/users",
		StartTime:   time.UnixMilli(1000),
		EndTime:     time.UnixMilli(1050),
		Method:      "GET",
		URL:         "http://localhost:4000/api/users",
		Path:        "/api/users",
		Status:      200,
		ServiceName: "test-svc",
	}
	out := SpanToOtlpJSON(span)

	if out["traceId"] != span.TraceID {
		t.Errorf("traceId mismatch")
	}
	if out["spanId"] != span.SpanID {
		t.Errorf("spanId mismatch")
	}
	if out["name"] != "GET /api/users" {
		t.Errorf("name mismatch: %v", out["name"])
	}

	status, ok := out["status"].(map[string]any)
	if !ok {
		t.Fatal("expected status map")
	}
	if status["code"] != "STATUS_CODE_OK" {
		t.Errorf("expected STATUS_CODE_OK, got %v", status["code"])
	}
}

func TestSpanToOtlpJSON_Error(t *testing.T) {
	span := &Span{
		TraceID:      "abc123abc123abc123abc123abc12345",
		SpanID:       "deadbeef12345678",
		StartTime:    time.UnixMilli(1000),
		EndTime:      time.UnixMilli(1100),
		Method:       "POST",
		Status:       503,
		Error:        true,
		ErrorMessage: "service unavailable",
		ServiceName:  "test-svc",
	}
	out := SpanToOtlpJSON(span)
	status := out["status"].(map[string]any)
	if status["code"] != "STATUS_CODE_ERROR" {
		t.Errorf("expected STATUS_CODE_ERROR, got %v", status["code"])
	}
	if status["message"] != "service unavailable" {
		t.Errorf("unexpected message: %v", status["message"])
	}
}

func TestSpanToOtlpJSON_ParentSpanId(t *testing.T) {
	span := &Span{
		TraceID:      "abc123abc123abc123abc123abc12345",
		SpanID:       "deadbeef12345678",
		ParentSpanID: "parentspan123456",
		StartTime:    time.UnixMilli(1000),
		EndTime:      time.UnixMilli(1010),
		ServiceName:  "test-svc",
	}
	out := SpanToOtlpJSON(span)
	if out["parentSpanId"] != "parentspan123456" {
		t.Errorf("expected parentSpanId, got %v", out["parentSpanId"])
	}
}

func TestSpanToOtlpJSON_NoParentSpanId(t *testing.T) {
	span := &Span{
		TraceID:     "abc123abc123abc123abc123abc12345",
		SpanID:      "deadbeef12345678",
		StartTime:   time.UnixMilli(1000),
		EndTime:     time.UnixMilli(1010),
		ServiceName: "test-svc",
	}
	out := SpanToOtlpJSON(span)
	if _, ok := out["parentSpanId"]; ok {
		t.Error("parentSpanId should be absent when empty")
	}
}

func TestNewSpan(t *testing.T) {
	tc := &TraceContext{TraceID: "traceid12345678901234567890abcd", SpanID: "span1234", TraceFlags: "01"}
	span := NewSpan(tc, "GET", "http://localhost:4000/api/orders?limit=10", "my-service")

	if span.TraceID != tc.TraceID {
		t.Errorf("traceID mismatch")
	}
	if span.Method != "GET" {
		t.Errorf("method mismatch")
	}
	if span.Path != "/api/orders?limit=10" {
		t.Errorf("path mismatch: %s", span.Path)
	}
	if span.ServiceName != "my-service" {
		t.Errorf("serviceName mismatch")
	}
	if span.StartTime.IsZero() {
		t.Error("startTime should be set")
	}
}
