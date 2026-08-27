package service

import (
	"fmt"
	"math"
	"time"

	"example.com/marine-farm-environment-service/domain"
	"example.com/marine-farm-environment-service/store"
)

// CreateBuoyRequest is the validated input of a buoy creation.
type CreateBuoyRequest struct {
	ZoneID    string
	Name      string
	Position  string
	Latitude  float64
	Longitude float64
	Operator  string
	RequestID string
}

// BuoyService manages monitoring-buoy lifecycle.
type BuoyService struct {
	store *store.Store
	audit *AuditService
}

// NewBuoyService builds the buoy service.
func NewBuoyService(st *store.Store, audit *AuditService) *BuoyService {
	return &BuoyService{store: st, audit: audit}
}

// Create validates and persists a new buoy attached to a zone.
func (s *BuoyService) Create(req CreateBuoyRequest) (domain.Buoy, error) {
	now := time.Now().UTC()
	if req.Name == "" {
		return domain.Buoy{}, domain.InvalidInput("name is required")
	}
	if math.IsNaN(req.Latitude) || math.IsInf(req.Latitude, 0) || math.IsNaN(req.Longitude) || math.IsInf(req.Longitude, 0) {
		return domain.Buoy{}, domain.InvalidInput("latitude/longitude must be finite numbers")
	}
	if req.Latitude < -90 || req.Latitude > 90 {
		return domain.Buoy{}, domain.InvalidInput("latitude must be within [-90, 90]")
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		return domain.Buoy{}, domain.InvalidInput("longitude must be within [-180, 180]")
	}
	if _, err := s.store.Zones().Get(req.ZoneID); err != nil {
		return domain.Buoy{}, err
	}
	buoy := domain.NewBuoy(s.store.NewID("buoy"), req.ZoneID, req.Name, req.Position, req.Latitude, req.Longitude, now)
	if err := s.store.Buoys().Create(buoy); err != nil {
		return domain.Buoy{}, err
	}
	_, _ = s.audit.Record(domain.AuditBuoyCreate, "buoy", buoy.ID, req.Operator,
		"create buoy "+buoy.Name+" in zone "+req.ZoneID, req.RequestID, now)
	return *buoy, nil
}

// Get returns one buoy.
func (s *BuoyService) Get(id string) (domain.Buoy, error) {
	b, err := s.store.Buoys().Get(id)
	if err != nil {
		return domain.Buoy{}, fmt.Errorf("buoy service get %s: %v", id, err)
	}
	return b, nil
}

// List returns all buoys.
func (s *BuoyService) List() []domain.Buoy {
	return s.store.Buoys().List()
}

// ListByZone returns the buoys of a zone.
func (s *BuoyService) ListByZone(zoneID string) []domain.Buoy {
	return s.store.Buoys().ListByZone(zoneID)
}
