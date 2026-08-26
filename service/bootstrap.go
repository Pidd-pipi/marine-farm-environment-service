package service

import (
	"log/slog"
	"math"
	"time"

	"example.com/marine-farm-environment-service/config"
	"example.com/marine-farm-environment-service/domain"
	"example.com/marine-farm-environment-service/store"
)

// Bootstrap seeds a realistic demo dataset on first run so every page
// renders meaningful content immediately. Seeding is idempotent: it only
// runs when the store holds no business entity yet.
type Bootstrap struct {
	cfg   *config.Config
	store *store.Store
}

// NewBootstrap builds the seeder.
func NewBootstrap(cfg *config.Config, st *store.Store) *Bootstrap {
	return &Bootstrap{cfg: cfg, store: st}
}

// SeedIfEmpty populates the demo dataset when the store is empty.
func (b *Bootstrap) SeedIfEmpty() error {
	if !b.store.IsEmpty() {
		return nil
	}
	now := time.Now().UTC()

	z1 := domain.NewFarmZone(b.store.NewID("zone"), "东区·1号养殖区", 120, 50000, 4.0, 3.0, now.Add(-48*time.Hour))
	z2 := domain.NewFarmZone(b.store.NewID("zone"), "西区·2号养殖区", 88, 32000, 4.0, 3.0, now.Add(-48*time.Hour))
	z3 := domain.NewFarmZone(b.store.NewID("zone"), "南区·3号养殖区", 156, 80000, 4.0, 3.0, now.Add(-24*time.Hour))

	// Zone 2 is in a sustained warning, zone 3 is aerating. Walk the zone
	// through the legal state-machine path: warning -> danger -> aerating.
	_ = z2.SetStatus(domain.ZoneStatusWarning, now.Add(-35*time.Minute))
	_ = z3.SetStatus(domain.ZoneStatusWarning, now.Add(-80*time.Minute))
	_ = z3.SetStatus(domain.ZoneStatusDanger, now.Add(-70*time.Minute))
	_ = z3.SetStatus(domain.ZoneStatusAerating, now.Add(-50*time.Minute))

	for _, z := range []*domain.FarmZone{z1, z2, z3} {
		if err := b.store.Zones().Create(z); err != nil {
			return err
		}
	}

	b1 := domain.NewBuoy(b.store.NewID("buoy"), z1.ID, "东区-1号浮标", "东经121.50 北纬30.20", 121.50, 30.20, now.Add(-48*time.Hour))
	b2 := domain.NewBuoy(b.store.NewID("buoy"), z1.ID, "东区-2号浮标", "东经121.51 北纬30.21", 121.51, 30.21, now.Add(-48*time.Hour))
	b3 := domain.NewBuoy(b.store.NewID("buoy"), z2.ID, "西区-1号浮标", "东经121.42 北纬30.18", 121.42, 30.18, now.Add(-48*time.Hour))
	b4 := domain.NewBuoy(b.store.NewID("buoy"), z2.ID, "西区-2号浮标", "东经121.43 北纬30.19", 121.43, 30.19, now.Add(-48*time.Hour))
	b5 := domain.NewBuoy(b.store.NewID("buoy"), z3.ID, "南区-1号浮标", "东经121.60 北纬30.25", 121.60, 30.25, now.Add(-24*time.Hour))
	b6 := domain.NewBuoy(b.store.NewID("buoy"), z3.ID, "南区-2号浮标", "东经121.61 北纬30.26", 121.61, 30.26, now.Add(-24*time.Hour))

	buoys := []*domain.Buoy{b1, b2, b3, b4, b5, b6}
	for _, bu := range buoys {
		if err := b.store.Buoys().Create(bu); err != nil {
			return err
		}
	}

	// ---- Historical samples -------------------------------------------
	// Each zone has a distinct dissolved-oxygen profile.
	profiles := []struct {
		buoy *domain.Buoy
		base float64 // DO baseline
	}{
		{b1, 6.4}, {b2, 6.1}, // zone 1: healthy
		{b3, 3.4}, {b4, 6.0}, // zone 2: one low buoy, one healthy
		{b5, 5.3}, {b6, 5.6}, // zone 3: recovering under aeration
	}
	sampleEvery := 5 * time.Minute
	history := 2 * time.Hour
	for _, p := range profiles {
		bu := p.buoy
		zoneID := bu.ZoneID
		for t := now.Add(-history); t.Before(now); t = t.Add(sampleEvery) {
			// Add a gentle sine wave so the trend chart looks organic.
			wave := math.Sin(float64(t.Unix())/900.0) * 0.25
			do := clamp(p.base+wave, 0.5, 12)
			sample := domain.NewWaterSample(
				b.store.NewID("sample"), bu.ID, zoneID,
				do, 24.5+math.Sin(float64(t.Unix())/3600.0)*1.2,
				31.0+math.Sin(float64(t.Unix())/1800.0)*0.8,
				8.1, 0.06+math.Abs(math.Sin(float64(t.Unix())/2400.0))*0.05,
				t,
			)
			th := b.zoneThresholds(zoneID)
			_, _ = sample.EvaluateLimits(th)
			if err := b.store.Samples().Create(sample, b.cfg.MaxSamplesPerBuoy); err != nil {
				return err
			}
			bu.RecordReport(sample.DO, t)
		}
		if err := b.store.Buoys().Update(bu); err != nil {
			return err
		}
	}

	// ---- Demo warnings -------------------------------------------------
	// A confirmed do_low warning on zone 2 (buoy b3, danger level).
	w1 := domain.NewWarningRecord(
		b.store.NewID("warning"), z2.ID, b3.ID,
		domain.WarningTypeDOLow, domain.WarningLevelDanger,
		&domain.WaterSample{ID: "seed", BuoyID: b3.ID, ZoneID: z2.ID, DO: 2.8, Timestamp: now.Add(-35 * time.Minute)},
		"溶解氧过低 2.8 mg/L（限值 3.0）", now.Add(-35*time.Minute),
	)
	_ = w1.Resolve(now.Add(-30 * time.Minute))
	if err := b.store.Warnings().Create(w1); err != nil {
		return err
	}
	// A pending (cross-checked) do_low warning on zone 2.
	w2 := domain.NewWarningRecord(
		b.store.NewID("warning"), z2.ID, b3.ID,
		domain.WarningTypeDOLow, domain.WarningLevelDanger,
		&domain.WaterSample{ID: "seed", BuoyID: b3.ID, ZoneID: z2.ID, DO: 2.9, Timestamp: now.Add(-12 * time.Minute)},
		"相邻浮标西区-2号于 12 分钟前上报溶解氧 6.0 mg/L（正常），本读数待人工核实", now.Add(-12*time.Minute),
	)
	w2.CrossChecked = true
	w2.CrossCheckOK = true
	w2.Pending()
	if err := b.store.Warnings().Create(w2); err != nil {
		return err
	}

	// ---- Demo aeration -------------------------------------------------
	aer, _ := domain.NewAerationLog(
		b.store.NewID("aeration"), z3.ID, "aerator_"+z3.ID,
		domain.AerationActionStart, domain.TriggerAuto,
		"自动增氧：溶解氧低于 3 mg/L", now.Add(-50*time.Minute),
	)
	_, _ = aer.ApplyFeedback(domain.FeedbackStarted, now.Add(-49*time.Minute))
	if err := b.store.Aeration().Create(aer); err != nil {
		return err
	}

	// ---- Demo farm logs ------------------------------------------------
	today := now.Format("2006-01-02")
	yesterday := now.Add(-24 * time.Hour).Format("2006-01-02")
	logs := []*domain.FarmLog{
		domain.NewFarmLog(b.store.NewID("farmlog"), z1.ID, today, 860.5, 12, "", "张师傅", z1.Stock, b.cfg.DeathAbnormalRatio, now),
		domain.NewFarmLog(b.store.NewID("farmlog"), z1.ID, yesterday, 812.0, 9, "", "张师傅", z1.Stock, b.cfg.DeathAbnormalRatio, now.Add(-24*time.Hour)),
		domain.NewFarmLog(b.store.NewID("farmlog"), z2.ID, today, 620.0, 460, "疑似细菌性败血症，已投药", "李师傅", z2.Stock, b.cfg.DeathAbnormalRatio, now),
		domain.NewFarmLog(b.store.NewID("farmlog"), z3.ID, today, 1150.0, 28, "", "王师傅", z3.Stock, b.cfg.DeathAbnormalRatio, now),
	}
	for _, l := range logs {
		if err := b.store.FarmLogs().Create(l); err != nil {
			return err
		}
	}

	slog.Info("bootstrap: seeded demo dataset", "zones", 3, "buoys", len(buoys), "samples", b.store.Samples().Count())
	return nil
}

// zoneThresholds resolves the evaluation thresholds for seeding.
func (b *Bootstrap) zoneThresholds(zoneID string) domain.SampleThresholds {
	z, err := b.store.Zones().Get(zoneID)
	if err != nil {
		return domain.SampleThresholds{DOWarnThreshold: 4, DODangerThreshold: 3}
	}
	r := b.cfg.WaterRange()
	return domain.SampleThresholds{
		DOWarnThreshold:   z.DOWarnThreshold,
		DODangerThreshold: z.DODangerThreshold,
		TempMin:           r.TempMin,
		TempMax:           r.TempMax,
		SalinityMin:       r.SalinityMin,
		SalinityMax:       r.SalinityMax,
		PHMin:             r.PHMin,
		PHMax:             r.PHMax,
		AmmoniaMax:        r.AmmoniaMax,
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
