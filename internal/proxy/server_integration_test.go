package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"chaos-proxy-go/internal/config"
)

func TestIntegration_BasicProxy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Start a test upstream server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream", "yes")
		w.WriteHeader(200)
		if _, err := w.Write([]byte("hello from upstream")); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	defer upstream.Close()

	// Prepare proxy config
	cfg := &config.Config{
		Target: upstream.URL,
		Port:   0, // not used with httptest.Server
		Global: nil,
		Routes: map[string][]map[string]any{},
	}

	// Create proxy server
	ps, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}

	// Start proxy as httptest.Server
	proxySrv := httptest.NewServer(ps.router)
	defer proxySrv.Close()

	// Send request to proxy
	resp, err := http.Get(proxySrv.URL + "/foo")
	if err != nil {
		t.Fatalf("proxy request failed: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("resp.Body.Close: %v", err)
		}
	}()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Upstream") != "yes" {
		t.Errorf("expected X-Upstream header from upstream")
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(data), "hello from upstream") {
		t.Errorf("expected upstream body, got %q", string(data))
	}
}

func TestIntegration_LatencyMiddleware(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Start a test upstream server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if _, err := w.Write([]byte("ok")); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	defer upstream.Close()

	// Prepare proxy config with latency middleware (e.g., 200ms)
	latencyMs := 200
	cfg := &config.Config{
		Target: upstream.URL,
		Port:   0,
		Global: []map[string]any{{"latency": map[string]any{"ms": latencyMs}}},
		Routes: map[string][]map[string]any{},
	}

	ps, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}
	proxySrv := httptest.NewServer(ps.router)
	defer proxySrv.Close()

	start := time.Now()
	resp, err := http.Get(proxySrv.URL + "/test-latency")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("proxy request failed: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("resp.Body.Close: %v", err)
		}
	}()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if elapsed < time.Duration(latencyMs)*time.Millisecond {
		t.Errorf("expected at least %dms latency, got %v", latencyMs, elapsed)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(data) != "ok" {
		t.Errorf("expected body 'ok', got %q", string(data))
	}
}

func TestIntegration_FailMiddleware(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Start a test upstream server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if _, err := w.Write([]byte("should not see this")); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	defer upstream.Close()

	// Prepare proxy config with fail middleware (e.g., status 418, body "fail")
	failStatus := 418
	failBody := "fail"
	cfg := &config.Config{
		Target: upstream.URL,
		Port:   0,
		Global: []map[string]any{{"fail": map[string]any{"status": failStatus, "body": failBody}}},
		Routes: map[string][]map[string]any{},
	}

	ps, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}
	proxySrv := httptest.NewServer(ps.router)
	defer proxySrv.Close()

	resp, err := http.Get(proxySrv.URL + "/fail-test")
	if err != nil {
		t.Fatalf("proxy request failed: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("resp.Body.Close: %v", err)
		}
	}()
	if resp.StatusCode != failStatus {
		t.Errorf("expected status %d, got %d", failStatus, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(data) != failBody {
		t.Errorf("expected body %q, got %q", failBody, string(data))
	}
}

func TestIntegration_HeaderTransform(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Start a test upstream server that echoes headers
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream", r.Header.Get("X-Transformed"))
		w.WriteHeader(200)
		if _, err := w.Write([]byte("ok")); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	defer upstream.Close()

	// Prepare proxy config with headerTransform middleware
	cfg := &config.Config{
		Target: upstream.URL,
		Port:   0,
		Global: []map[string]any{{
			"headerTransform": map[string]any{
				"request": map[string]any{
					"set":    map[string]any{"X-Transformed": "foo"},
					"delete": []any{"X-RemoveMe"},
				},
				"response": map[string]any{
					"set":    map[string]any{"X-Response-Set": "bar"},
					"delete": []any{"X-Upstream"},
				},
			},
		}},
		Routes: map[string][]map[string]any{},
	}

	ps, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}
	proxySrv := httptest.NewServer(ps.router)
	defer proxySrv.Close()

	req, _ := http.NewRequest("GET", proxySrv.URL+"/header-test", nil)
	req.Header.Set("X-RemoveMe", "bye")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("proxy request failed: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("resp.Body.Close: %v", err)
		}
	}()
	// X-Transformed should be set in request to upstream, X-RemoveMe should be deleted
	// X-Response-Set should be set in response, X-Upstream should be deleted
	if resp.Header.Get("X-Response-Set") != "bar" {
		t.Errorf("expected X-Response-Set=bar, got %q", resp.Header.Get("X-Response-Set"))
	}
	if resp.Header.Get("X-Upstream") != "" {
		t.Errorf("expected X-Upstream to be deleted, got %q", resp.Header.Get("X-Upstream"))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(data) != "ok" {
		t.Errorf("expected body 'ok', got %q", string(data))
	}
}

