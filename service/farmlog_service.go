package service

import (
	"fmt"
	"time"

	"example.com/marine-farm-environment-service/config"
	"example.com/marine-farm-environment-service/domain"
	"example.com/marine-farm-environment-service/store"
)

// FarmLogRequest is the validated input of a farm-log entry.
type FarmLogRequest struct {
	ZoneID      string
	Date        string
	FeedAmount  float64
	DeathCount  int
	DiseaseNote string
	Operator    string
	RequestID   string
}

// FarmLogService manages daily farming records.
type FarmLogService struct {
	cfg   *config.Config
	store *store.Store
	audit *AuditService
}

// NewFarmLogService builds the farm-log service.
func NewFarmLogService(cfg *config.Config, st *store.Store, audit *AuditService) *FarmLogService {
	return &FarmLogService{cfg: cfg, store: st, audit: audit}
}

// Create validates and persists a daily farm log. Exactly one record per
// zone and date is allowed; the death-abnormal rule (> 1% of stock) is
// evaluated automatically.
func (s *FarmLogService) Create(req FarmLogRequest) (domain.FarmLog, error) {
	now := time.Now().UTC()
	zone, err := s.store.Zones().Get(req.ZoneID)
	if err != nil {
		return domain.FarmLog{}, err
	}
	if err := domain.ValidateLogInput(req.Date, req.FeedAmount, req.DeathCount, zone.Stock); err != nil {
		return domain.FarmLog{}, err
	}
	if _, exists := s.store.FarmLogs().ByZoneAndDate(req.ZoneID, req.Date); exists {
		return domain.FarmLog{}, domain.Conflict(
			"farm log already exists for zone %s on %s", req.ZoneID, req.Date)
	}
	log := domain.NewFarmLog(
		s.store.NewID("farmlog"), zone.ID, req.Date,
		req.FeedAmount, req.DeathCount, req.DiseaseNote, req.Operator,
		zone.Stock, s.cfg.DeathAbnormalRatio, now,
	)
	if err := s.store.FarmLogs().Create(log); err != nil {
		return domain.FarmLog{}, err
	}
	_, _ = s.audit.Record(domain.AuditFarmLogCreate, "farmlog", log.ID, req.Operator,
		fmt.Sprintf("farm log %s feed=%v death=%d abnormal=%v", req.Date, f2(req.FeedAmount), req.DeathCount, log.DeathAbnormal),
		req.RequestID, now)
	return *log, nil
}

// List returns farm logs, optionally filtered by zone.
func (s *FarmLogService) List(zoneID string, limit int) []domain.FarmLog {
	if zoneID != "" {
		return s.store.FarmLogs().ListByZone(zoneID, limit)
	}
	return s.store.FarmLogs().List(limit)
}

// Get returns one farm log.
func (s *FarmLogService) Get(id string) (domain.FarmLog, error) {
	return s.store.FarmLogs().Get(id)
}
