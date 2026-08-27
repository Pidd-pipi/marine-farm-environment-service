package service

import (
	"math"
	"time"

	"example.com/marine-farm-environment-service/domain"
	"example.com/marine-farm-environment-service/store"
)

// CreateZoneRequest is the validated input of a zone creation.
type CreateZoneRequest struct {
	Name              string
	Area              float64
	Stock             int
	DOWarnThreshold   float64
	DODangerThreshold float64
	Operator          string
	RequestID         string
}

// ZoneService manages farm-zone lifecycle.
type ZoneService struct {
	store *store.Store
	audit *AuditService
}

// NewZoneService builds the zone service.
func NewZoneService(st *store.Store, audit *AuditService) *ZoneService {
	return &ZoneService{store: st, audit: audit}
}

// Create validates and persists a new farm zone. When the caller does not
// provide DO thresholds, the service defaults are injected.
func (s *ZoneService) Create(req CreateZoneRequest) (domain.FarmZone, error) {
	now := time.Now().UTC()
	if req.Name == "" {
		return domain.FarmZone{}, domain.InvalidInput("name is required")
	}
	if math.IsNaN(req.Area) || math.IsInf(req.Area, 0) || req.Area <= 0 {
		return domain.FarmZone{}, domain.InvalidInput("area must be a finite positive number")
	}
	if req.Stock <= 0 {
		return domain.FarmZone{}, domain.InvalidInput("stock must be positive")
	}
	warn := req.DOWarnThreshold
	danger := req.DODangerThreshold
	if math.IsNaN(warn) || math.IsInf(warn, 0) || math.IsNaN(danger) || math.IsInf(danger, 0) {
		return domain.FarmZone{}, domain.InvalidInput("do thresholds must be finite numbers")
	}
	if warn < 0 || danger < 0 {
		return domain.FarmZone{}, domain.InvalidInput("do thresholds must not be negative")
	}
	if warn <= 0 {
		warn = 4.0
	}
	if danger <= 0 {
		danger = 3.0
	}
	if warn <= danger {
		return domain.FarmZone{}, domain.InvalidInput("do_warn_threshold must exceed do_danger_threshold")
	}
	zone := domain.NewFarmZone(s.store.NewID("zone"), req.Name, req.Area, req.Stock, warn, danger, now)
	if err := s.store.Zones().Create(zone); err != nil {
		return domain.FarmZone{}, err
	}
	_, _ = s.audit.Record(domain.AuditZoneCreate, "zone", zone.ID, req.Operator,
		"create zone "+zone.Name+" (area="+f2(req.Area)+", stock="+itoa(req.Stock)+")", req.RequestID, now)
	return *zone, nil
}

// Get returns one zone. The store error is returned verbatim so its domain
// error code (e.g. CodeNotFound) is preserved end-to-end through to the HTTP
// response; wrapping it with fmt.Errorf would strip the typed error and make
// a missing zone surface as a 500 instead of a 404.
func (s *ZoneService) Get(id string) (domain.FarmZone, error) {
	return s.store.Zones().Get(id)
}

// List returns all zones.
func (s *ZoneService) List() []domain.FarmZone {
	return s.store.Zones().List()
}
