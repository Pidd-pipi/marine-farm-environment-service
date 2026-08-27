package middleware

import (
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

// RequestLoggerMiddleware emits one structured access-log line per request
// using log/slog. It runs after RequestIDMiddleware so every line carries a
// trace id.
func RequestLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&activeRequests, 1)
		defer atomic.AddInt64(&activeRequests, -1)
		start := time.Now()
		rec := &logStatusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", RequestID(r.Context()),
			"remote_addr", r.RemoteAddr,
		)
	})
}

var activeRequests int64

// ActiveRequests returns the number of requests currently being served.
// Reads of a concurrently mutated counter must go through the atomic API;
// a plain int64 read would race with the increments in
// RequestLoggerMiddleware and undercount under load.
func ActiveRequests() int64 {
	return atomic.LoadInt64(&activeRequests)
}

type logStatusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *logStatusRecorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *logStatusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}
