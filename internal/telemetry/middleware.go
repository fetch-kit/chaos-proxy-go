package telemetry

import (
	"net/http"
	"time"

	"chaos-proxy-go/internal/config"
)

// statusCapturingWriter wraps http.ResponseWriter to capture the written status code.
type statusCapturingWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (w *statusCapturingWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		w.written = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusCapturingWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.statusCode = http.StatusOK
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}

// NewMiddleware returns an http middleware that traces each request via OTLP.
func NewMiddleware(cfg config.OtelConfig, exporter *OtlpExporter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tc := ExtractTraceContext(r)
			if tc == nil {
				tc = NewTraceContext()
			}

			// Build the full request URL for the span
			rawURL := r.URL.String()
			if r.Host != "" {
				scheme := "http"
				rawURL = scheme + "://" + r.Host + r.RequestURI
			}

			span := NewSpan(tc, r.Method, rawURL, cfg.ServiceName)

			// Inject traceparent with our span's ID so downstream sees a valid context.
			InjectTraceContext(r, &TraceContext{
				TraceID:    tc.TraceID,
				SpanID:     span.SpanID,
				TraceFlags: tc.TraceFlags,
			})

			sw := &statusCapturingWriter{ResponseWriter: w, statusCode: http.StatusOK}

			defer func() {
				span.EndTime = time.Now()
				span.Status = sw.statusCode
				span.Error = sw.statusCode >= 400
				exporter.AddSpan(span)
			}()

			next.ServeHTTP(sw, r)
		})
	}
}
