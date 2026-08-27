package service

import (
	"time"

	"example.com/marine-farm-environment-service/config"
	"example.com/marine-farm-environment-service/domain"
	"example.com/marine-farm-environment-service/store"
)

// IngestRequest is the validated input of a buoy water-quality report.
type IngestRequest struct {
	BuoyID      string
	DO          float64
	Temperature float64
	Salinity    float64
	PH          float64
	Ammonia     float64
	Timestamp   time.Time
	Operator    string
	RequestID   string
}

// IngestResult summarises everything that happened while processing one
// sample: the persisted sample, violations, warnings, cross-validation,
// zone transition and aeration linkage.
type IngestResult struct {
	Sample           domain.WaterSample       `json:"sample"`
	Zone             domain.FarmZone          `json:"zone"`
	Buoy             domain.Buoy              `json:"buoy"`
	OverLimit        bool                     `json:"over_limit"`
	Violations       []domain.Violation       `json:"violations"`
	WarningsCreated  []domain.WarningRecord   `json:"warnings_created"`
	CrossCheck       *domain.CrossCheckResult `json:"cross_check,omitempty"`
	ZoneTransitioned bool                     `json:"zone_transitioned"`
	ZoneFrom         domain.ZoneStatus        `json:"zone_from"`
	ZoneTo           domain.ZoneStatus        `json:"zone_to"`
	AerationIssued   bool                     `json:"aeration_issued"`
	AerationLog      *domain.AerationLog      `json:"aeration_log,omitempty"`
	ResolvedWarnings int                      `json:"resolved_warnings"`
}

// IngestService processes buoy reports end-to-end: sample persistence,
// over-limit judgement, warning creation, neighbouring-buoy cross
// validation, zone state-machine transitions and automatic aeration
// linkage for confirmed dangers.
type IngestService struct {
	cfg      *config.Config
	store    *store.Store
	aeration *AerationService
	audit    *AuditService
}

// NewIngestService builds the ingest service.
func NewIngestService(cfg *config.Config, st *store.Store, aeration *AerationService, audit *AuditService) *IngestService {
	return &IngestService{cfg: cfg, store: st, aeration: aeration, audit: audit}
}

