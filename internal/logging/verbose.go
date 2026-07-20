// Package logging provides a lightweight structured verbose logging utility
// with sensitive-data redaction, mirroring the chaos-proxy (TypeScript) design.
package logging

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Level is the severity of a verbose log line.
type Level string

// Verbose severity levels for structured log lines.
const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

var traceparentRe = regexp.MustCompile(`(?i)^[\da-f]{2}-([\da-f]{32})-[\da-f]{16}-[\da-f]{2}$`)

// sensitiveQueryKeys are query-string keys whose values are redacted.
var sensitiveQueryKeys = map[string]struct{}{
	"token":         {},
	"secret":        {},
	"password":      {},
	"apikey":        {},
	"api_key":       {},
	"access_token":  {},
	"refresh_token": {},
}

// sanitizeControlChars replaces control characters (and DEL) with spaces so that
// untrusted input cannot inject newlines or terminal escape sequences into logs.
func sanitizeControlChars(input string) string {
	var b strings.Builder
	b.Grow(len(input))
	for _, r := range input {
		if r <= 31 || r == 127 {
			b.WriteRune(' ')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// formatFieldValue renders a single field value for key=value output, quoting
// when the value contains whitespace, '=' or '"'.
func formatFieldValue(value any) string {
	switch v := value.(type) {
	case nil:
		return "null"
	case bool:
		return fmt.Sprintf("%t", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		// Match Go's default numeric rendering without trailing noise.
		return fmt.Sprintf("%v", v)
	}

	var asString string
	if s, ok := value.(string); ok {
		asString = sanitizeControlChars(s)
	} else {
		encoded, err := json.Marshal(value)
		if err != nil {
			asString = sanitizeControlChars(fmt.Sprintf("%v", value))
		} else {
			asString = sanitizeControlChars(string(encoded))
		}
	}

	if asString == "" {
		return `""`
	}
	if strings.ContainsAny(asString, " \t=\"") {
		quoted, _ := json.Marshal(asString)
		return string(quoted)
	}
	return asString
}

// RedactURLQuery sanitizes a URL path and redacts sensitive query-string values.
func RedactURLQuery(urlPath string) string {
	qIndex := strings.IndexByte(urlPath, '?')
	if qIndex == -1 {
		return sanitizeControlChars(urlPath)
	}

	path := urlPath[:qIndex]
	rawQuery := urlPath[qIndex+1:]

	params, err := url.ParseQuery(rawQuery)
	if err != nil {
		return sanitizeControlChars(urlPath)
	}

	for key := range params {
		if _, ok := sensitiveQueryKeys[strings.ToLower(key)]; ok {
			params.Set(key, "[REDACTED]")
		}
	}

	redactedQuery := params.Encode()
	if redactedQuery == "" {
		return sanitizeControlChars(path)
	}
	return sanitizeControlChars(path + "?" + redactedQuery)
}

// CreateRequestID returns a short random request identifier such as "rq_1a2b3c4d".
func CreateRequestID() string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		for i := range buf {
			buf[i] = byte(i)
		}
	}
	return fmt.Sprintf("rq_%x", buf)
}

// ExtractTraceID returns the 32-hex trace-id from a W3C traceparent header, or "".
func ExtractTraceID(headers http.Header) string {
	traceparent := headers.Get("Traceparent")
	if traceparent == "" {
		return ""
	}
	m := traceparentRe.FindStringSubmatch(traceparent)
	if m == nil {
		return ""
	}
	return m[1]
}

// EmitVerbose writes a structured key=value log line when enabled. INFO/DEBUG go
// to stdout; WARN/ERROR go to stderr. Field keys are emitted in a stable order.
func EmitVerbose(enabled bool, event string, fields map[string]any, level Level) {
	if !enabled {
		return
	}
	if level == "" {
		level = LevelInfo
	}

	var b strings.Builder
	b.WriteString("ts=")
	b.WriteString(formatFieldValue(time.Now().UTC().Format(time.RFC3339Nano)))
	b.WriteString(" level=")
	b.WriteString(formatFieldValue(string(level)))
	b.WriteString(" event=")
	b.WriteString(formatFieldValue(event))

	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := fields[k]
		if v == nil {
			// Preserve explicit nulls; skip only truly absent keys (not in map).
			b.WriteByte(' ')
			b.WriteString(k)
			b.WriteString("=null")
			continue
		}
		b.WriteByte(' ')
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(formatFieldValue(v))
	}

	line := b.String()
	if level == LevelWarn || level == LevelError {
		_, _ = fmt.Fprintln(Err, line)
		return
	}
	_, _ = fmt.Fprintln(Out, line)
}
