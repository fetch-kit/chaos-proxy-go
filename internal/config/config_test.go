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