// Process handles one reported sample and returns the full result.
func (s *IngestService) Process(req IngestRequest) (IngestResult, error) {
	res := IngestResult{}
	now := time.Now().UTC()

	if !finiteNonNegative(req.DO) {
		return res, domain.InvalidInput("do must be a finite non-negative number")
	}
	if !finiteNonNegative(req.Temperature) {
		return res, domain.InvalidInput("temperature must be a finite non-negative number")
	}
	if !finiteNonNegative(req.Salinity) {
		return res, domain.InvalidInput("salinity must be a finite non-negative number")
	}
	if !finiteNonNegative(req.PH) {
		return res, domain.InvalidInput("ph must be a finite non-negative number")
	}
	if !finiteNonNegative(req.Ammonia) {
		return res, domain.InvalidInput("ammonia must be a finite non-negative number")
	}

	buoy, err := s.store.Buoys().Get(req.BuoyID)
	if err != nil {
		return res, err
	}
	zone, err := s.store.Zones().Get(buoy.ZoneID)
	if err != nil {
		return res, err
	}
	if !buoy.IsReporting() {
		return res, domain.Conflict("buoy %s is %s and cannot report samples", buoy.ID, buoy.Status)
	}

	ts := req.Timestamp
	if ts.IsZero() {
		ts = now
	}
	if err := s.validateReportTime(buoy, ts, now); err != nil {
		return res, err
	}

	sample := domain.NewWaterSample(
		s.store.NewID("sample"), buoy.ID, zone.ID,
		req.DO, req.Temperature, req.Salinity, req.PH, req.Ammonia, ts,
	)

	thresholds := s.sampleThresholds(&zone)
	violations, over := sample.EvaluateLimits(thresholds)
	res.Sample = *sample
	res.Violations = violations
	res.OverLimit = over

	// ---- Persist the sample first (evidence for cross validation) ------
	if err := s.store.Samples().Create(sample, s.cfg.MaxSamplesPerBuoy); err != nil {
		return res, err
	}

	// ---- Warning creation + cross validation ---------------------------
	zoneFrom := zone.Status
	contradicted := false
	var crossCheck *domain.CrossCheckResult

	for _, v := range violations {
		rec := domain.NewWarningRecord(
			s.store.NewID("warning"), zone.ID, buoy.ID,
			v.Type, v.Level, sample,
			domain.ViolationSummary([]domain.Violation{v}), now,
		)
		// Only a dangerous dissolved-oxygen reading is cross-validated.
		if v.Type == domain.WarningTypeDOLow && v.Level == domain.WarningLevelDanger {
			cc := s.crossValidate(zone, buoy, sample, thresholds, now)
			crossCheck = &cc
			rec.CrossChecked = true
			rec.CrossCheckOK = cc.Contradicted
			if cc.Contradicted {
				rec.Pending()
				contradicted = true
				rec.Detail = cc.Reason
			}
		}
		if err := s.store.Warnings().Create(rec); err != nil {
			return res, err
		}
		res.WarningsCreated = append(res.WarningsCreated, *rec)
		_, _ = s.audit.Record(domain.AuditWarningCreated, "warning", rec.ID, req.Operator,
			"warning "+string(rec.Type)+"/"+string(rec.Level)+" -> "+string(rec.Status)+" (do="+f2(sample.DO)+")", req.RequestID, now)
	}
	res.CrossCheck = crossCheck

	// ---- Zone state machine --------------------------------------------
	target := s.zoneTargetStatus(&zone, sample.DO, contradicted, thresholds)
	zoneTransitioned := false
	// The aerating and restored states are owned by the restore flow: only
	// new incidents may pull them out (restored -> warning/danger), the
	// aerating state itself is never auto-downgraded by a single sample.
	if zone.Status != domain.ZoneStatusAerating {
		if domain.CanZoneTransition(zone.Status, target) {
			if err := zone.SetStatus(target, now); err == nil {
				zoneTransitioned = true
			}
		}
	}
	res.ZoneFrom = zoneFrom
	res.ZoneTo = zone.Status
	res.ZoneTransitioned = zoneTransitioned

	// Persist the zone state now so aeration.Start observes the fresh
	// danger status when it reads the repository.
	if err := s.store.Zones().Update(&zone); err != nil {
		return res, err
	}

	// ---- Auto aeration for confirmed danger ------------------------------
	// A confirmed danger (not contradicted by a neighbour) automatically
	// issues the aerator start command; aeration.Start owns the
	// danger -> aerating transition so both stay consistent in the store.
	if target == domain.ZoneStatusDanger && !contradicted &&
		zone.Status != domain.ZoneStatusAerating && zone.Status != domain.ZoneStatusRestored {
		aerLog, aerr := s.aeration.Start(zone.ID, domain.TriggerAuto, req.Operator, req.RequestID)
		if aerr != nil {
			return res, aerr
		}
		res.AerationIssued = true
		res.AerationLog = &aerLog
		// Reload the zone: aeration.Start moved it to aerating in the store.
		zone, err = s.store.Zones().Get(zone.ID)
		if err != nil {
			return res, err
		}
		res.ZoneTo = zone.Status
	}

	// ---- Auto-resolve warnings whose condition is now normal -------------
	resolved := s.resolveOpenWarnings(&zone, sample, thresholds, now, req.RequestID)
	res.ResolvedWarnings = resolved

	if err := s.store.Zones().Update(&zone); err != nil {
		return res, err
	}
	buoy.RecordReport(sample.DO, ts)
	if err := s.store.Buoys().Update(&buoy); err != nil {
		return res, err
	}

	res.Zone = zone
	res.Buoy = buoy
	_, _ = s.audit.Record(domain.AuditSampleIngest, "sample", sample.ID, req.Operator,
		"sample from "+buoy.ID+" do="+f2(sample.DO)+" temp="+f2(sample.Temperature)+" over_limit="+boolStr(over)+" zone="+string(zone.Status),
		req.RequestID, now)
	return res, nil
}

