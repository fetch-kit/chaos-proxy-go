package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RateLimitConfig is the configuration for the RateLimit middleware.
type RateLimitConfig struct {
	Limit    int    `yaml:"limit"`
	WindowMs int    `yaml:"windowMs"`
	Key      string `yaml:"key"`
}

type rateLimitEntry struct {
	Count     int
	ExpiresAt time.Time
}

type rateLimitStore struct {
	mu    sync.Mutex
	store map[string]*rateLimitEntry
}

func newRateLimitStore() *rateLimitStore {
	return &rateLimitStore{store: make(map[string]*rateLimitEntry)}
}

func (s *rateLimitStore) get(key string) *rateLimitEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.store[key]
	if !ok || time.Now().After(entry.ExpiresAt) {
		entry = &rateLimitEntry{Count: 0, ExpiresAt: time.Now()}
		s.store[key] = entry
	}
	return entry
}

// RateLimitMiddleware returns a middleware that rate-limits requests.
func RateLimitMiddleware(config RateLimitConfig) func(http.Handler) http.Handler {
	if config.Limit <= 0 || config.WindowMs <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	store := newRateLimitStore()
	window := time.Duration(config.WindowMs) * time.Millisecond
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var key string
			if config.Key != "" {
				key = r.Header.Get(config.Key)
				if key == "" {
					key = r.RemoteAddr
				}
			} else {
				key = r.RemoteAddr
			}
			entry := store.get(key)
			store.mu.Lock()
			if time.Now().After(entry.ExpiresAt) {
				entry.Count = 0
				entry.ExpiresAt = time.Now().Add(window)
			}
			entry.Count++
			remaining := config.Limit - entry.Count
			reset := int(time.Until(entry.ExpiresAt).Seconds())
			store.mu.Unlock()
			w.Header().Set("X-RateLimit-Remaining", itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", itoa(reset))
			w.Header().Set("X-RateLimit-Limit", itoa(config.Limit))
			if entry.Count > config.Limit {
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte("Rate limit exceeded"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
