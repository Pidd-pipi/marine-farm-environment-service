package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"example.com/marine-farm-environment-service/store"
)

// TestAuditLoggerConcurrentNoRace exercises the audit middleware under
// concurrent requests; the race detector must report no shared-state race.
func TestAuditLoggerConcurrentNoRace(t *testing.T) {
	audit := NewAuditLogger(store.NewMemoryStore(), 20000, "/healthz")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := audit.Wrap(next)

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 60; j++ {
				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/zones", nil))
			}
		}()
	}
	close(start)
	wg.Wait()
}

// TestRequestLoggerConcurrentNoRace exercises the access logger under
// concurrent requests; the race detector must report no shared-state race.
func TestRequestLoggerConcurrentNoRace(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := RequestLoggerMiddleware(next)

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 60; j++ {
				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/zones", nil))
			}
		}()
	}
	close(start)
	wg.Wait()
}
