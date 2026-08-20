package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultOtelFlushIntervalMs = 5000
	defaultOtelMaxBatchSize    = 100
	defaultOtelMaxQueueSize    = 1000
	maxOtelFlushIntervalMs     = 86_400_000
	maxOtelBatchSize           = 1000
	maxOtelQueueSize           = 10000
)

// MiddlewareConfig represents a generic middleware configuration
type MiddlewareConfig[T any] struct {
	Name   string
	Config T
}

// OtelConfig holds OpenTelemetry exporter configuration.
// Field names match chaos-proxy's YAML format exactly (camelCase) for config portability.
type OtelConfig struct {
	ServiceName     string            `yaml:"serviceName"`
	Endpoint        string            `yaml:"endpoint"`
	FlushIntervalMs int               `yaml:"flushIntervalMs"`
	MaxBatchSize    int               `yaml:"maxBatchSize"`
	MaxQueueSize    int               `yaml:"maxQueueSize"`
	Headers         map[string]string `yaml:"headers"`
}

// Config represents the main configuration structure
type Config struct {
	Target string                      `yaml:"target"`
	Port   int                         `yaml:"port"`
	Otel   *OtelConfig                 `yaml:"otel"`
	Global []map[string]any            `yaml:"global"`
	Routes map[string][]map[string]any `yaml:"routes"`
}

// LatencyConfig represents latency middleware configuration
type LatencyConfig struct {
	Ms int `yaml:"ms"`
}

// FailConfig represents fail middleware configuration
type FailConfig struct {
	Status int    `yaml:"status"`
	Body   string `yaml:"body"`
}

// Load loads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return Validate(&cfg)
}

// ParseJSON parses and validates a config from a JSON byte slice.
func ParseJSON(data []byte) (*Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return Validate(&cfg)
}

// Validate applies defaults and rejects invalid or unsafe configuration values.
func Validate(cfg *Config) (*Config, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Set defaults
	if cfg.Port == 0 {
		cfg.Port = 5000
	}

	// Validate required fields
	if cfg.Target == "" {
		return nil, fmt.Errorf("target is required")
	}

	if cfg.Otel != nil {
		if cfg.Otel.ServiceName == "" {
			return nil, fmt.Errorf("otel.serviceName is required")
		}
		if cfg.Otel.Endpoint == "" {
			return nil, fmt.Errorf("otel.endpoint is required")
		}
		endpoint, err := validateOtelEndpoint(cfg.Otel.Endpoint)
		if err != nil {
			return nil, err
		}
		cfg.Otel.Endpoint = endpoint
		if cfg.Otel.FlushIntervalMs == 0 {
			cfg.Otel.FlushIntervalMs = defaultOtelFlushIntervalMs
		} else if cfg.Otel.FlushIntervalMs < 0 || cfg.Otel.FlushIntervalMs > maxOtelFlushIntervalMs {
			return nil, fmt.Errorf("otel.flushIntervalMs must be between 1 and %d", maxOtelFlushIntervalMs)
		}
		if cfg.Otel.MaxBatchSize == 0 {
			cfg.Otel.MaxBatchSize = defaultOtelMaxBatchSize
		} else if cfg.Otel.MaxBatchSize < 0 || cfg.Otel.MaxBatchSize > maxOtelBatchSize {
			return nil, fmt.Errorf("otel.maxBatchSize must be between 1 and %d", maxOtelBatchSize)
		}
		if cfg.Otel.MaxQueueSize == 0 {
			cfg.Otel.MaxQueueSize = defaultOtelMaxQueueSize
		} else if cfg.Otel.MaxQueueSize < 0 || cfg.Otel.MaxQueueSize > maxOtelQueueSize {
			return nil, fmt.Errorf("otel.maxQueueSize must be between 1 and %d", maxOtelQueueSize)
		}
		if cfg.Otel.MaxBatchSize > cfg.Otel.MaxQueueSize {
			return nil, fmt.Errorf("otel.maxBatchSize must not exceed otel.maxQueueSize")
		}
	}

	return cfg, nil
}

func validateOtelEndpoint(rawEndpoint string) (string, error) {
	endpoint, err := url.Parse(rawEndpoint)
	if err != nil || endpoint.Host == "" {
		return "", fmt.Errorf("otel.endpoint must be an absolute HTTP(S) URL")
	}
	if endpoint.Scheme != "http" && endpoint.Scheme != "https" {
		return "", fmt.Errorf("otel.endpoint must use http or https")
	}
	if endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return "", fmt.Errorf("otel.endpoint must not contain credentials, a query, or a fragment")
	}
	return strings.TrimRight(endpoint.String(), "/"), nil
}
