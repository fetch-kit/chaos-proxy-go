package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"chaos-proxy-go/internal/config"
	"chaos-proxy-go/internal/middleware"

	"github.com/go-chi/chi/v5"
)

const maxReloadBodyBytes = 1024 * 1024

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

// runtimeState holds the resolved middleware router and config for one config version.
type runtimeState struct {
	cfg     *config.Config
	router  *chi.Mux
	version int
}

// ReloadResult is returned by ReloadConfig.
type ReloadResult struct {
	OK       bool   `json:"ok"`
	Version  int    `json:"version"`
	ReloadMs int64  `json:"reload_ms"`
	Error    string `json:"error,omitempty"`
}

// Server represents the chaos proxy server
type Server struct {
	httpServer *http.Server
	// router is the top-level handler that dispatches to the active snapshot.
	// Exposed for httptest.NewServer compatibility.
	router      http.Handler
	state       atomic.Pointer[runtimeState]
	reloadMu    sync.Mutex
	isReloading bool
	verbose     bool
	registry    *middleware.Registry
}

// New creates a new proxy server
func New(cfg *config.Config, verbose bool) (*Server, error) {
	registry := middleware.DefaultRegistry

	server := &Server{
		verbose:  verbose,
		registry: registry,
	}

	state, err := server.buildState(cfg, 1)
	if err != nil {
		return nil, err
	}
	server.state.Store(state)

	// Top-level handler: captures snapshot at request-entry for deterministic in-flight behavior.
	topLevel := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle reload endpoint before snapshot dispatch.
		if r.URL.Path == "/reload" && r.Method == http.MethodPost {
			server.handleReload(w, r)
			return
		}
		// Capture snapshot — immune to any reload that happens during this request.
		snap := server.state.Load()
		snap.router.ServeHTTP(w, r)
	})

	server.router = topLevel
	server.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: topLevel,
	}

	return server, nil
}

// buildState validates config and resolves a full middleware router.
func (s *Server) buildState(cfg *config.Config, version int) (*runtimeState, error) {
	router := chi.NewRouter()

	// Apply global middlewares
	for _, middlewareMap := range cfg.Global {
		for name, mcfg := range middlewareMap {
			handler, err := s.registry.Create(name, mcfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create global middleware %s: %w", name, err)
			}
			router.Use(handler)
		}
	}

	// Setup route-specific middlewares
	for route, middlewares := range cfg.Routes {
		proxyHandler := s.createProxyHandler(cfg.Target)

		for i := len(middlewares) - 1; i >= 0; i-- {
			middlewareMap := middlewares[i]
			for name, mcfg := range middlewareMap {
				handler, err := s.registry.Create(name, mcfg)
				if err != nil {
					return nil, fmt.Errorf("failed to create route middleware %s for route %s: %w", name, route, err)
				}
				proxyHandler = handler(proxyHandler)
			}
		}

		method, path := parseRoute(route)
		if method == "" {
			router.HandleFunc(path, proxyHandler.ServeHTTP)
		} else {
			router.Method(method, path, proxyHandler)
		}
	}

	// Default catch-all proxy handler
	router.HandleFunc("/*", s.createProxyHandler(cfg.Target).ServeHTTP)

	return &runtimeState{cfg: cfg, router: router, version: version}, nil
}

// ReloadConfig validates newCfg, builds a new runtime state, and atomically swaps it.
// All-or-nothing: on any error the active state is unchanged.
func (s *Server) ReloadConfig(newCfg *config.Config) ReloadResult {
	s.reloadMu.Lock()
	if s.isReloading {
		v := s.state.Load().version
		s.reloadMu.Unlock()
		return ReloadResult{OK: false, Error: "reload already in progress", Version: v}
	}
	s.isReloading = true
	s.reloadMu.Unlock()

	start := time.Now()
	defer func() {
		s.reloadMu.Lock()
		s.isReloading = false
		s.reloadMu.Unlock()
	}()

	current := s.state.Load()
	nextState, err := s.buildState(newCfg, current.version+1)
	if err != nil {
		return ReloadResult{
			OK:       false,
			Error:    err.Error(),
			Version:  current.version,
			ReloadMs: time.Since(start).Milliseconds(),
		}
	}

	s.state.Store(nextState)
	return ReloadResult{
		OK:       true,
		Version:  nextState.version,
		ReloadMs: time.Since(start).Milliseconds(),
	}
}

// handleReload handles POST /reload requests.
func (s *Server) handleReload(w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(ct), "application/json") {
		writeJSON(w, http.StatusUnsupportedMediaType, ReloadResult{
			OK:      false,
			Error:   "reload endpoint expects Content-Type: application/json",
			Version: s.state.Load().version,
		})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxReloadBodyBytes+1))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ReloadResult{
			OK:      false,
			Error:   "failed to read request body",
			Version: s.state.Load().version,
		})
		return
	}
	if len(body) > maxReloadBodyBytes {
		writeJSON(w, http.StatusBadRequest, ReloadResult{
			OK:      false,
			Error:   fmt.Sprintf("reload payload too large (max %d bytes)", maxReloadBodyBytes),
			Version: s.state.Load().version,
		})
		return
	}

	newCfg, err := config.ParseJSON(body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ReloadResult{
			OK:      false,
			Error:   err.Error(),
			Version: s.state.Load().version,
		})
		return
	}

	result := s.ReloadConfig(newCfg)
	status := http.StatusOK
	if !result.OK {
		if strings.Contains(result.Error, "already in progress") {
			status = http.StatusConflict
		} else {
			status = http.StatusBadRequest
		}
	}
	writeJSON(w, status, result)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// parseRoute parses method and path from route string
func parseRoute(route string) (method, path string) {
	parts := strings.SplitN(route, " ", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", route
}

// createProxyHandler creates the proxy handler for a given target.
func (s *Server) createProxyHandler(target string) http.Handler {
	targetURL, err := url.Parse(target)
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Invalid target URL %q: %v", target, err)
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("Internal proxy error"))
		})
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = http.DefaultTransport
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.verbose {
			log.Printf("[%s] %s %s", r.RemoteAddr, r.Method, r.RequestURI)
		}

		start := time.Now()

		var lw *loggingResponseWriter
		if s.verbose {
			lw = &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			w = lw
		}

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
	cfg := s.state.Load().cfg
	if s.verbose {
		log.Printf("Starting chaos proxy on %s, forwarding to %s", s.httpServer.Addr, cfg.Target)
	}
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server without interrupting active connections.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
