package httpapi

import (
	"net/http"
	"time"

	"example.com/marine-farm-environment-service/service"
)

// SampleHandler serves water-quality sample reporting and trend queries.
type SampleHandler struct {
	svc *service.Services
}

// NewSampleHandler builds the sample handler.
func NewSampleHandler(svc *service.Services) *SampleHandler {
	return &SampleHandler{svc: svc}
}

// postSampleRequest is the POST /api/buoys/{id}/samples payload. The
// timestamp is optional and defaults to the server time.
type postSampleRequest struct {
	DO          float64 `json:"do"`
	Temperature float64 `json:"temperature"`
	Salinity    float64 `json:"salinity"`
	PH          float64 `json:"ph"`
	Ammonia     float64 `json:"ammonia"`
	Timestamp   string  `json:"timestamp,omitempty"`
}

// PostSample handles POST /api/buoys/{id}/samples: water-quality report
// with over-limit judgement and cross validation.
func (h *SampleHandler) PostSample(w http.ResponseWriter, r *http.Request) {
	buoyID := pathValue(r, "id")
	var req postSampleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		Err(w, r, err)
		return
	}
	ts, err := parseOptionalTime(req.Timestamp)
	if err != nil {
		Err(w, r, err)
		return
	}
	result, err := h.svc.Ingest.Process(service.IngestRequest{
		BuoyID:      buoyID,
		DO:          req.DO,
		Temperature: req.Temperature,
		Salinity:    req.Salinity,
		PH:          req.PH,
		Ammonia:     req.Ammonia,
		Timestamp:   ts,
		Operator:    operatorName(r),
		RequestID:   requestIDFrom(r),
	})
	if err != nil {
		Err(w, r, err)
		return
	}
	Created(w, r, result)
}

// ZoneSamples handles GET /api/zones/{id}/samples?limit=&offset=&buoy_id=X:
// the water-quality trend of a zone or single buoy.
func (h *SampleHandler) ZoneSamples(w http.ResponseWriter, r *http.Request) {
	zoneID := pathValue(r, "id")
	limit, offset, err := parseLimitOffset(r, 200, 2000)
	if err != nil {
		Err(w, r, err)
		return
	}
	buoyID := r.URL.Query().Get("buoy_id")
	full := h.svc.Store.Samples().ListByZone(zoneID, 0)
	if buoyID != "" {
		full = h.svc.Store.Samples().ListByBuoy(buoyID, 0)
	}
	page, total := paginate(full, offset, limit)
	setListHeaders(w, limit, offset, total)
	OK(w, r, page)
}

// parseOptionalTime parses an RFC3339 timestamp; empty means zero time.
func parseOptionalTime(v string) (time.Time, error) {
	if v == "" {
		return time.Time{}, nil
	}
	ts, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}, domainInvalidTime(v)
	}
	return ts, nil
}
