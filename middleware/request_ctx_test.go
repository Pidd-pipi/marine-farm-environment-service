package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRequestIDMiddlewareCancellationPropagation pins the context contract:
// a cancelled request must stay cancelled inside the handler.
func TestRequestIDMiddlewareCancellationPropagation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)

	var got context.Context
	h := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Context()
	}))
	h.ServeHTTP(httptest.NewRecorder(), req)

	if got.Err() == nil {
		t.Fatalf("request cancellation was lost by the middleware")
	}
}

// TestRequestIDMiddlewarePreservesID pins the value contract: the injected
// request id must still be readable inside the handler.
func TestRequestIDMiddlewarePreservesID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "trace-abc")

	var got context.Context
	h := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Context()
	}))
	h.ServeHTTP(httptest.NewRecorder(), req)

	if RequestID(got) != "trace-abc" {
		t.Fatalf("request id lost: %q", RequestID(got))
	}
}

// TestWithRequestIDPreservesContext pins the helper contract: attaching an
// id must not drop the parent context's cancellation.
func TestWithRequestIDPreservesContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	derived := WithRequestID(ctx, "trace-abc")
	if derived.Err() == nil {
		t.Fatalf("WithRequestID dropped the parent cancellation")
	}
}
