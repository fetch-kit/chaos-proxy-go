package config

import (
	"net/url"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func FuzzParseJSONConfiguration(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`{"Target":"http://example.test"}`),
		[]byte(`{"Target":"http://example.test","Port":8080,"Otel":{"ServiceName":"svc","Endpoint":"https://otel.example.test/","FlushIntervalMs":1,"MaxBatchSize":1,"MaxQueueSize":1}}`),
		[]byte(`null`),
		[]byte(`{"Target":`),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		cfg, err := ParseJSON(data)
		if err != nil {
			return
		}
		assertValidatedConfigInvariants(t, cfg)
	})
}

func FuzzParseYAMLConfiguration(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte("target: http://example.test\n"),
		[]byte("target: http://example.test\notel:\n  serviceName: svc\n  endpoint: https://otel.example.test/\n"),
		[]byte(":bad yaml:"),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var cfg Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return
		}
		validated, err := Validate(&cfg)
		if err != nil {
			return
		}
		assertValidatedConfigInvariants(t, validated)
	})
}

func FuzzValidateOtelConfiguration(f *testing.F) {
	f.Add("http://collector.example.test", int32(1), int32(1), int32(1))
	f.Add("https://user:pass@collector.example.test/path", int32(1000), int32(10), int32(100))
	f.Add("file:///tmp/collector", int32(-1), int32(-1), int32(-1))

	f.Fuzz(func(t *testing.T, endpoint string, flush, batch, queue int32) {
		cfg := &Config{
			Target: "http://example.test",
			Otel: &OtelConfig{
				ServiceName:     "fuzz",
				Endpoint:        endpoint,
				FlushIntervalMs: int(flush),
				MaxBatchSize:    int(batch),
				MaxQueueSize:    int(queue),
			},
		}
		validated, err := Validate(cfg)
		if err != nil {
			return
		}
		assertValidatedConfigInvariants(t, validated)
	})
}

func assertValidatedConfigInvariants(t *testing.T, cfg *Config) {
	t.Helper()
	if cfg == nil {
		t.Fatal("successful validation returned nil config")
	}
	if cfg.Target == "" {
		t.Fatal("successful validation returned an empty target")
	}
	if cfg.Port == 0 {
		t.Fatal("successful validation left the port unset")
	}
	if cfg.Otel == nil {
		return
	}

	endpoint, err := url.Parse(cfg.Otel.Endpoint)
	if err != nil || endpoint.Host == "" {
		t.Fatalf("successful validation returned an invalid OTLP endpoint %q", cfg.Otel.Endpoint)
	}
	if endpoint.Scheme != "http" && endpoint.Scheme != "https" {
		t.Fatalf("successful validation returned disallowed scheme %q", endpoint.Scheme)
	}
	if endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		t.Fatalf("successful validation retained forbidden endpoint components: %q", cfg.Otel.Endpoint)
	}
	if strings.HasSuffix(cfg.Otel.Endpoint, "/") {
		t.Fatalf("successful validation retained a trailing slash: %q", cfg.Otel.Endpoint)
	}
	if cfg.Otel.FlushIntervalMs < 1 || cfg.Otel.FlushIntervalMs > maxOtelFlushIntervalMs {
		t.Fatalf("flush interval escaped bounds: %d", cfg.Otel.FlushIntervalMs)
	}
	if cfg.Otel.MaxBatchSize < 1 || cfg.Otel.MaxBatchSize > maxOtelBatchSize {
		t.Fatalf("batch size escaped bounds: %d", cfg.Otel.MaxBatchSize)
	}
	if cfg.Otel.MaxQueueSize < 1 || cfg.Otel.MaxQueueSize > maxOtelQueueSize {
		t.Fatalf("queue size escaped bounds: %d", cfg.Otel.MaxQueueSize)
	}
	if cfg.Otel.MaxBatchSize > cfg.Otel.MaxQueueSize {
		t.Fatalf("batch size %d exceeds queue size %d", cfg.Otel.MaxBatchSize, cfg.Otel.MaxQueueSize)
	}
}
