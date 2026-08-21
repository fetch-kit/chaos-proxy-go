package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func FuzzFailFirstNSchedule(f *testing.F) {
	f.Add(uint8(2), uint8(5))
	f.Add(uint8(0), uint8(1))
	f.Add(uint8(20), uint8(20))

	f.Fuzz(func(t *testing.T, rawN, rawRequests uint8) {
		n := int(rawN % 33)
		requests := int(rawRequests % 65)
		passed := 0
		handler := FailFirstNMiddleware(FailFirstNConfig{N: n, Status: http.StatusServiceUnavailable})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			passed++
			w.WriteHeader(http.StatusNoContent)
		}))

		for i := 0; i < requests; i++ {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
			want := http.StatusNoContent
			if i < n {
				want = http.StatusServiceUnavailable
			}
			if recorder.Code != want {
				t.Fatalf("request %d: got status %d, want %d (n=%d)", i+1, recorder.Code, want, n)
			}
		}
		wantPassed := requests - min(n, requests)
		if passed != wantPassed {
			t.Fatalf("downstream called %d times, want %d", passed, wantPassed)
		}
	})
}

func FuzzFailNthSchedule(f *testing.F) {
	f.Add(uint8(3), uint8(9))
	f.Add(uint8(1), uint8(1))
	f.Add(uint8(20), uint8(40))

	f.Fuzz(func(t *testing.T, rawN, rawRequests uint8) {
		n := int(rawN%32) + 1
		requests := int(rawRequests % 65)
		passed := 0
		handler := FailNthMiddleware(FailNthConfig{N: n, Status: http.StatusBadGateway})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			passed++
			w.WriteHeader(http.StatusNoContent)
		}))

		for i := 1; i <= requests; i++ {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
			want := http.StatusNoContent
			if i%n == 0 {
				want = http.StatusBadGateway
			}
			if recorder.Code != want {
				t.Fatalf("request %d: got status %d, want %d (n=%d)", i, recorder.Code, want, n)
			}
		}
		wantPassed := requests - requests/n
		if passed != wantPassed {
			t.Fatalf("downstream called %d times, want %d", passed, wantPassed)
		}
	})
}

func FuzzThrottlePreservesBytes(f *testing.F) {
	f.Add([]byte("hello"), uint8(2), uint16(0))
	f.Add([]byte{}, uint8(1), uint16(0))
	f.Add([]byte{0, 1, 2, 255}, uint8(3), uint16(2))

	f.Fuzz(func(t *testing.T, data []byte, rawChunkSize uint8, rawBurst uint16) {
		if len(data) > 4096 {
			t.Skip()
		}
		chunkSize := int(rawChunkSize%64) + 1
		burst := int(rawBurst)
		if burst > len(data) {
			burst = len(data)
		}
		handler := ThrottleMiddleware(ThrottleConfig{
			Rate:      int(^uint(0) >> 1),
			ChunkSize: chunkSize,
			Burst:     burst,
		})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			for offset := 0; offset < len(data); {
				end := offset + chunkSize
				if end > len(data) {
					end = len(data)
				}
				if _, err := w.Write(data[offset:end]); err != nil {
					t.Fatalf("downstream write failed: %v", err)
				}
				offset = end
			}
		}))

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		if !bytes.Equal(recorder.Body.Bytes(), data) {
			t.Fatalf("payload changed: got %x, want %x", recorder.Body.Bytes(), data)
		}
	})
}
