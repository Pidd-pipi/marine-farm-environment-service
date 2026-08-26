package httpapi

import (
	"net/http"

	"example.com/marine-farm-environment-service/domain"
	"example.com/marine-farm-environment-service/service"
)

// OverviewHandler serves the dashboard aggregation and the audit trail.
type OverviewHandler struct {
	svc *service.Services
}

// NewOverviewHandler builds the overview handler.
func NewOverviewHandler(svc *service.Services) *OverviewHandler {
	return &OverviewHandler{svc: svc}
}

// Get handles GET /api/overview.
func (h *OverviewHandler) Get(w http.ResponseWriter, r *http.Request) {
	OK(w, r, h.svc.Overview.Get())
}

// Audit handles GET /api/audit?limit=&offset=&target_type=&target_id=.
func (h *OverviewHandler) Audit(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parseLimitOffset(r, 100, 1000)
	if err != nil {
		Err(w, r, err)
		return
	}
	targetType := r.URL.Query().Get("target_type")
	targetID := r.URL.Query().Get("target_id")
	var full []domain.AuditEntry
	if targetType != "" && targetID != "" {
		full = h.svc.Audit.ListByTarget(targetType, targetID, 0)
	} else {
		full = h.svc.Audit.List(0)
	}
	page, total := paginate(full, offset, limit)
	setListHeaders(w, limit, offset, total)
	OK(w, r, page)
}
