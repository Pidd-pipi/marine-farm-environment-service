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

// writePanicResponse writes the unified 500 JSON body for a recovered panic.
// It is best-effort: if the connection is already broken or headers were
// already flushed, writing silently fails and we only keep the log line.
func writePanicResponse(w http.ResponseWriter, r *http.Request) {
	body := panicBody{
		Code:      http.StatusInternalServerError,
		Message:   "internal server error",
		Error:     "internal",
		RequestID: RequestID(r.Context()),
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	enc := json.NewEncoder(w)
	_ = enc.Encode(body)
}

// PanicRecoveryMiddleware converts panics into unified 500 responses so a
// single bad request cannot take the whole service down. The recovered
// panic is logged with its stack and a 500 JSON envelope is written back to
// the client; without the explicit write the connection would be left
// dangling (defaulting to an empty 200).
func PanicRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "panic", rec, "stack", string(debug.Stack()))
				writePanicResponse(w, r)
				return
			}
		}()
		next.ServeHTTP(w, r)
	})
}
