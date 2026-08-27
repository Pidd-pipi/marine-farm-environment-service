package httpapi

import (
	"net/http"

	"example.com/marine-farm-environment-service/service"
)

// FarmLogHandler serves daily farm logs.
type FarmLogHandler struct {
	svc *service.Services
}

// NewFarmLogHandler builds the farm-log handler.
func NewFarmLogHandler(svc *service.Services) *FarmLogHandler {
	return &FarmLogHandler{svc: svc}
}

// createFarmLogRequest is the POST /api/logs payload.
type createFarmLogRequest struct {
	ZoneID      string  `json:"zone_id"`
	Date        string  `json:"date"`
	FeedAmount  float64 `json:"feed_amount"`
	DeathCount  int     `json:"death_count"`
	DiseaseNote string  `json:"disease_note"`
}

// Create handles POST /api/logs: records one daily farm log with automatic
// death-abnormal prompting.
func (h *FarmLogHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createFarmLogRequest
	if err := decodeJSON(w, r, &req); err != nil {
		Err(w, r, err)
		return
	}
	log, err := h.svc.FarmLog.Create(service.FarmLogRequest{
		ZoneID:      req.ZoneID,
		Date:        req.Date,
		FeedAmount:  req.FeedAmount,
		DeathCount:  req.DeathCount,
		DiseaseNote: req.DiseaseNote,
		Operator:    operatorName(r),
		RequestID:   requestIDFrom(r),
	})
	if err != nil {
		Err(w, r, err)
		return
	}
	Created(w, r, log)
}

// List handles GET /api/logs?zone_id=&limit=&offset=.
func (h *FarmLogHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parseLimitOffset(r, 100, 1000)
	if err != nil {
		Err(w, r, err)
		return
	}
	zoneID := r.URL.Query().Get("zone_id")
	page, total := paginate(h.svc.FarmLog.List(zoneID, 0), offset, limit)
	setListHeaders(w, limit, offset, total)
	OK(w, r, page)
}

// Get handles GET /api/logs/{id}.
func (h *FarmLogHandler) Get(w http.ResponseWriter, r *http.Request) {
	log, err := h.svc.FarmLog.Get(pathValue(r, "id"))
	if err != nil {
		Err(w, r, err)
		return
	}
	OK(w, r, log)
}
