package middleware

import (
	"net/http"
	"time"
)

// ThrottleConfig is the configuration for the Throttle middleware.
type ThrottleConfig struct {
	Rate      int `yaml:"rate"`
	ChunkSize int `yaml:"chunkSize"`
	Burst     int `yaml:"burst"`
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

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// burstLeft is per-request, so each request gets a fresh burst allowance
			burstLeft := config.Burst
			tw := &throttleWriter{
				ResponseWriter: w,
				rate:           config.Rate,
				chunkSize:      chunkSize,
				burstLeft:      &burstLeft,
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
			delay := time.Duration(expected*float64(time.Second)) - elapsed
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
