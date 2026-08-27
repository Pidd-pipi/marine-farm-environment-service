package service

import (
	"context"
	"log/slog"
	"time"

	"example.com/marine-farm-environment-service/config"
	"example.com/marine-farm-environment-service/domain"
	"example.com/marine-farm-environment-service/store"
)

// RestoreCheckResult summarises one sweep of the restore checker.
type RestoreCheckResult struct {
	CheckedZones    int       `json:"checked_zones"`
	RestoreEligible int       `json:"restore_eligible"`
	AeratorTimeouts int       `json:"aerator_timeouts"`
	CheckedAt       time.Time `json:"checked_at"`
}

// RestoreChecker is the background dissolved-oxygen recovery detector. It
// runs every RestoreCheckInterval (5 minutes by default), evaluates the
// sustained-restore condition of every aerating zone and marks eligible
// zones so the operator may confirm restore. It also treats aerator
// commands without feedback as faults.
type RestoreChecker struct {
	cfg      *config.Config
	store    *store.Store
	aeration *AerationService
	audit    *AuditService
}

// NewRestoreChecker builds the restore checker.
func NewRestoreChecker(cfg *config.Config, st *store.Store, aeration *AerationService, audit *AuditService) *RestoreChecker {
	return &RestoreChecker{cfg: cfg, store: st, aeration: aeration, audit: audit}
}

// Run loops the checker until the context is cancelled.
func (r *RestoreChecker) Run(ctx context.Context) {
	ticker := time.NewTicker(r.cfg.RestoreCheckInterval)
	defer ticker.Stop()
	// Run once shortly after boot so short restore windows are detected
	// promptly in demo/test setups.
	r.RunOnce(time.Now().UTC())
	for {
		select {
		case <-ctx.Done():
			slog.Info("restore checker stopping")
			return
		case now := <-ticker.C:
			res := r.RunOnce(now.UTC())
			if res.RestoreEligible > 0 || res.AeratorTimeouts > 0 {
				slog.Info("restore checker sweep done",
					"zones", res.CheckedZones,
					"eligible", res.RestoreEligible,
					"timeouts", res.AeratorTimeouts)
			}
		}
	}
}

// RunOnce performs one full sweep and returns the summary. It is exported
// for deterministic testing.
func (r *RestoreChecker) RunOnce(now time.Time) RestoreCheckResult {
	res := RestoreCheckResult{CheckedAt: now}
	res.AeratorTimeouts = r.aeration.CheckTimeouts(now, "restore_checker")

	for _, zone := range r.store.Zones().List() {
		if zone.Status != domain.ZoneStatusAerating {
			continue
		}
		res.CheckedZones++
		eligible := r.evaluateSustainedDO(zone, now)
		if eligible {
			if !zone.RestoreEligible {
				zone.MarkRestoreEligible(now)
				if err := r.store.Zones().Update(&zone); err == nil {
					_, _ = r.audit.Record(domain.AuditRestoreCheck, "zone", zone.ID, "system",
						"zone "+zone.ID+" now restore-eligible (DO >= "+f2(r.cfg.DORestoreThreshold)+" sustained "+r.cfg.RestoreSustained.String()+")",
						"restore_checker", now)
					res.RestoreEligible++
				}
			} else {
				res.RestoreEligible++
			}
		}
	}
	return res
}

// evaluateSustainedDO checks whether the most recent dissolved-oxygen run
// of the zone has stayed above the restore threshold for RestoreSustained.
func (r *RestoreChecker) evaluateSustainedDO(zone domain.FarmZone, now time.Time) bool {
	window := r.cfg.RestoreSustained + 2*r.cfg.SamplePeriod
	from := now.Add(-window)
	samples := r.store.Samples().ListByZoneSince(zone.ID, from, now)
	if len(samples) == 0 {
		return false
	}
	newest := samples[len(samples)-1]
	if newest.DO < r.cfg.DORestoreThreshold {
		return false
	}
	// Stale data (no recent report) must not qualify as sustained recovery.
	if now.Sub(newest.Timestamp) > 2*r.cfg.SamplePeriod {
		return false
	}
	// Walk backwards through the consecutive run of DO >= threshold.
	runStart := newest.Timestamp
	for i := len(samples) - 2; i >= 0; i-- {
		s := samples[i]
		if s.DO >= r.cfg.DORestoreThreshold {
			runStart = s.Timestamp
		} else {
			break
		}
	}
	return now.Sub(runStart) >= r.cfg.RestoreSustained
}
