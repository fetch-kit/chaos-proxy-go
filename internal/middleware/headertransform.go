package middleware

import (
	"net/http"
)

// HeaderTransformConfig is the configuration for the HeaderTransform middleware.
type HeaderTransformConfig struct {
	Request  *HeaderOps `yaml:"request"`
	Response *HeaderOps `yaml:"response"`
}

// HeaderOps defines set/delete operations for headers.
type HeaderOps struct {
	Set    map[string]string `yaml:"set"`
	Delete []string          `yaml:"delete"`
}

// HeaderTransformMiddleware returns a middleware that transforms request and response headers.
func HeaderTransformMiddleware(config HeaderTransformConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Request header ops
			if config.Request != nil {
				for k, v := range config.Request.Set {
					r.Header.Set(k, v)
				}
				for _, k := range config.Request.Delete {
					r.Header.Del(k)
				}
			}

			// Wrap ResponseWriter to modify response headers before first write
			if config.Response != nil {
				w = &headerTransformResponseWriter{
					ResponseWriter: w,
					headersToSet:   config.Response.Set,
					headersToDel:   config.Response.Delete,
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

type headerTransformResponseWriter struct {
	http.ResponseWriter
	headersToSet map[string]string
	headersToDel []string
	wroteHeader  bool
}

func (w *headerTransformResponseWriter) WriteHeader(statusCode int) {
	if !w.wroteHeader {
		for k, v := range w.headersToSet {
			w.Header().Set(k, v)
		}
		for _, k := range w.headersToDel {
			w.Header().Del(k)
		}
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *headerTransformResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}
