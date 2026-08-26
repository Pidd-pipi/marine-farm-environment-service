package httpapi

import (
	"net/http"

	"example.com/marine-farm-environment-service/service"
)

// ZoneHandler serves farm-zone CRUD.
type ZoneHandler struct {
	svc *service.Services
}

// NewZoneHandler builds the zone handler.
func NewZoneHandler(svc *service.Services) *ZoneHandler {
	return &ZoneHandler{svc: svc}
}

// Register wires the zone routes (kept for the Handler contract).
func (h *ZoneHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/zones", h.List)
	mux.HandleFunc("POST /api/zones", h.Create)
	mux.HandleFunc("GET /api/zones/{id}", h.Get)
}

// createZoneRequest is the POST /api/zones payload.
type createZoneRequest struct {
	Name              string  `json:"name"`
	Area              float64 `json:"area"`
	Stock             int     `json:"stock"`
	DOWarnThreshold   float64 `json:"do_warn_threshold"`
	DODangerThreshold float64 `json:"do_danger_threshold"`
}

// Create handles POST /api/zones.
func (h *ZoneHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createZoneRequest
	if err := decodeJSON(w, r, &req); err != nil {
		Err(w, r, err)
		return
	}
	zone, err := h.svc.Zones.Create(service.CreateZoneRequest{
		Name:              req.Name,
		Area:              req.Area,
		Stock:             req.Stock,
		DOWarnThreshold:   req.DOWarnThreshold,
		DODangerThreshold: req.DODangerThreshold,
		Operator:          operatorName(r),
		RequestID:         requestIDFrom(r),
	})
	if err != nil {
		Err(w, r, err)
		return
	}
	Created(w, r, zone)
}

// List handles GET /api/zones?limit=&offset=.
func (h *ZoneHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parseLimitOffset(r, 100, 1000)
	if err != nil {
		Err(w, r, err)
		return
	}
	page, total := paginate(h.svc.Zones.List(), offset, limit)
	setListHeaders(w, limit, offset, total)
	OK(w, r, page)
}

// Get handles GET /api/zones/{id}.
func (h *ZoneHandler) Get(w http.ResponseWriter, r *http.Request) {
	zone, err := h.svc.Zones.Get(pathValue(r, "id"))
	if err != nil {
		Err(w, r, err)
		return
	}
	OK(w, r, zone)
}
