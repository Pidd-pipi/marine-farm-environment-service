package httpapi

import (
	"net/http"

	"example.com/marine-farm-environment-service/service"
)

// BuoyHandler serves monitoring-buoy CRUD.
type BuoyHandler struct {
	svc *service.Services
}

// NewBuoyHandler builds the buoy handler.
func NewBuoyHandler(svc *service.Services) *BuoyHandler {
	return &BuoyHandler{svc: svc}
}

// createBuoyRequest is the POST /api/buoys payload.
type createBuoyRequest struct {
	ZoneID    string  `json:"zone_id"`
	Name      string  `json:"name"`
	Position  string  `json:"position"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Create handles POST /api/buoys.
func (h *BuoyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createBuoyRequest
	if err := decodeJSON(w, r, &req); err != nil {
		Err(w, r, err)
		return
	}
	buoy, err := h.svc.Buoys.Create(service.CreateBuoyRequest{
		ZoneID:    req.ZoneID,
		Name:      req.Name,
		Position:  req.Position,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		Operator:  operatorName(r),
		RequestID: requestIDFrom(r),
	})
	if err != nil {
		Err(w, r, err)
		return
	}
	Created(w, r, buoy)
}

// List handles GET /api/buoys?limit=&offset=.
func (h *BuoyHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parseLimitOffset(r, 100, 1000)
	if err != nil {
		Err(w, r, err)
		return
	}
	page, total := paginate(h.svc.Buoys.List(), offset, limit)
	setListHeaders(w, limit, offset, total)
	OK(w, r, page)
}

// Get handles GET /api/buoys/{id}.
func (h *BuoyHandler) Get(w http.ResponseWriter, r *http.Request) {
	buoy, err := h.svc.Buoys.Get(pathValue(r, "id"))
	if err != nil {
		Err(w, r, err)
		return
	}
	OK(w, r, buoy)
}
