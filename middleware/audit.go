package middleware

import (
	"net/http"
	"strings"
	"time"

	"example.com/marine-farm-environment-service/domain"
	"example.com/marine-farm-environment-service/store"
)

// AuditLogger is the cross-cutting request-audit middleware. Every request
// produces an http.request audit entry carrying method, path, status and
// latency.
type AuditLogger struct {
	store *store.Store
	// skipPaths lists prefixes that are not audited (health checks, static
	// assets) to keep the audit trail focused on business operations.
	skipPaths []string
	// maxEntries caps the audit trail size.
	maxEntries int
	// counts tracks how many requests hit each path.
	counts map[string]int
}

// NewAuditLogger builds the audit middleware.
func NewAuditLogger(st *store.Store, maxEntries int, skipPaths ...string) *AuditLogger {
	return &AuditLogger{store: st, skipPaths: skipPaths, maxEntries: maxEntries, counts: map[string]int{}}
}

// Wrap returns the middleware handler.
func (m *AuditLogger) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := StartTime(r.Context())
		if start.IsZero() {
			start = time.Now()
		}
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		path := r.URL.Path
		for _, skip := range m.skipPaths {
			if skip != "" && strings.HasPrefix(path, skip) {
				return
			}
		}
		latency := time.Since(start).Milliseconds()
		m.counts[path]++
		entry := domain.NewAuditEntry(
			m.store.NewID("audit"),
			domain.AuditHTTPRequest,
			"http",
			path,
			"anonymous",
			detailLine(r.Method, path, rec.status, int(latency)),
			time.Now().UTC(),
		)
		entry.RequestID = RequestID(r.Context())
		_ = m.store.Audit().Create(entry, m.maxEntries)
	})
}

// statusRecorder captures the response status code for auditing.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader captures the status.
func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Write records implicit 200 responses.
func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

func detailLine(method, path string, status, latencyMS int) string {
	return method + " " + path + " -> " + itoa(status) + " (" + itoa(latencyMS) + "ms)"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := make([]byte, 0, 8)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
