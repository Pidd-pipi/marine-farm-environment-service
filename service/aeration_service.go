package service

import (
	"fmt"
	"time"

	"example.com/marine-farm-environment-service/config"
	"example.com/marine-farm-environment-service/domain"
	"example.com/marine-farm-environment-service/store"
)

// AerationService manages aerator commands: start/stop issuance, device
// feedback and the aerator state machine.
type AerationService struct {
	cfg   *config.Config
	store *store.Store
	audit *AuditService
}

// NewAerationService builds the aeration service.
func NewAerationService(cfg *config.Config, st *store.Store, audit *AuditService) *AerationService {
	return &AerationService{cfg: cfg, store: st, audit: audit}
}

// Start issues an aerator start command for a zone. When the zone is in
// the danger state the zone moves to aerating (the canonical transition).
// A manual start on a normal zone only runs the aerator without changing
// the zone lifecycle.
func (s *AerationService) Start(zoneID string, trigger domain.AerationTrigger, operator, requestID string) (domain.AerationLog, error) {
	now := time.Now().UTC()
	zone, err := s.store.Zones().Get(zoneID)
	if err != nil {
		return domain.AerationLog{}, err
	}
	if latest, ok := s.store.Aeration().LatestByZone(zoneID); ok && latest.IsActive() {
		return domain.AerationLog{}, domain.Conflict(
			"zone %s already aerating: command %s is %s", zoneID, latest.ID, latest.Status)
	}
	note := fmt.Sprintf("trigger=%s zone=%s", trigger, zone.Name)
	log, err := domain.NewAerationLog(
		s.store.NewID("aeration"), zone.ID, "aerator_"+zone.ID,
		domain.AerationActionStart, trigger, note, now)
	if err != nil {
		return domain.AerationLog{}, err
	}
	if err := s.store.Aeration().Create(log); err != nil {
		return domain.AerationLog{}, err
	}
	if zone.Status == domain.ZoneStatusDanger {
		if err := zone.SetStatus(domain.ZoneStatusAerating, now); err != nil {
			return domain.AerationLog{}, err
		}
		if err := s.store.Zones().Update(&zone); err != nil {
			return domain.AerationLog{}, err
		}
	}
	_, _ = s.audit.Record(domain.AuditAerationStart, "zone", zone.ID, operator,
		"aeration start issued: "+log.ID+" ("+string(trigger)+")", requestID, now)
	return *log, nil
}

// Stop issues an aerator stop command for a zone.
func (s *AerationService) Stop(zoneID string, trigger domain.AerationTrigger, operator, requestID string) (domain.AerationLog, error) {
	now := time.Now().UTC()
	zone, err := s.store.Zones().Get(zoneID)
	if err != nil {
		return domain.AerationLog{}, err
	}
	log, err := domain.NewAerationLog(
		s.store.NewID("aeration"), zone.ID, "aerator_"+zone.ID,
		domain.AerationActionStop, trigger, "stop command for zone "+zone.Name, now)
	if err != nil {
		return domain.AerationLog{}, err
	}
	if err := s.store.Aeration().Create(log); err != nil {
		return domain.AerationLog{}, err
	}
	_, _ = s.audit.Record(domain.AuditAerationStop, "zone", zone.ID, operator,
		"aeration stop issued: "+log.ID+" ("+string(trigger)+")", requestID, now)
	return *log, nil
}

// Feedback applies device feedback to an aeration command and advances the
// aerator state machine.
func (s *AerationService) Feedback(logID string, fb domain.FeedbackStatus, operator, requestID string) (domain.AerationLog, error) {
	now := time.Now().UTC()
	log, err := s.store.Aeration().Get(logID)
	if err != nil {
		return domain.AerationLog{}, err
	}
	status, err := log.ApplyFeedback(fb, now)
	if err != nil {
		return domain.AerationLog{}, err
	}
	if err := s.store.Aeration().Update(&log); err != nil {
		return domain.AerationLog{}, err
	}
	_, _ = s.audit.Record(domain.AuditAerationFeedback, "aeration", log.ID, operator,
		fmt.Sprintf("feedback %s -> %s", fb, status), requestID, now)
	return log, nil
}

