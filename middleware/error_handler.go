package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// panicBody is the unified JSON body written for recovered panics.
type panicBody struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Error     string `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

// PanicRecoveryMiddleware converts panics into unified 500 responses so a
// single bad request cannot take the whole service down.
func PanicRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "panic", rec, "stack", string(debug.Stack()))
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(panicBody{
					Code:      500,
					Message:   "internal server error",
					Error:     "panic",
					RequestID: RequestID(r.Context()),
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