func TestIntegration_BodyTransformJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Start a test upstream server that echoes JSON body
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, err := w.Write(body); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	defer upstream.Close()

	// Prepare proxy config with bodyTransformJSON middleware
	cfg := &config.Config{
		Target: upstream.URL,
		Port:   0,
		Global: []map[string]any{{
			"bodyTransformJSON": map[string]any{
				"request": map[string]any{
					"set":    map[string]any{"foo": "bar"},
					"delete": []any{"removeMe"},
				},
				"response": map[string]any{
					"set":    map[string]any{"baz": 123},
					"delete": []any{"deleteMe"},
				},
			},
		}},
		Routes: map[string][]map[string]any{},
	}

	ps, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}
	proxySrv := httptest.NewServer(ps.router)
	defer proxySrv.Close()

	reqBody := `{"removeMe":"bye","foo":"original"}`
	req, _ := http.NewRequest("POST", proxySrv.URL+"/body-test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("proxy request failed: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("resp.Body.Close: %v", err)
		}
	}()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("invalid JSON in response: %v", err)
	}
	// foo should be set to "bar", removeMe should be deleted, baz should be set, deleteMe should be deleted
	if m["foo"] != "bar" {
		t.Errorf("expected foo=bar, got %v", m["foo"])
	}
	if _, ok := m["removeMe"]; ok {
		t.Errorf("expected removeMe to be deleted, got %v", m["removeMe"])
	}
	if m["baz"] != float64(123) {
		t.Errorf("expected baz=123, got %v", m["baz"])
	}
	if _, ok := m["deleteMe"]; ok {
		t.Errorf("expected deleteMe to be deleted, got %v", m["deleteMe"])
	}
}

func TestIntegration_RouteSpecificMiddleware(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Start a test upstream server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream", "yes")
		w.Header().Set("X-Route", r.URL.Path)
		w.WriteHeader(200)
		if _, err := w.Write([]byte("ok")); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	defer upstream.Close()

	// Prepare proxy config with different middleware for different routes
	cfg := &config.Config{
		Target: upstream.URL,
		Port:   0,
		Global: nil,
		Routes: map[string][]map[string]any{
			"/foo": {{"headerTransform": map[string]any{"response": map[string]any{"set": map[string]any{"X-Foo": "bar"}}}}},
			"/bar": {{"fail": map[string]any{"status": 418, "body": "teapot"}}},
		},
	}

	ps, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}
	proxySrv := httptest.NewServer(ps.router)
	defer proxySrv.Close()

	// /foo should get X-Foo header
	resp, err := http.Get(proxySrv.URL + "/foo")
	if err != nil {
		t.Fatalf("proxy request to /foo failed: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("resp.Body.Close: %v", err)
		}
	}()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for /foo, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Foo") != "bar" {
		t.Errorf("expected X-Foo=bar for /foo, got %q", resp.Header.Get("X-Foo"))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(data) != "ok" {
		t.Errorf("expected body 'ok' for /foo, got %q", string(data))
	}

	// /bar should get fail middleware (418, teapot)
	resp2, err := http.Get(proxySrv.URL + "/bar")
	if err != nil {
		t.Fatalf("proxy request to /bar failed: %v", err)
	}
	defer func() {
		if err := resp2.Body.Close(); err != nil {
			t.Errorf("resp2.Body.Close: %v", err)
		}
	}()
	if resp2.StatusCode != 418 {
		t.Errorf("expected 418 for /bar, got %d", resp2.StatusCode)
	}
	data2, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(data2) != "teapot" {
		t.Errorf("expected body 'teapot' for /bar, got %q", string(data2))
	}
}

