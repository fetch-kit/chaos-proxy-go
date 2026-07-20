package logging

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

// captureOutput redirects the package writers for the duration of fn.
func captureOutput(t *testing.T, fn func()) (out string, errOut string) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	origOut, origErr := Out, Err
	Out, Err = &outBuf, &errBuf
	defer func() { Out, Err = origOut, origErr }()
	fn()
	return outBuf.String(), errBuf.String()
}

func TestEmitVerbose_Disabled(t *testing.T) {
	out, errOut := captureOutput(t, func() {
		EmitVerbose(false, "verbose.request.begin", map[string]any{"a": 1}, LevelInfo)
	})
	if out != "" || errOut != "" {
		t.Errorf("expected no output when disabled, got out=%q err=%q", out, errOut)
	}
}

func TestEmitVerbose_NumericAndBooleanFields(t *testing.T) {
	out, _ := captureOutput(t, func() {
		EmitVerbose(true, "verbose.request.end", map[string]any{
			"status":      200,
			"duration_ms": int64(12),
			"cached":      true,
		}, LevelInfo)
	})
	for _, want := range []string{
		"event=verbose.request.end",
		"level=INFO",
		"status=200",
		"duration_ms=12",
		"cached=true",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q: %s", want, out)
		}
	}
}

func TestEmitVerbose_WarnGoesToStderr(t *testing.T) {
	out, errOut := captureOutput(t, func() {
		EmitVerbose(true, "verbose.request.end", map[string]any{"status": 503}, LevelWarn)
	})
	if out != "" {
		t.Errorf("expected nothing on stdout, got %q", out)
	}
	if !strings.Contains(errOut, "level=WARN") || !strings.Contains(errOut, "status=503") {
		t.Errorf("stderr missing warn line: %s", errOut)
	}
}

func TestEmitVerbose_QuotesValuesWithWhitespace(t *testing.T) {
	out, _ := captureOutput(t, func() {
		EmitVerbose(true, "verbose.error", map[string]any{
			"message": "connection refused by host",
		}, LevelInfo)
	})
	if !strings.Contains(out, `message="connection refused by host"`) {
		t.Errorf("expected quoted message, got %s", out)
	}
}

func TestEmitVerbose_SanitizesControlChars(t *testing.T) {
	out, _ := captureOutput(t, func() {
		EmitVerbose(true, "verbose.request.begin", map[string]any{
			"path": "/a\nb\tc",
		}, LevelInfo)
	})
	if strings.Contains(out, "\n/a") || strings.Contains(out, "\tc") {
		t.Errorf("control characters were not sanitized: %q", out)
	}
	// Newline injection must not create extra log lines.
	if strings.Count(strings.TrimRight(out, "\n"), "\n") != 0 {
		t.Errorf("output contains embedded newline: %q", out)
	}
}

func TestEmitVerbose_DefaultLevelIsInfo(t *testing.T) {
	out, _ := captureOutput(t, func() {
		EmitVerbose(true, "verbose.startup", map[string]any{}, "")
	})
	if !strings.Contains(out, "level=INFO") {
		t.Errorf("expected default INFO level, got %s", out)
	}
}

func TestRedactURLQuery(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"no query", "/api/users", "/api/users"},
		{"redacts token", "/api?token=abc123", "/api?token=%5BREDACTED%5D"},
		{"case-insensitive key", "/api?Token=abc123", "/api?Token=%5BREDACTED%5D"},
		{"keeps safe params", "/api?page=2", "/api?page=2"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := RedactURLQuery(c.in)
			if got != c.want {
				t.Errorf("RedactURLQuery(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestRedactURLQuery_MultipleSensitiveKeys(t *testing.T) {
	got := RedactURLQuery("/x?password=p&secret=s&keep=1")
	if strings.Contains(got, "password=p") || strings.Contains(got, "secret=s") {
		t.Errorf("sensitive values not redacted: %s", got)
	}
	if !strings.Contains(got, "keep=1") {
		t.Errorf("non-sensitive value dropped: %s", got)
	}
}

func TestCreateRequestID(t *testing.T) {
	id := CreateRequestID()
	if !strings.HasPrefix(id, "rq_") {
		t.Errorf("expected rq_ prefix, got %q", id)
	}
	if len(id) != len("rq_")+8 {
		t.Errorf("expected 8 hex chars, got %q", id)
	}
	if CreateRequestID() == id {
		t.Error("expected unique request ids")
	}
}

func TestExtractTraceID(t *testing.T) {
	h := http.Header{}
	if got := ExtractTraceID(h); got != "" {
		t.Errorf("expected empty for missing header, got %q", got)
	}

	h.Set("Traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	if got := ExtractTraceID(h); got != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("unexpected trace id: %q", got)
	}

	h.Set("Traceparent", "garbage")
	if got := ExtractTraceID(h); got != "" {
		t.Errorf("expected empty for invalid traceparent, got %q", got)
	}
}
