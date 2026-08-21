package proxy

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"chaos-proxy-go/internal/config"
)

func TestNewRejectsInvalidRouteWithoutPanicking(t *testing.T) {
	cfg := &config.Config{
		Target: "http://127.0.0.1:1",
		Routes: map[string][]map[string]any{"": nil},
	}
	if _, err := New(cfg, false); err == nil {
		t.Fatal("expected an invalid empty route to return an error")
	}
}

func FuzzRouteConfigurationNeverPanics(f *testing.F) {
	for _, route := range []string{"/fuzz", "GET /fuzz", "", "GET", "GET ", "{", "/{"} {
		f.Add(route)
	}

	f.Fuzz(func(t *testing.T, route string) {
		cfg := &config.Config{
			Target: "http://127.0.0.1:1",
			Routes: map[string][]map[string]any{route: nil},
		}
		_, _ = New(cfg, false)
	})
}

func FuzzFailNthConfigurationNeverPanics(f *testing.F) {
	f.Add(int32(0))
	f.Add(int32(1))
	f.Add(int32(-1))
	f.Add(int32(10))

	f.Fuzz(func(t *testing.T, n int32) {
		cfg := &config.Config{
			Target: "http://127.0.0.1:1",
			Routes: map[string][]map[string]any{
				"/fuzz": {{"failNth": map[string]any{"n": int(n)}}},
			},
		}
		server, err := New(cfg, false)
		if err != nil {
			return
		}
		recorder := httptest.NewRecorder()
		server.router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/fuzz", nil))
	})
}

func FuzzProxyRoundTripPreservesBody(f *testing.F) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	f.Cleanup(upstream.Close)

	server, err := New(&config.Config{Target: upstream.URL}, false)
	if err != nil {
		f.Fatalf("create proxy: %v", err)
	}

	for _, seed := range [][]byte{nil, []byte("hello"), {0, 1, 2, 255}} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 64*1024 {
			t.Skip()
		}
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/fuzz?value=1", bytes.NewReader(data))
		server.router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("proxy returned status %d: %s", recorder.Code, recorder.Body.String())
		}
		if !bytes.Equal(recorder.Body.Bytes(), data) {
			t.Fatalf("round trip changed payload: got %x, want %x", recorder.Body.Bytes(), data)
		}
	})
}

func FuzzReloadVersionTransitions(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3, 4})
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, operations []byte) {
		if len(operations) > 64 {
			t.Skip()
		}
		server, err := New(&config.Config{Target: "http://127.0.0.1:1"}, false)
		if err != nil {
			t.Fatalf("create initial server: %v", err)
		}

		for i, operation := range operations {
			before := server.state.Load()
			next := reloadConfigForOperation(operation)
			result := server.ReloadConfig(next)
			after := server.state.Load()

			if result.OK {
				if result.Version != before.version+1 || after.version != before.version+1 {
					t.Fatalf("operation %d: successful reload advanced from %d to result=%d state=%d", i, before.version, result.Version, after.version)
				}
				if after == before {
					t.Fatalf("operation %d: successful reload retained the old state pointer", i)
				}
				continue
			}

			if result.Version != before.version || after != before {
				t.Fatalf("operation %d: failed reload changed active state from version %d", i, before.version)
			}
		}
	})
}

func reloadConfigForOperation(operation byte) *config.Config {
	switch operation % 5 {
	case 0:
		return &config.Config{Target: "http://127.0.0.1:1"}
	case 1:
		return &config.Config{}
	case 2:
		return &config.Config{
			Target: "http://127.0.0.1:1",
			Routes: map[string][]map[string]any{"": nil},
		}
	case 3:
		return &config.Config{
			Target: "http://127.0.0.1:1",
			Global: []map[string]any{{"unknown": map[string]any{}}},
		}
	default:
		return &config.Config{
			Target: "http://127.0.0.1:1",
			Routes: map[string][]map[string]any{
				"GET /fuzz": {{"failNth": map[string]any{"n": 0}}},
			},
		}
	}
}
