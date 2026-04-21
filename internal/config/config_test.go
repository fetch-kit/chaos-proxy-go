package config

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfig_YAMLParseValid(t *testing.T) {
	yamlStr := `
port: 8080
target: http://localhost:9000
global:
  - latency:
      ms: 100
routes:
  /api:
    - fail:
        status: 500
        body: error
`
	var cfg Config
	err := yaml.Unmarshal([]byte(yamlStr), &cfg)
	if err != nil {
		t.Fatalf("failed to parse valid yaml: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected port=8080, got %d", cfg.Port)
	}
	if cfg.Target != "http://localhost:9000" {
		t.Errorf("expected target, got %s", cfg.Target)
	}
	if len(cfg.Global) != 1 {
		t.Errorf("expected 1 global middleware, got %d", len(cfg.Global))
	}
	if len(cfg.Routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(cfg.Routes))
	}
}

func TestConfig_YAMLParseDefaultPort(t *testing.T) {
	yamlStr := `
target: http://localhost:9000
`
	f, err := os.CreateTemp("", "cfg-*.yaml")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(f.Name()); err != nil {
			t.Logf("failed to remove temp file: %v", err)
		}
	}()
	if _, err := f.WriteString(yamlStr); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	cfg, err := Load(f.Name())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Port != 5000 {
		t.Errorf("expected default port=5000, got %d", cfg.Port)
	}
}

func TestConfig_YAMLParseMissingTarget(t *testing.T) {
	yamlStr := `
port: 1234
`
	f, err := os.CreateTemp("", "cfg-*.yaml")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(f.Name()); err != nil {
			t.Logf("failed to remove temp file: %v", err)
		}
	}()
	if _, err := f.WriteString(yamlStr); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	_, err = Load(f.Name())
	if err == nil || err.Error() != "target is required" {
		t.Errorf("expected error for missing target, got %v", err)
	}
}

func TestConfig_YAMLParseInvalidYAML(t *testing.T) {
	yamlStr := `:bad yaml:`
	f, err := os.CreateTemp("", "cfg-*.yaml")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(f.Name()); err != nil {
			t.Logf("failed to remove temp file: %v", err)
		}
	}()
	if _, err := f.WriteString(yamlStr); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	_, err = Load(f.Name())
	if err == nil {
		t.Error("expected error for invalid yaml")
	}
}

func TestConfig_YAMLParseGlobalAndRoutes(t *testing.T) {
	yamlStr := `
target: http://localhost:9000
global:
  - latency:
      ms: 50
  - fail:
      status: 400
      body: fail
routes:
  /foo:
    - latency:
        ms: 10
  /bar:
    - fail:
        status: 404
        body: not found
`
	var cfg Config
	err := yaml.Unmarshal([]byte(yamlStr), &cfg)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(cfg.Global) != 2 {
		t.Errorf("expected 2 global, got %d", len(cfg.Global))
	}
	if len(cfg.Routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(cfg.Routes))
	}
	if _, ok := cfg.Routes["/foo"]; !ok {
		t.Errorf("expected /foo route")
	}
	if _, ok := cfg.Routes["/bar"]; !ok {
		t.Errorf("expected /bar route")
	}
}

func TestConfig_OtelParsed(t *testing.T) {
	yamlStr := `
target: http://localhost:9000
otel:
  serviceName: my-service
  endpoint: http://localhost:4318
  flushIntervalMs: 1000
  maxBatchSize: 20
  maxQueueSize: 500
  headers:
    x-tenant-id: local-dev
`
	var cfg Config
	err := yaml.Unmarshal([]byte(yamlStr), &cfg)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.Otel == nil {
		t.Fatal("expected otel config, got nil")
	}
	if cfg.Otel.ServiceName != "my-service" {
		t.Errorf("expected serviceName=my-service, got %s", cfg.Otel.ServiceName)
	}
	if cfg.Otel.Endpoint != "http://localhost:4318" {
		t.Errorf("expected endpoint, got %s", cfg.Otel.Endpoint)
	}
	if cfg.Otel.FlushIntervalMs != 1000 {
		t.Errorf("expected flushIntervalMs=1000, got %d", cfg.Otel.FlushIntervalMs)
	}
	if cfg.Otel.MaxBatchSize != 20 {
		t.Errorf("expected maxBatchSize=20, got %d", cfg.Otel.MaxBatchSize)
	}
	if cfg.Otel.MaxQueueSize != 500 {
		t.Errorf("expected maxQueueSize=500, got %d", cfg.Otel.MaxQueueSize)
	}
	if cfg.Otel.Headers["x-tenant-id"] != "local-dev" {
		t.Errorf("expected header x-tenant-id=local-dev, got %s", cfg.Otel.Headers["x-tenant-id"])
	}
}

func TestConfig_OtelDefaults(t *testing.T) {
	yamlStr := `
target: http://localhost:9000
otel:
  serviceName: my-service
  endpoint: http://localhost:4318
`
	f, err := os.CreateTemp("", "cfg-*.yaml")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(f.Name()); err != nil {
			t.Logf("failed to remove temp file: %v", err)
		}
	}()
	if _, err := f.WriteString(yamlStr); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	cfg, err := Load(f.Name())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Otel.FlushIntervalMs != 5000 {
		t.Errorf("expected default flushIntervalMs=5000, got %d", cfg.Otel.FlushIntervalMs)
	}
	if cfg.Otel.MaxBatchSize != 100 {
		t.Errorf("expected default maxBatchSize=100, got %d", cfg.Otel.MaxBatchSize)
	}
	if cfg.Otel.MaxQueueSize != 1000 {
		t.Errorf("expected default maxQueueSize=1000, got %d", cfg.Otel.MaxQueueSize)
	}
}

func TestConfig_OtelMissingServiceName(t *testing.T) {
	yamlStr := `
target: http://localhost:9000
otel:
  endpoint: http://localhost:4318
`
	f, err := os.CreateTemp("", "cfg-*.yaml")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(f.Name()); err != nil {
			t.Logf("failed to remove temp file: %v", err)
		}
	}()
	if _, err := f.WriteString(yamlStr); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	_, err = Load(f.Name())
	if err == nil || err.Error() != "otel.serviceName is required" {
		t.Errorf("expected otel.serviceName error, got %v", err)
	}
}

func TestConfig_OtelMissingEndpoint(t *testing.T) {
	yamlStr := `
target: http://localhost:9000
otel:
  serviceName: my-service
`
	f, err := os.CreateTemp("", "cfg-*.yaml")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(f.Name()); err != nil {
			t.Logf("failed to remove temp file: %v", err)
		}
	}()
	if _, err := f.WriteString(yamlStr); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	_, err = Load(f.Name())
	if err == nil || err.Error() != "otel.endpoint is required" {
		t.Errorf("expected otel.endpoint error, got %v", err)
	}
}

func TestConfig_OtelAbsent(t *testing.T) {
	yamlStr := `
target: http://localhost:9000
`
	var cfg Config
	err := yaml.Unmarshal([]byte(yamlStr), &cfg)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.Otel != nil {
		t.Errorf("expected otel to be nil when not configured")
	}
}
