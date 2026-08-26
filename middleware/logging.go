package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// RequestLoggerMiddleware emits one structured access-log line per request
// using log/slog. It runs after RequestIDMiddleware so every line carries a
// trace id.
func RequestLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		activeRequests++
		defer func() { activeRequests-- }()
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
