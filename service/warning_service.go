package service

import (
	"time"

	"example.com/marine-farm-environment-service/domain"
	"example.com/marine-farm-environment-service/store"
)

// WarningService handles warning verification and resolution.
type WarningService struct {
	store    *store.Store
	aeration *AerationService
	audit    *AuditService
}

// NewWarningService builds the warning service.
func NewWarningService(st *store.Store, aeration *AerationService, audit *AuditService) *WarningService {
	return &WarningService{store: st, aeration: aeration, audit: audit}
}

// Verify confirms a pending warning. For a pending dissolved-oxygen
// danger the latest zone reading is re-evaluated: if it is still below the
// danger threshold the aerator starts automatically; if it recovered, the
// warning is resolved right away.
func (s *WarningService) Verify(id, operator, requestID string) (domain.WarningRecord, error) {
	now := time.Now().UTC()
	rec, err := s.store.Warnings().Get(id)
	if err != nil {
		return domain.WarningRecord{}, err
	}
	if rec.Status != domain.WarningStatusPending {
		return domain.WarningRecord{}, domain.Conflict(
			"warning %s is %s, only pending warnings can be verified", id, rec.Status)
	}
	if err := rec.Verify(now); err != nil {
		return domain.WarningRecord{}, err
	}

	// Re-evaluate the zone's latest dissolved oxygen before persisting.
	var aerLog *domain.AerationLog
	if rec.Type == domain.WarningTypeDOLow && rec.Level == domain.WarningLevelDanger {
		zone, zerr := s.store.Zones().Get(rec.ZoneID)
		if zerr != nil {
			return domain.WarningRecord{}, zerr
		}
		latest, ok := s.store.Samples().LatestByZone(rec.ZoneID)
		if ok {
			switch {
			case latest.DO < zone.DODangerThreshold && zone.Status != domain.ZoneStatusAerating:
				// Escalate the zone to danger (warning -> danger) first so
				// aeration.Start observes the canonical pre-aeration state.
				if zone.Status != domain.ZoneStatusDanger {
					if err := zone.SetStatus(domain.ZoneStatusDanger, now); err != nil {
						return domain.WarningRecord{}, err
					}
					if err := s.store.Zones().Update(&zone); err != nil {
						return domain.WarningRecord{}, err
					}
				}
				log, aerr := s.aeration.Start(zone.ID, domain.TriggerVerify, operator, requestID)
				if aerr != nil {
					return domain.WarningRecord{}, aerr
				}
				aerLog = &log
			case latest.DO >= zone.DOWarnThreshold:
				// The reading recovered before the operator acted.
				_ = rec.Resolve(now)
			}
		}
	}

	if err := s.store.Warnings().Update(&rec); err != nil {
		return domain.WarningRecord{}, err
	}
	detail := "warning verified: " + rec.ID
	if aerLog != nil {
		detail += ", aeration auto-started: " + aerLog.ID
	}
	_, _ = s.audit.Record(domain.AuditWarningVerify, "warning", rec.ID, operator, detail, requestID, now)
	return rec, nil
}

// Resolve lets an operator close a warning manually.
func (s *WarningService) Resolve(id, operator, requestID string) (domain.WarningRecord, error) {
	now := time.Now().UTC()
	rec, err := s.store.Warnings().Get(id)
	if err != nil {
		return domain.WarningRecord{}, err
	}
	if err := rec.Resolve(now); err != nil {
		return domain.WarningRecord{}, err
	}
	if err := s.store.Warnings().Update(&rec); err != nil {
		return domain.WarningRecord{}, err
	}
	_, _ = s.audit.Record(domain.AuditWarningResolve, "warning", rec.ID, operator,
		"warning resolved by operator", requestID, now)
	return rec, nil
}

// List returns warnings matching the filter.
func (s *WarningService) List(filter store.WarningFilter) []domain.WarningRecord {
	return s.store.Warnings().List(filter)
}
