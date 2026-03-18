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

// increment atomically retrieves/resets the entry and increments the count,
// returning the current count, remaining, and reset time — all under a single lock.
func (s *rateLimitStore) increment(key string, limit int, window time.Duration) (count, remaining, reset int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	entry, ok := s.store[key]
	if !ok || now.After(entry.ExpiresAt) {
		entry = &rateLimitEntry{Count: 0, ExpiresAt: now.Add(window)}
		s.store[key] = entry
	}
	entry.Count++
	remaining = limit - entry.Count
	if remaining < 0 {
		remaining = 0
	}
	reset = int(time.Until(entry.ExpiresAt).Seconds())
	return entry.Count, remaining, reset
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
			count, remaining, reset := store.increment(key, config.Limit, window)
			w.Header().Set("X-RateLimit-Remaining", itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", itoa(reset))
			w.Header().Set("X-RateLimit-Limit", itoa(config.Limit))
			if count > config.Limit {
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
