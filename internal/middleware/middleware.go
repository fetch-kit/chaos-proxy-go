package middleware

import (
	"fmt"
	"net/http"

	"gopkg.in/yaml.v3"
)

// Factory is a generic middleware factory function.
type Factory[T any] func(config T) func(http.Handler) http.Handler

// Registry represents the middleware registry.
type Registry struct {
	factories map[string]func(config any) (func(http.Handler) http.Handler, error)
}

// DefaultRegistry is the default middleware registry.
var DefaultRegistry *Registry

func init() {
	DefaultRegistry = &Registry{
		factories: make(map[string]func(config any) (func(http.Handler) http.Handler, error)),
	}
	register(DefaultRegistry, "latency", LatencyMiddleware)
	register(DefaultRegistry, "fail", FailMiddleware)
	register(DefaultRegistry, "failNth", FailNthMiddleware)
	register(DefaultRegistry, "failRandomly", FailRandomlyMiddleware)
	register(DefaultRegistry, "latencyRange", LatencyRangeMiddleware)
	register(DefaultRegistry, "cors", CorsMiddleware)
	register(DefaultRegistry, "dropConnection", DropConnectionMiddleware)
	register(DefaultRegistry, "rateLimit", RateLimitMiddleware)
	register(DefaultRegistry, "throttle", ThrottleMiddleware)
	register(DefaultRegistry, "headerTransform", HeaderTransformMiddleware)
	register(DefaultRegistry, "bodyTransformJSON", BodyTransformJSONMiddleware)
}

// register registers a middleware factory with type safety
func register[T any](r *Registry, name string, factory Factory[T]) {
	r.factories[name] = func(config any) (func(http.Handler) http.Handler, error) {
		// Convert map to struct using YAML marshaling
		var typedConfig T
		configBytes, err := yaml.Marshal(config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config for middleware %s: %w", name, err)
		}

		if err := yaml.Unmarshal(configBytes, &typedConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config for middleware %s: %w", name, err)
		}

		return factory(typedConfig), nil
	}
}

// Create creates a middleware instance from config
func (r *Registry) Create(name string, config any) (func(http.Handler) http.Handler, error) {
	factory, exists := r.factories[name]
	if !exists {
		return nil, fmt.Errorf("unknown middleware: %s", name)
	}
	return factory(config)
}
