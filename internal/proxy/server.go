package proxy

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"chaos-proxy-go/internal/config"
	"chaos-proxy-go/internal/middleware"

	"github.com/go-chi/chi/v5"
)

// loggingResponseWriter wraps http.ResponseWriter to capture the status code
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (lw *loggingResponseWriter) WriteHeader(code int) {
	if !lw.written {
		lw.statusCode = code
		lw.written = true
	}
	lw.ResponseWriter.WriteHeader(code)
}

func (lw *loggingResponseWriter) Write(b []byte) (int, error) {
	if !lw.written {
		lw.statusCode = http.StatusOK
		lw.written = true
	}
	return lw.ResponseWriter.Write(b)
}

// Server represents the chaos proxy server
type Server struct {
	httpServer *http.Server
	router     *chi.Mux
	config     *config.Config
	verbose    bool
	registry   *middleware.Registry
}

// New creates a new proxy server
func New(cfg *config.Config, verbose bool) (*Server, error) {
	router := chi.NewRouter()
	registry := middleware.DefaultRegistry

	server := &Server{
		router:   router,
		config:   cfg,
		verbose:  verbose,
		registry: registry,
	}

	if err := server.setupRoutes(); err != nil {
		return nil, err
	}

	server.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}

	return server, nil
}

// setupRoutes configures the middleware chain and proxy routes
func (s *Server) setupRoutes() error {
	// Apply global middlewares
	for _, middlewareMap := range s.config.Global {
		for name, config := range middlewareMap {
			handler, err := s.registry.Create(name, config)
			if err != nil {
				return fmt.Errorf("failed to create global middleware %s: %w", name, err)
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
					return fmt.Errorf("failed to create route middleware %s for route %s: %w", name, route, err)
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
	return nil
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
	targetURL, err := url.Parse(s.config.Target)
	if err != nil {
		// This should never happen since target is validated in New(), but handle gracefully
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Invalid target URL %q: %v", s.config.Target, err)
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("Internal proxy error"))
		})
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	// Use default http.Transport for best performance (unless skip-verify is needed)
	proxy.Transport = http.DefaultTransport
	// No custom Director or error handler
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.verbose {
			log.Printf("[%s] %s %s", r.RemoteAddr, r.Method, r.RequestURI)
		}

		start := time.Now()

		// Wrap response writer to capture status code if verbose logging is enabled
		var lw *loggingResponseWriter
		if s.verbose {
			lw = &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			w = lw
		}

		// Minimal mutation: only set Host if needed
		r.Host = targetURL.Host
		proxy.ServeHTTP(w, r)

		if s.verbose && lw != nil {
			elapsed := time.Since(start)
			log.Printf("[%s] %s %s -> %d (%v)", r.RemoteAddr, r.Method, r.RequestURI, lw.statusCode, elapsed)
		}
	})
}

// Start starts the proxy server
func (s *Server) Start() error {
	if s.verbose {
		log.Printf("Starting chaos proxy on %s, forwarding to %s", s.httpServer.Addr, s.config.Target)
	}
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server without interrupting active connections.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
