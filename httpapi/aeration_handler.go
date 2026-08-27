package httpapi

import (
	"net/http"

	"example.com/marine-farm-environment-service/service"

	"example.com/marine-farm-environment-service/domain"
)

// AerationHandler serves aerator control: manual start/stop, restore
// confirmation and device feedback.
type AerationHandler struct {
	svc *service.Services
}

// NewAerationHandler builds the aeration handler.
func NewAerationHandler(svc *service.Services) *AerationHandler {
	return &AerationHandler{svc: svc}
}

// Start handles POST /api/zones/{id}/aerate: manual aeration start.
func (h *AerationHandler) Start(w http.ResponseWriter, r *http.Request) {
	log, err := h.svc.Aeration.Start(pathValue(r, "id"), domain.TriggerManual, operatorName(r), requestIDFrom(r))
	if err != nil {
		Err(w, r, err)
		return
	}
	Created(w, r, log)
}

// Stop handles POST /api/zones/{id}/stop-aeration: manual aerator stop.
func (h *AerationHandler) Stop(w http.ResponseWriter, r *http.Request) {
	log, err := h.svc.Aeration.Stop(pathValue(r, "id"), domain.TriggerManual, operatorName(r), requestIDFrom(r))
	if err != nil {
		Err(w, r, err)
		return
	}
	OK(w, r, log)
}

// Restore handles POST /api/zones/{id}/restore: restore confirmation. Only
// allowed when the restore checker marked the zone restore-eligible.
func (h *AerationHandler) Restore(w http.ResponseWriter, r *http.Request) {
	log, err := h.svc.Aeration.Restore(pathValue(r, "id"), operatorName(r), requestIDFrom(r))
	if err != nil {
		Err(w, r, err)
		return
	}
	OK(w, r, log)
}

// feedbackRequest is the POST /api/aeration/{id}/feedback payload.
type feedbackRequest struct {
	Feedback string `json:"feedback"`
}

// Feedback handles POST /api/aeration/{id}/feedback: device feedback for
// an aerator command (acknowledged/started/stopped/fault).
func (h *AerationHandler) Feedback(w http.ResponseWriter, r *http.Request) {
	var req feedbackRequest
	if err := decodeJSON(w, r, &req); err != nil {
		Err(w, r, err)
		return
	}
	fb := domain.FeedbackStatus(req.Feedback)
	if !fb.Valid() {
		Err(w, r, domain.InvalidInput("invalid feedback %q", req.Feedback))
		return
	}
	log, err := h.svc.Aeration.Feedback(pathValue(r, "id"), domain.FeedbackAcknowledged, operatorName(r), requestIDFrom(r))
	if err != nil {
		Err(w, r, err)
		return
	}
	OK(w, r, log)
}

// List handles GET /api/aeration?limit=&offset=.
func (h *AerationHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parseLimitOffset(r, 100, 1000)
	if err != nil {
		Err(w, r, err)
		return
	}
	page, total := paginate(h.svc.Aeration.List(0), offset, limit)
	setListHeaders(w, limit, offset, total)
	OK(w, r, page)
}
