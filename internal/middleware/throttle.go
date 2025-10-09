package middleware

import (
	"net/http"
	"sync"
	"time"
)

// ThrottleConfig is the configuration for the Throttle middleware.
type ThrottleConfig struct {
	Rate      int    `yaml:"rate"`
	ChunkSize int    `yaml:"chunkSize"`
	Burst     int    `yaml:"burst"`
	Key       string `yaml:"key"`
}

type throttleEntry struct {
	BurstLeft int
}

type throttleStore struct {
	mu    sync.Mutex
	store map[string]*throttleEntry
}

func newThrottleStore() *throttleStore {
	return &throttleStore{store: make(map[string]*throttleEntry)}
}

func (s *throttleStore) get(key string, burst int) *throttleEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.store[key]
	if !ok {
		entry = &throttleEntry{BurstLeft: burst}
		s.store[key] = entry
	}
	return entry
}

// ThrottleMiddleware returns a middleware that throttles response bandwidth.
func ThrottleMiddleware(config ThrottleConfig) func(http.Handler) http.Handler {
	if config.Rate <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	chunkSize := config.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 16384
	}
	burst := config.Burst
	store := newThrottleStore()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.RemoteAddr
			if config.Key != "" {
				if v := r.Header.Get(config.Key); v != "" {
					key = v
				}
			}
			entry := store.get(key, burst)
			tw := &throttleWriter{
				ResponseWriter: w,
				rate:           config.Rate,
				chunkSize:      chunkSize,
				burstLeft:      &entry.BurstLeft,
			}
			next.ServeHTTP(tw, r)
		})
	}
}

type throttleWriter struct {
	http.ResponseWriter
	rate      int
	chunkSize int
	burstLeft *int
}

func (tw *throttleWriter) Write(data []byte) (int, error) {
	total := 0
	offset := 0
	for offset < len(data) {
		toSend := tw.chunkSize
		if offset+toSend > len(data) {
			toSend = len(data) - offset
		}
		// Handle burst
		if tw.burstLeft != nil && *tw.burstLeft > 0 {
			burstSend := toSend
			if *tw.burstLeft < toSend {
				burstSend = *tw.burstLeft
			}
			n, _ := tw.ResponseWriter.Write(data[offset : offset+burstSend])
			total += n
			offset += burstSend
			*tw.burstLeft -= burstSend
			if burstSend < toSend {
				toSend -= burstSend
			} else {
				continue
			}
		}
		// Throttle
		if toSend > 0 {
			start := time.Now()
			n, err := tw.ResponseWriter.Write(data[offset : offset+toSend])
			total += n
			offset += toSend
			elapsed := time.Since(start)
			expected := float64(toSend) / float64(tw.rate)
			delay := time.Duration(expected*1000)*time.Millisecond - elapsed
			if delay > 0 {
				time.Sleep(delay)
			}
			if err != nil {
				return total, err
			}
		}
	}
	return total, nil
}
