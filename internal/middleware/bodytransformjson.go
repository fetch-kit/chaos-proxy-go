package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// BodyOps defines set/delete operations for JSON body transformation.
type BodyOps struct {
	Set    map[string]interface{} `yaml:"set"`
	Delete []string               `yaml:"delete"`
}

// BodyTransformJSONConfig is the configuration for the BodyTransformJSON middleware.
type BodyTransformJSONConfig struct {
	Request  *BodyOps `yaml:"request"`
	Response *BodyOps `yaml:"response"`
}

// BodyTransformJSONMiddleware returns a middleware that transforms JSON bodies in requests and responses.
func BodyTransformJSONMiddleware(config BodyTransformJSONConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Request body transformation
			if config.Request != nil && strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				var body map[string]interface{}
				data, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "Failed to read body", http.StatusBadRequest)
					return
				}
				_ = r.Body.Close()
				if len(data) > 0 {
					if err := json.Unmarshal(data, &body); err != nil {
						http.Error(w, "Invalid JSON", http.StatusBadRequest)
						return
					}
				} else {
					body = make(map[string]interface{})
				}
				// Set fields
				for k, v := range config.Request.Set {
					body[k] = v
				}
				// Delete fields
				for _, k := range config.Request.Delete {
					delete(body, k)
				}
				newData, err := json.Marshal(body)
				if err != nil {
					http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
					return
				}
				r.Body = io.NopCloser(bytes.NewReader(newData))
				r.ContentLength = int64(len(newData))
				r.Header.Set("Content-Length", strconv.Itoa(len(newData)))
			}

			// Wrap ResponseWriter for response body transformation
			if config.Response != nil {
				rw := &bodyTransformJSONResponseWriter{
					ResponseWriter: w,
					ops:            config.Response,
					buf:            &bytes.Buffer{},
					statusCode:     http.StatusOK,
				}
				next.ServeHTTP(rw, r)

				ct := rw.Header().Get("Content-Type")
				data := rw.buf.Bytes()
				var body map[string]interface{}
				var out []byte
				var err error
				if strings.HasPrefix(ct, "application/json") && json.Unmarshal(data, &body) == nil {
					for k, v := range rw.ops.Set {
						body[k] = v
					}
					for _, k := range rw.ops.Delete {
						delete(body, k)
					}
					out, err = json.Marshal(body)
					if err == nil {
						w.Header().Set("Content-Type", "application/json")
					} else {
						// fallback to original data if marshal fails
						out = data
					}
				} else {
					// not JSON, just pass through
					out = data
				}
				if len(out) > 0 {
					w.Header().Set("Content-Length", strconv.Itoa(len(out)))
				} else {
					w.Header().Del("Content-Length")
				}
				if rw.statusCode != 0 {
					w.WriteHeader(rw.statusCode)
				}
				_, _ = w.Write(out)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type bodyTransformJSONResponseWriter struct {
	http.ResponseWriter
	ops         *BodyOps
	buf         *bytes.Buffer
	statusCode  int
	wroteHeader bool
}

func (w *bodyTransformJSONResponseWriter) WriteHeader(statusCode int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		w.statusCode = statusCode
	}
}

func (w *bodyTransformJSONResponseWriter) Write(data []byte) (int, error) {
	return w.buf.Write(data)
}
