package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"chaos-proxy-go/internal/config"
	"chaos-proxy-go/internal/middleware"

	"github.com/go-chi/chi/v5"
)

// Server represents the chaos proxy server
type Server struct {
	router   *chi.Mux
	config   *config.Config
	verbose  bool
	registry *middleware.Registry
}

// New creates a new proxy server
func New(cfg *config.Config, verbose bool) *Server {
	router := chi.NewRouter()
	registry := middleware.DefaultRegistry

	server := &Server{
		router:   router,
		config:   cfg,
		verbose:  verbose,
		registry: registry,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures the middleware chain and proxy routes
func (s *Server) setupRoutes() {
	// Apply global middlewares
	for _, middlewareMap := range s.config.Global {
		for name, config := range middlewareMap {
			handler, err := s.registry.Create(name, config)
			if err != nil {
				log.Fatalf("Failed to create global middleware %s: %v", name, err)
			}
			s.router.Use(handler)
			if s.verbose {
				log.Printf("Applied global middleware: %s", name)
			}
		}
	}

	// Setup route-specific middlewares
	for route, middlewares := range s.config.Routes {
		var routeHandler = s.createProxyHandler()

		// Apply middlewares in reverse order (last one wraps first)
		for i := len(middlewares) - 1; i >= 0; i-- {
			middlewareMap := middlewares[i]
			for name, config := range middlewareMap {
				handler, err := s.registry.Create(name, config)
				if err != nil {
					log.Fatalf("Failed to create route middleware %s for route %s: %v", name, route, err)
				}
				routeHandler = handler(routeHandler)
				if s.verbose {
					log.Printf("Applied middleware %s to route: %s", name, route)
				}
			}
		}

		// Parse method and path from route
		method, path := s.parseRoute(route)
		if method == "" {
			// No method specified, apply to all methods
			s.router.HandleFunc(path, routeHandler.ServeHTTP)
		} else {
			s.router.Method(method, path, routeHandler)
		}
	}

	// Default catch-all proxy handler for routes without specific middlewares
	s.router.HandleFunc("/*", s.createProxyHandler().ServeHTTP)
}

// parseRoute parses method and path from route string
func (s *Server) parseRoute(route string) (method, path string) {
	parts := strings.SplitN(route, " ", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", route
}

// createProxyHandler creates the proxy handler
func (s *Server) createProxyHandler() http.Handler {
	targetURL, _ := url.Parse(s.config.Target)
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	// Use default http.Transport for best performance (unless skip-verify is needed)
	proxy.Transport = http.DefaultTransport
	// No custom Director or error handler
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Minimal mutation: only set Host if needed
		r.Host = targetURL.Host
		proxy.ServeHTTP(w, r)
	})
}

// Start starts the proxy server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Port)
	if s.verbose {
		log.Printf("Starting chaos proxy on %s, forwarding to %s", addr, s.config.Target)
	}
	return http.ListenAndServe(addr, s.router)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	// Chi doesn't have a built-in shutdown method, would need http.Server for graceful shutdown
	return nil
}
