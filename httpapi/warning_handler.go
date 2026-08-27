package httpapi

import (
	"net/http"

	"example.com/marine-farm-environment-service/service"

	"example.com/marine-farm-environment-service/domain"
	"example.com/marine-farm-environment-service/store"
)

// WarningHandler serves the warning desk.
type WarningHandler struct {
	svc *service.Services
}

// NewWarningHandler builds the warning handler.
func NewWarningHandler(svc *service.Services) *WarningHandler {
	return &WarningHandler{svc: svc}
}

// List handles GET /api/warnings?status=&zone_id=&type=&limit=&offset=.
func (h *WarningHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parseLimitOffset(r, 100, 1000)
	if err != nil {
		Err(w, r, err)
		return
	}
	q := r.URL.Query()
	filter := store.WarningFilter{
		ZoneID: q.Get("zone_id"),
		Status: q.Get("status"),
		Type:   q.Get("type"),
	}
	page, total := paginate(h.svc.Warnings.List(filter), offset, limit)
	setListHeaders(w, limit, offset, total)
	OK(w, r, page)
}

// Verify handles POST /api/warnings/{id}/verify: confirms a pending
// warning and, for confirmed dangers, triggers aeration automatically.
func (h *WarningHandler) Verify(w http.ResponseWriter, r *http.Request) {
	rec, err := h.svc.Warnings.Verify(pathValue(r, "id"), operatorName(r), requestIDFrom(r))
	if err != nil {
		Err(w, r, err)
		return
	}
	OK(w, r, rec)
}

// Resolve handles POST /api/warnings/{id}/resolve.
func (h *WarningHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	rec, err := h.svc.Warnings.Resolve(pathValue(r, "id"), operatorName(r), requestIDFrom(r))
	if err != nil {
		Err(w, r, err)
		return
	}
	OK(w, r, rec)
}

// domainInvalidTime builds a unified invalid-input error for timestamps.
func domainInvalidTime(v string) error {
	return domain.InvalidInput("invalid timestamp %q, expected RFC3339", v)
}