func TestIntegration_404NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Start a test upstream server that returns 404 for unknown routes
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/known" {
			w.WriteHeader(200)
			if _, err := w.Write([]byte("ok")); err != nil {
				t.Errorf("write: %v", err)
			}
		} else {
			w.WriteHeader(404)
			if _, err := w.Write([]byte("not found")); err != nil {
				t.Errorf("write: %v", err)
			}
		}
	}))
	defer upstream.Close()

	// Proxy config with no matching routes
	cfg := &config.Config{
		Target: upstream.URL,
		Port:   0,
		Global: nil,
		Routes: map[string][]map[string]any{},
	}

	ps, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}
	proxySrv := httptest.NewServer(ps.router)
	defer proxySrv.Close()

	// Send request to unknown route
	resp, err := http.Get(proxySrv.URL + "/notfound")
	if err != nil {
		t.Fatalf("proxy request failed: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("resp.Body.Close: %v", err)
		}
	}()
	if resp.StatusCode != 404 {
		t.Errorf("expected 404 for unknown route, got %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if len(data) == 0 {
		t.Errorf("expected non-empty body for 404 response")
	}
}

func TestIntegration_UpstreamError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Start a test upstream server that returns 502 for a specific route
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bad-gateway" {
			w.WriteHeader(502)
			if _, err := w.Write([]byte("bad gateway error")); err != nil {
				t.Errorf("write: %v", err)
			}
		} else {
			w.WriteHeader(200)
			if _, err := w.Write([]byte("ok")); err != nil {
				t.Errorf("write: %v", err)
			}
		}
	}))
	defer upstream.Close()

	cfg := &config.Config{
		Target: upstream.URL,
		Port:   0,
		Global: nil,
		Routes: map[string][]map[string]any{},
	}

	ps, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}
	proxySrv := httptest.NewServer(ps.router)
	defer proxySrv.Close()

	// Send request to /bad-gateway, expect 502
	resp, err := http.Get(proxySrv.URL + "/bad-gateway")
	if err != nil {
		t.Fatalf("proxy request failed: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("resp.Body.Close: %v", err)
		}
	}()
	if resp.StatusCode != 502 {
		t.Errorf("expected 502 from upstream, got %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(data) != "bad gateway error" {
		t.Errorf("expected body 'bad gateway error', got %q", string(data))
	}
}

func TestIntegration_ConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Start a test upstream server that echoes the request path
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if _, err := w.Write([]byte(r.URL.Path)); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	defer upstream.Close()

	cfg := &config.Config{
		Target: upstream.URL,
		Port:   0,
		Global: nil,
		Routes: map[string][]map[string]any{},
	}

	ps, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}
	proxySrv := httptest.NewServer(ps.router)
	defer proxySrv.Close()

	numRequests := 20
	errs := make(chan error, numRequests)
	for i := 0; i < numRequests; i++ {
		go func(i int) {
			path := "/concurrent-" + strconv.Itoa(i)
			resp, err := http.Get(proxySrv.URL + path)
			if err != nil {
				errs <- err
				return
			}
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Errorf("resp.Body.Close: %v", err)
				}
			}()
			if resp.StatusCode != 200 {
				errs <- fmt.Errorf("expected 200, got %d", resp.StatusCode)
				return
			}
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				errs <- fmt.Errorf("read body: %v", err)
				return
			}
			if string(data) != path {
				errs <- fmt.Errorf("expected body %q, got %q", path, string(data))
				return
			}
			errs <- nil
		}(i)
	}
	for i := 0; i < numRequests; i++ {
		if err := <-errs; err != nil {
			t.Error(err)
		}
	}
}

func TestIntegration_ServerShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Start a test upstream server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		if _, err := w.Write([]byte("ok")); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	defer upstream.Close()

	cfg := &config.Config{
		Target: upstream.URL,
		Port:   0,
		Global: nil,
		Routes: map[string][]map[string]any{},
	}

	ps, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}
	proxySrv := httptest.NewServer(ps.router)

	// Send a request to ensure the server is running
	resp, err := http.Get(proxySrv.URL + "/shutdown-test")
	if err != nil {
		t.Fatalf("proxy request failed: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("resp.Body.Close: %v", err)
		}
	}()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(data) != "ok" {
		t.Errorf("expected body 'ok', got %q", string(data))
	}

	// Shut down the proxy server
	proxySrv.Close()

	// After shutdown, requests should fail
	_, err = http.Get(proxySrv.URL + "/shutdown-test")
	if err == nil {
		t.Errorf("expected error after server shutdown, got nil")
	}
}