// validateReportTime enforces the buoy reporting rules: no out-of-order
// reports, at least one sample period between consecutive reports, and no
// timestamps far in the future.
func (s *IngestService) validateReportTime(buoy domain.Buoy, ts, now time.Time) error {
	if ts.After(now.Add(5 * time.Minute)) {
		return domain.InvalidInput("timestamp %s is too far in the future", ts.Format(time.RFC3339))
	}
	if buoy.LastReportAt != nil {
		last := *buoy.LastReportAt
		if ts.Before(last) {
			return domain.InvalidInput("timestamp %s is before the buoy's last report %s", ts.Format(time.RFC3339), last.Format(time.RFC3339))
		}
		minGap := s.cfg.SamplePeriod - s.cfg.SamplePeriodTolerance
		if minGap > 0 && ts.Sub(last) < minGap {
			return domain.Conflict(
				"buoy %s reporting too frequently: %v since last report, minimum interval %v",
				buoy.ID, ts.Sub(last).Round(time.Second), minGap.Round(time.Second))
		}
	}
	return nil
}

// sampleThresholds merges the zone DO thresholds with the global water
// range into the evaluation envelope.
func (s *IngestService) sampleThresholds(zone *domain.FarmZone) domain.SampleThresholds {
	r := s.cfg.WaterRange()
	return domain.SampleThresholds{
		DOWarnThreshold:   zone.DOWarnThreshold,
		DODangerThreshold: zone.DODangerThreshold,
		TempMin:           r.TempMin,
		TempMax:           r.TempMax,
		SalinityMin:       r.SalinityMin,
		SalinityMax:       r.SalinityMax,
		PHMin:             r.PHMin,
		PHMax:             r.PHMax,
		AmmoniaMax:        r.AmmoniaMax,
	}
}

// crossValidate collects neighbouring-buoy samples inside the window and
// runs the cross validation.
func (s *IngestService) crossValidate(zone domain.FarmZone, buoy domain.Buoy, sample *domain.WaterSample, th domain.SampleThresholds, now time.Time) domain.CrossCheckResult {
	window := s.cfg.CrossCheckWindow
	from := sample.Timestamp.Add(-window)
	neighbours := s.store.Samples().ListByZoneSince(zone.ID, from, sample.Timestamp)
	ns := domain.NeighbourSamples{
		ZoneID:  zone.ID,
		BuoyID:  buoy.ID,
		From:    from,
		To:      sample.Timestamp,
		Samples: neighbours,
	}
	return domain.EvaluateCrossValidation(ns, zone.DOWarnThreshold)
}

// zoneTargetStatus computes the status a sample drives the zone towards.
// A contradicted danger keeps the zone at warning until it is verified.
func (s *IngestService) zoneTargetStatus(zone *domain.FarmZone, do float64, contradicted bool, th domain.SampleThresholds) domain.ZoneStatus {
	if contradicted {
		return domain.ZoneStatusWarning
	}
	return zone.TargetStatusFromDO(do)
}

// resolveOpenWarnings clears confirmed warnings whose indicator is now
// within range. Pending warnings always require operator action.
func (s *IngestService) resolveOpenWarnings(zone *domain.FarmZone, sample *domain.WaterSample, th domain.SampleThresholds, now time.Time, requestID string) int {
	resolved := 0
	open := s.store.Warnings().List(store.WarningFilter{ZoneID: zone.ID, Status: string(domain.WarningStatusConfirmed)})
	for i := range open {
		rec := &open[i]
		if !rec.IsOpen() {
			continue
		}
		if !warningConditionCleared(rec.Type, sample, th) {
			continue
		}
		if err := rec.Resolve(now); err != nil {
			continue
		}
		if err := s.store.Warnings().Update(rec); err != nil {
			continue
		}
		resolved++
		_, _ = s.audit.Record(domain.AuditWarningResolve, "warning", rec.ID, "system",
			"auto-resolve "+string(rec.Type)+" after data normalised", requestID, now)
	}
	return resolved
}

// warningConditionCleared reports whether the warning's indicator is now
// back within range.
func warningConditionCleared(wtype domain.WarningType, sample *domain.WaterSample, th domain.SampleThresholds) bool {
	switch wtype {
	case domain.WarningTypeDOLow:
		return sample.DO >= th.DOWarnThreshold
	case domain.WarningTypeTempShock:
		return sample.Temperature >= th.TempMin && sample.Temperature <= th.TempMax
	case domain.WarningTypePHAbnormal:
		return sample.PH >= th.PHMin && sample.PH <= th.PHMax
	case domain.WarningTypeAmmoniaHigh:
		return sample.Ammonia <= th.AmmoniaMax
	}
	return false
}

// boolStr formats a boolean for audit detail lines.
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