// CheckTimeouts marks every unacknowledged command older than the
// configured timeout as a fault (告警 via audit + fault status). Returns
// the number of commands that timed out.
func (s *AerationService) CheckTimeouts(now time.Time, requestID string) int {
	count := 0
	for _, log := range s.store.Aeration().ListPending() {
		if !log.TimedOut(s.cfg.AeratorFeedbackTimeout, now) {
			continue
		}
		if err := log.MarkTimeout(now); err != nil {
			continue
		}
		if err := s.store.Aeration().Update(&log); err != nil {
			continue
		}
		count++
		_, _ = s.audit.Record(domain.AuditAerationTimeout, "aeration", log.ID, "system",
			fmt.Sprintf("aeration command %s timed out without feedback -> fault", log.ID), requestID, now)
	}
	return count
}

// Restore confirms the recovery of a zone: the aerator receives a stop
// command, the zone moves to restored and every open warning is resolved.
// The restore is only allowed when the restore checker marked the zone
// restore-eligible (DO >= DORestoreThreshold sustained for
// RestoreSustained).
func (s *AerationService) Restore(zoneID, operator, requestID string) (domain.AerationLog, error) {
	now := time.Now().UTC()
	zone, err := s.store.Zones().Get(zoneID)
	if err != nil {
		return domain.AerationLog{}, err
	}
	if zone.Status != domain.ZoneStatusAerating {
		return domain.AerationLog{}, domain.Conflict(
			"zone %s is %s, only aerating zones can confirm restore", zoneID, zone.Status)
	}
	if !zone.RestoreEligible {
		return domain.AerationLog{}, domain.Conflict(
			"zone %s restore not eligible yet: dissolved oxygen must stay above %v mg/L for %v",
			zoneID, s.cfg.DORestoreThreshold, s.cfg.RestoreSustained)
	}

	// Stop the aerator (restore-triggered stop command).
	stopLog, err := domain.NewAerationLog(
		s.store.NewID("aeration"), zone.ID, "aerator_"+zone.ID,
		domain.AerationActionStop, domain.TriggerRestore,
		"stop on restore confirmation", now)
	if err != nil {
		return domain.AerationLog{}, err
	}
	if err := s.store.Aeration().Create(stopLog); err != nil {
		return domain.AerationLog{}, err
	}

	if err := zone.SetStatus(domain.ZoneStatusRestored, now); err != nil {
		return domain.AerationLog{}, err
	}
	zone.ClearRestoreEligibility()
	if err := s.store.Zones().Update(&zone); err != nil {
		return domain.AerationLog{}, err
	}

	// Resolve every open warning of the zone.
	resolved := 0
	for _, rec := range s.store.Warnings().List(store.WarningFilter{ZoneID: zoneID}) {
		if !rec.IsOpen() {
			continue
		}
		if err := rec.Resolve(now); err != nil {
			continue
		}
		if err := s.store.Warnings().Update(&rec); err != nil {
			continue
		}
		resolved++
	}

	_, _ = s.audit.Record(domain.AuditZoneRestore, "zone", zone.ID, operator,
		fmt.Sprintf("restore confirmed for zone %s (resolved %d warnings)", zone.Name, resolved), requestID, now)
	_, _ = s.audit.Record(domain.AuditAerationStop, "aeration", stopLog.ID, operator,
		"aeration stop on restore: "+stopLog.ID, requestID, now)
	return *stopLog, nil
}

// LatestByZone returns the latest aeration command of a zone.
func (s *AerationService) LatestByZone(zoneID string) (domain.AerationLog, bool) {
	return s.store.Aeration().LatestByZone(zoneID)
}

// List returns recent aeration logs.
func (s *AerationService) List(limit int) []domain.AerationLog {
	return s.store.Aeration().List(limit)
}
