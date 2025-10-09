package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// MiddlewareConfig represents a generic middleware configuration
type MiddlewareConfig[T any] struct {
	Name   string
	Config T
}

// Config represents the main configuration structure
type Config struct {
	Target string                      `yaml:"target"`
	Port   int                         `yaml:"port"`
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

	// Set defaults
	if cfg.Port == 0 {
		cfg.Port = 5000
	}

	// Validate required fields
	if cfg.Target == "" {
		return nil, fmt.Errorf("target is required")
	}

	return &cfg, nil
}
