package httpapi

import (
	"net/http"
	"time"

	"example.com/marine-farm-environment-service/store"
)

// HealthHandler serves the readiness probe.
type HealthHandler struct {
	store *store.Store
}

// NewHealthHandler builds the health handler.
func NewHealthHandler(st *store.Store) *HealthHandler {
	return &HealthHandler{store: st}
}

// Healthz responds 200 with a short service summary (liveness probe).
func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	OK(w, r, map[string]interface{}{
		"status":   "ok",
		"service":  "marine-farm-environment-service",
		"time":     time.Now().UTC().Format(time.RFC3339),
		"entities": h.store.Count(),
	})
}

// Readyz responds 200 once the store is loaded and the service can serve
// traffic (readiness probe).
func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	OK(w, r, map[string]interface{}{
		"status":  "ready",
		"service": "marine-farm-environment-service",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}
