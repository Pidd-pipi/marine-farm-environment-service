package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestPanicReturns500 pins the panic contract: a panicking handler must be
// converted into a 500 response rather than silently swallowed.
func TestPanicReturns500(t *testing.T) {
	h := PanicRecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("panic status = %d, want 500", rec.Code)
	}
}
