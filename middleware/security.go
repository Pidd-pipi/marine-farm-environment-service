package middleware

import (
	"net/http"
	"strings"
)

// SecurityHeadersMiddleware applies a defensive baseline of browser
// security headers to every response. It does not set a
// strict Content-Security-Policy because the frontend uses inline styles
// and ES module scripts; adding a strict CSP would break the SPA.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		// API responses carry business data and must not be cached.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			h.Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}
