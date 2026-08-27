package service

import (
	"time"

	"example.com/marine-farm-environment-service/config"
	"example.com/marine-farm-environment-service/domain"
	"example.com/marine-farm-environment-service/store"
)

// ZoneOverview is the per-zone aggregation rendered on the overview and
// detail pages.
type ZoneOverview struct {
	Zone             domain.FarmZone       `json:"zone"`
	Buoys            []domain.Buoy         `json:"buoys"`
	LatestSample     *domain.WaterSample   `json:"latest_sample,omitempty"`
	LatestDO         float64               `json:"latest_do"`
	LatestSampleAt   *time.Time            `json:"latest_sample_at,omitempty"`
	BuoyCount        int                   `json:"buoy_count"`
	StaleBuoyCount   int                   `json:"stale_buoy_count"`
	OpenWarningCount int                   `json:"open_warning_count"`
	AeratorStatus    domain.AeratorStatus  `json:"aerator_status"`
	AeratorAction    domain.AerationAction `json:"aerator_action"`
	LastAerationAt   *time.Time            `json:"last_aeration_at,omitempty"`
}

// Totals aggregates the whole farm.
type Totals struct {
	ZoneCount           int `json:"zone_count"`
	BuoyCount           int `json:"buoy_count"`
	SampleCount         int `json:"sample_count"`
	OpenWarningCount    int `json:"open_warning_count"`
	PendingWarningCount int `json:"pending_warning_count"`
	ActiveAerators      int `json:"active_aerators"`
	FarmLogCount        int `json:"farm_log_count"`
	AbnormalLogCount    int `json:"abnormal_log_count"`
}

// Overview is the full dashboard aggregation.
type Overview struct {
	Zones          []ZoneOverview         `json:"zones"`
	Totals         Totals                 `json:"totals"`
	RecentWarnings []domain.WarningRecord `json:"recent_warnings"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// OverviewService builds dashboard aggregations.
type OverviewService struct {
	cfg   *config.Config
	store *store.Store
}

// NewOverviewService builds the overview service.
func NewOverviewService(cfg *config.Config, st *store.Store) *OverviewService {
	return &OverviewService{cfg: cfg, store: st}
}

// Get aggregates every zone with its latest water reading, open warnings
// and aerator state.
func (s *OverviewService) Get() Overview {
	now := time.Now().UTC()
	snapshot := s.store.Snapshot()

	zoneIdx := make(map[string]*domain.FarmZone, len(snapshot.Zones))
	for i := range snapshot.Zones {
		zoneIdx[snapshot.Zones[i].ID] = &snapshot.Zones[i]
	}
	buoysByZone := make(map[string][]domain.Buoy)
	for i := range snapshot.Buoys {
		b := snapshot.Buoys[i]
		buoysByZone[b.ZoneID] = append(buoysByZone[b.ZoneID], b)
	}
	latestByZone := make(map[string]*domain.WaterSample)
	for i := range snapshot.Samples {
		sample := &snapshot.Samples[i]
		if cur, ok := latestByZone[sample.ZoneID]; !ok || sample.Timestamp.After(cur.Timestamp) {
			latestByZone[sample.ZoneID] = sample
		}
	}
	aeratorByZone := make(map[string]*domain.AerationLog)
	for i := range snapshot.Aeration {
		log := &snapshot.Aeration[i]
		if cur, ok := aeratorByZone[log.ZoneID]; !ok || log.CommandTime.After(cur.CommandTime) {
			aeratorByZone[log.ZoneID] = log
		}
	}

	ov := Overview{UpdatedAt: now}
	ov.Totals.ZoneCount = len(snapshot.Zones)
	ov.Totals.BuoyCount = len(snapshot.Buoys)
	ov.Totals.SampleCount = len(snapshot.Samples)

	for _, z := range snapshot.Zones {
		zo := ZoneOverview{Zone: z, LatestDO: -1}
		zo.Buoys = buoysByZone[z.ID]
		zo.BuoyCount = len(zo.Buoys)
		if latest, ok := latestByZone[z.ID]; ok {
			latestCopy := *latest
			zo.LatestSample = &latestCopy
			zo.LatestDO = latest.DO
			ts := latest.Timestamp
			zo.LatestSampleAt = &ts
		}
		for i := range zo.Buoys {
			b := &zo.Buoys[i]
			if b.Stale(2*s.cfg.SamplePeriod, now) {
				zo.StaleBuoyCount++
			}
		}
		zo.OpenWarningCount = openWarningsIn(&snapshot, z.ID)
		if aer, ok := aeratorByZone[z.ID]; ok {
			zo.AeratorStatus = aer.Status
			zo.AeratorAction = aer.Action
			ts := aer.CommandTime
			zo.LastAerationAt = &ts
			if aer.IsActive() {
				ov.Totals.ActiveAerators++
			}
		}
		ov.Zones = append(ov.Zones, zo)
	}

	ov.Totals.OpenWarningCount = openWarningsTotal(&snapshot)
	ov.Totals.PendingWarningCount = pendingWarningsTotal(&snapshot)
	ov.Totals.FarmLogCount = len(snapshot.FarmLogs)
	for i := range snapshot.FarmLogs {
		if snapshot.FarmLogs[i].DeathAbnormal {
			ov.Totals.AbnormalLogCount++
		}
	}
	ov.RecentWarnings = recentWarnings(&snapshot, 8)
	return ov
}

func openWarningsIn(s *store.State, zoneID string) int {
	n := 0
	for i := range s.Warnings {
		w := &s.Warnings[i]
		if w.ZoneID == zoneID && w.IsOpen() {
			n++
		}
	}
	return n
}

func openWarningsTotal(s *store.State) int {
	n := 0
	for i := range s.Warnings {
		if s.Warnings[i].IsOpen() {
			n++
		}
	}
	return n
}

func pendingWarningsTotal(s *store.State) int {
	n := 0
	for i := range s.Warnings {
		if s.Warnings[i].Status == domain.WarningStatusPending {
			n++
		}
	}
	return n
}

func recentWarnings(s *store.State, limit int) []domain.WarningRecord {
	out := make([]domain.WarningRecord, 0, len(s.Warnings))
	out = append(out, s.Warnings...)
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].ReportedAt.After(out[i].ReportedAt) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}
