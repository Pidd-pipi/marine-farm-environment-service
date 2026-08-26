// Package middleware provides HTTP cross-cutting concerns: trace-id
// injection, unified panic/error handling and operation audit logging.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

type ctxKey int

const (
	ctxKeyRequestID ctxKey = iota
	ctxKeyStartTime
)

// RequestID returns the trace id stored in the context ("" when absent).
func RequestID(ctx context.Context) string {
	if id, ok := ctx.Value(ctxKeyRequestID).(string); ok {
		return id
	}
	return ""
}

// StartTime returns the request start time stored by RequestIDMiddleware.
func StartTime(ctx context.Context) time.Time {
	if t, ok := ctx.Value(ctxKeyStartTime).(time.Time); ok {
		return t
	}
	return time.Time{}
}

// WithRequestID returns a context carrying the given trace id.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID, id)
}

// RequestIDMiddleware injects a trace id into every request. An incoming
// X-Request-Id header is honoured when present.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = newRequestID()
		}
		ctx := context.WithValue(r.Context(), ctxKeyRequestID, id)
		ctx = context.WithValue(ctx, ctxKeyStartTime, time.Now())
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// newRequestID generates a short random hex trace id.
func newRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "req-" + hex.EncodeToString([]byte(time.Now().Format("150405.000000000")))
	}
	return "req-" + hex.EncodeToString(b)
}
