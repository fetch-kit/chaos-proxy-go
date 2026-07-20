package proxy

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"chaos-proxy-go/internal/config"
	"chaos-proxy-go/internal/logging"
)

// captureVerbose redirects the logging package writers for the duration of fn.
func captureVerbose(t *testing.T, fn func()) (out, errOut string) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	origOut, origErr := logging.Out, logging.Err
	logging.Out, logging.Err = &outBuf, &errBuf
	defer func() { logging.Out, logging.Err = origOut, origErr }()
	fn()
	return outBuf.String(), errBuf.String()
}

func TestIntegration_VerboseRequestEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	cfg := &config.Config{
		Target: upstream.URL,
		Port:   0,
		Routes: map[string][]map[string]any{},
	}

	ps, err := New(cfg, true)
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}
	proxySrv := httptest.NewServer(ps.router)
	defer proxySrv.Close()

	out, _ := captureVerbose(t, func() {
		resp, err := http.Get(proxySrv.URL + "/foo?token=secretvalue&page=2")
		if err != nil {
			t.Fatalf("proxy request failed: %v", err)
		}
		_ = resp.Body.Close()
	})

	if !strings.Contains(out, "event=verbose.request.begin") {
		t.Errorf("missing request.begin event: %s", out)
	}
	if !strings.Contains(out, "event=verbose.request.end") {
		t.Errorf("missing request.end event: %s", out)
	}
	if !strings.Contains(out, "req_id=rq_") {
		t.Errorf("missing req_id: %s", out)
	}
	if !strings.Contains(out, "result=ok") {
		t.Errorf("missing ok result: %s", out)
	}
	// Sensitive query values must be redacted, safe ones preserved.
	if strings.Contains(out, "token=secretvalue") {
		t.Errorf("token was not redacted: %s", out)
	}
	if !strings.Contains(out, "REDACTED") {
		t.Errorf("expected REDACTED marker: %s", out)
	}
	if !strings.Contains(out, "page=2") {
		t.Errorf("expected safe query param preserved: %s", out)
	}
}

func TestIntegration_VerboseServerErrorLevel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer upstream.Close()

	cfg := &config.Config{
		Target: upstream.URL,
		Port:   0,
		Routes: map[string][]map[string]any{},
	}

	ps, err := New(cfg, true)
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}
	proxySrv := httptest.NewServer(ps.router)
	defer proxySrv.Close()

	_, errOut := captureVerbose(t, func() {
		resp, err := http.Get(proxySrv.URL + "/boom")
		if err != nil {
			t.Fatalf("proxy request failed: %v", err)
		}
		_ = resp.Body.Close()
	})

	// 5xx should log request.end at WARN level (stderr) with result=error.
	if !strings.Contains(errOut, "event=verbose.request.end") {
		t.Errorf("missing request.end on stderr: %s", errOut)
	}
	if !strings.Contains(errOut, "level=WARN") {
		t.Errorf("expected WARN level for 5xx: %s", errOut)
	}
	if !strings.Contains(errOut, "result=error") {
		t.Errorf("expected result=error: %s", errOut)
	}
}

func TestIntegration_VerboseShutdownEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	cfg := &config.Config{
		Target: upstream.URL,
		Port:   0,
		Routes: map[string][]map[string]any{},
	}

	ps, err := New(cfg, true)
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}

	out, _ := captureVerbose(t, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := ps.Shutdown(ctx); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	})

	if !strings.Contains(out, "event=verbose.shutdown") {
		t.Errorf("missing shutdown event: %s", out)
	}
}
