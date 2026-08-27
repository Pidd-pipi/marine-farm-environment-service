package service

import (
	"context"
	"testing"
	"time"

	"example.com/marine-farm-environment-service/config"
	"example.com/marine-farm-environment-service/domain"
	"example.com/marine-farm-environment-service/store"
)

// newTestServices wires the full service graph over an in-memory store.
func newTestServices(t *testing.T) (*config.Config, *store.Store, *Services) {
	t.Helper()
	cfg := config.Default()
	cfg.DataFile = ""
	cfg.SamplePeriod = time.Second        // allow rapid reports in tests
	cfg.SamplePeriodTolerance = time.Hour // effectively no min interval
	cfg.RestoreCheckInterval = time.Second
	st := store.NewMemoryStore()
	svc := New(cfg, st)
	return cfg, st, svc
}

func seedZoneAndBuoy(t *testing.T, svc *Services, name string) (domain.FarmZone, domain.Buoy) {
	t.Helper()
	zone, err := svc.Zones.Create(CreateZoneRequest{Name: name, Area: 100, Stock: 50000})
	if err != nil {
		t.Fatalf("create zone: %v", err)
	}
	buoy, err := svc.Buoys.Create(CreateBuoyRequest{ZoneID: zone.ID, Name: name + "-浮标"})
	if err != nil {
		t.Fatalf("create buoy: %v", err)
	}
	return zone, buoy
}

func TestIngestNormalSample(t *testing.T) {
	_, _, svc := newTestServices(t)
	zone, buoy := seedZoneAndBuoy(t, svc, "东区")
	res, err := svc.Ingest.Process(IngestRequest{
		BuoyID: buoy.ID, DO: 6.2, Temperature: 24, Salinity: 31, PH: 8.1, Ammonia: 0.05,
		Timestamp: time.Now().UTC(), Operator: "test",
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if res.OverLimit || len(res.WarningsCreated) != 0 {
		t.Fatalf("normal sample must not create warnings: %+v", res)
	}
	if res.Zone.Status != domain.ZoneStatusNormal {
		t.Fatalf("zone should stay normal, got %s", res.Zone.Status)
	}
	latest, ok := svc.Store.Samples().LatestByBuoy(buoy.ID)
	if !ok || latest.DO != 6.2 {
		t.Fatalf("latest sample missing: %+v", latest)
	}
	_ = zone
}

func TestIngestDangerAutoAeration(t *testing.T) {
	_, _, svc := newTestServices(t)
	_, buoy := seedZoneAndBuoy(t, svc, "东区")
	res, err := svc.Ingest.Process(IngestRequest{
		BuoyID: buoy.ID, DO: 2.4, Temperature: 24, Salinity: 31, PH: 8.1, Ammonia: 0.05,
		Timestamp: time.Now().UTC(), Operator: "test",
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if !res.AerationIssued || res.AerationLog == nil {
		t.Fatalf("danger must auto-issue aeration: %+v", res)
	}
	if res.Zone.Status != domain.ZoneStatusAerating {
		t.Fatalf("zone should be aerating, got %s", res.Zone.Status)
	}
	zone, _ := svc.Store.Zones().Get(res.Zone.ID)
	if zone.Status != domain.ZoneStatusAerating {
		t.Fatalf("store zone should be aerating, got %s", zone.Status)
	}
	warnings := svc.Store.Warnings().ListByZone(zone.ID, 10)
	if len(warnings) != 1 || warnings[0].Status != domain.WarningStatusConfirmed {
		t.Fatalf("expected one confirmed warning, got %+v", warnings)
	}
}

func TestIngestDangerCrossCheckedPending(t *testing.T) {
	_, _, svc := newTestServices(t)
	zone, buoyA := seedZoneAndBuoy(t, svc, "东区")
	buoyB, err := svc.Buoys.Create(CreateBuoyRequest{ZoneID: zone.ID, Name: "邻居浮标"})
	if err != nil {
		t.Fatalf("create buoy b: %v", err)
	}
	now := time.Now().UTC()
	// Neighbour reports normal data 2 minutes before the danger.
	if _, err := svc.Ingest.Process(IngestRequest{
		BuoyID: buoyB.ID, DO: 6.0, Temperature: 24, Salinity: 31, PH: 8.1, Ammonia: 0.05,
		Timestamp: now.Add(-2 * time.Minute), Operator: "test",
	}); err != nil {
		t.Fatalf("neighbour ingest: %v", err)
	}
	res, err := svc.Ingest.Process(IngestRequest{
		BuoyID: buoyA.ID, DO: 2.6, Temperature: 24, Salinity: 31, PH: 8.1, Ammonia: 0.05,
		Timestamp: now, Operator: "test",
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if res.CrossCheck == nil || !res.CrossCheck.Contradicted {
		t.Fatalf("expected contradicted cross-check, got %+v", res.CrossCheck)
	}
	if res.AerationIssued {
		t.Fatal("pending danger must NOT auto-issue aeration")
	}
	if len(res.WarningsCreated) != 1 || res.WarningsCreated[0].Status != domain.WarningStatusPending {
		t.Fatalf("expected one pending warning, got %+v", res.WarningsCreated)
	}
	if res.Zone.Status != domain.ZoneStatusWarning {
		t.Fatalf("pending danger keeps zone at warning, got %s", res.Zone.Status)
	}
}

func TestWarningVerifyTriggersAeration(t *testing.T) {
	_, _, svc := newTestServices(t)
	zone, buoyA := seedZoneAndBuoy(t, svc, "东区")
	buoyB, _ := svc.Buoys.Create(CreateBuoyRequest{ZoneID: zone.ID, Name: "邻居"})
	now := time.Now().UTC()
	_, _ = svc.Ingest.Process(IngestRequest{BuoyID: buoyB.ID, DO: 6.0, Timestamp: now.Add(-2 * time.Minute), Operator: "t"})
	res, _ := svc.Ingest.Process(IngestRequest{BuoyID: buoyA.ID, DO: 2.5, Timestamp: now, Operator: "t"})

	pend := res.WarningsCreated[0]
	verified, err := svc.Warnings.Verify(pend.ID, "operator", "req-1")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if verified.Status != domain.WarningStatusConfirmed {
		t.Fatalf("verified warning should be confirmed, got %s", verified.Status)
	}
	zoneAfter, _ := svc.Store.Zones().Get(zone.ID)
	if zoneAfter.Status != domain.ZoneStatusAerating {
		t.Fatalf("verified danger should start aeration, zone=%s", zoneAfter.Status)
	}
	// Double verify must conflict.
	if _, err := svc.Warnings.Verify(pend.ID, "operator", "req-2"); err == nil {
		t.Fatal("double verify must be rejected")
	}
}

func TestRestoreRequiresEligibility(t *testing.T) {
	cfg, _, svc := newTestServices(t)
	_, buoy := seedZoneAndBuoy(t, svc, "东区")
	now := time.Now().UTC()
	res, err := svc.Ingest.Process(IngestRequest{BuoyID: buoy.ID, DO: 2.5, Timestamp: now, Operator: "t"})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	// Not eligible: reject restore.
	if _, err := svc.Aeration.Restore(res.Zone.ID, "operator", "req"); err == nil {
		t.Fatal("restore before eligibility must be rejected")
	}

	// Make it eligible with a short sustained window.
	cfg.RestoreSustained = 2 * time.Second
	cfg.SamplePeriod = time.Second
	cfg.SamplePeriodTolerance = time.Hour
	var lastRecovery time.Time
	for _, dt := range []time.Duration{2 * time.Second, 4 * time.Second, 6 * time.Second} {
		ts := now.Add(dt)
		if _, err := svc.Ingest.Process(IngestRequest{
			BuoyID: buoy.ID, DO: 5.8, Timestamp: ts, Operator: "t",
		}); err != nil {
			t.Fatalf("recovery ingest at %v: %v", dt, err)
		}
		lastRecovery = ts
	}
	checkTime := lastRecovery.Add(time.Second)
	check := svc.Restore.RunOnce(checkTime)
	if check.RestoreEligible != 1 {
		t.Fatalf("expected zone restore-eligible, got %+v", check)
	}
	zone, _ := svc.Store.Zones().Get(res.Zone.ID)
	if !zone.RestoreEligible {
		t.Fatal("zone must be marked restore-eligible")
	}
	stopLog, err := svc.Aeration.Restore(res.Zone.ID, "operator", "req")
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if stopLog.Action != domain.AerationActionStop {
		t.Fatalf("restore should issue a stop command, got %s", stopLog.Action)
	}
	zone, _ = svc.Store.Zones().Get(res.Zone.ID)
	if zone.Status != domain.ZoneStatusRestored || zone.RestoreEligible {
		t.Fatalf("zone should be restored and not eligible, got %+v", zone)
	}
	// All warnings resolved.
	if n := svc.Store.Warnings().CountOpen(); n != 0 {
		t.Fatalf("all warnings should be resolved, %d open", n)
	}
}

func TestAerationFeedbackAndTimeout(t *testing.T) {
	_, _, svc := newTestServices(t)
	zone, buoy := seedZoneAndBuoy(t, svc, "东区")
	now := time.Now().UTC()
	res, err := svc.Ingest.Process(IngestRequest{BuoyID: buoy.ID, DO: 2.5, Timestamp: now, Operator: "t"})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	aerLog := res.AerationLog
	// Acknowledge then report started.
	updated, err := svc.Aeration.Feedback(aerLog.ID, domain.FeedbackAcknowledged, "dev", "req")
	if err != nil {
		t.Fatalf("ack: %v", err)
	}
	if updated.Status != domain.AeratorStatusStarting {
		t.Fatalf("ack should keep starting, got %s", updated.Status)
	}
	updated, err = svc.Aeration.Feedback(aerLog.ID, domain.FeedbackStarted, "dev", "req")
	if err != nil {
		t.Fatalf("started: %v", err)
	}
	if updated.Status != domain.AeratorStatusRunning {
		t.Fatalf("started feedback should reach running, got %s", updated.Status)
	}

	// New zone/buoy for the timeout path.
	_, buoy2 := seedZoneAndBuoy(t, svc, "西区")
	res2, _ := svc.Ingest.Process(IngestRequest{BuoyID: buoy2.ID, DO: 2.6, Timestamp: time.Now().UTC(), Operator: "t"})
	old := res2.AerationLog.CommandTime
	timeouts := svc.Aeration.CheckTimeouts(old.Add(time.Hour), "req")
	if timeouts != 1 {
		t.Fatalf("expected 1 timeout, got %d", timeouts)
	}
	timeoutLog, _ := svc.Store.Aeration().Get(res2.AerationLog.ID)
	if timeoutLog.Status != domain.AeratorStatusFault || timeoutLog.Feedback != domain.FeedbackTimeout {
		t.Fatalf("timeout log should be fault/timeout, got %s/%s", timeoutLog.Status, timeoutLog.Feedback)
	}
	_ = zone
}

func TestFarmLogCreateAndDeathAbnormal(t *testing.T) {
	_, _, svc := newTestServices(t)
	zone, _ := seedZoneAndBuoy(t, svc, "东区")
	log, err := svc.FarmLog.Create(FarmLogRequest{
		ZoneID: zone.ID, Date: "2026-08-25", FeedAmount: 500, DeathCount: 800, DiseaseNote: "败血症", Operator: "张师傅",
	})
	if err != nil {
		t.Fatalf("create farm log: %v", err)
	}
	if !log.DeathAbnormal {
		t.Fatal("800 deaths on 50000 stock must be abnormal")
	}
	// Duplicate date conflicts.
	if _, err := svc.FarmLog.Create(FarmLogRequest{
		ZoneID: zone.ID, Date: "2026-08-25", FeedAmount: 100, DeathCount: 1, Operator: "张师傅",
	}); err == nil {
		t.Fatal("duplicate zone+date must conflict")
	}
	// Different date succeeds.
	if _, err := svc.FarmLog.Create(FarmLogRequest{
		ZoneID: zone.ID, Date: "2026-08-24", FeedAmount: 100, DeathCount: 1, Operator: "张师傅",
	}); err != nil {
		t.Fatalf("second date: %v", err)
	}
	if len(svc.FarmLog.List("", 10)) != 2 {
		t.Fatal("expected 2 farm logs")
	}
}

func TestAuditTrailCoverage(t *testing.T) {
	_, _, svc := newTestServices(t)
	zone, buoy := seedZoneAndBuoy(t, svc, "东区")
	now := time.Now().UTC()
	_, _ = svc.Ingest.Process(IngestRequest{BuoyID: buoy.ID, DO: 2.5, Timestamp: now, Operator: "t"})
	_, _ = svc.FarmLog.Create(FarmLogRequest{ZoneID: zone.ID, Date: "2026-08-25", FeedAmount: 10, DeathCount: 1, Operator: "t"})

	entries := svc.Audit.List(100)
	actions := map[domain.AuditAction]bool{}
	for _, e := range entries {
		actions[e.Action] = true
	}
	for _, want := range []domain.AuditAction{
		domain.AuditZoneCreate, domain.AuditBuoyCreate, domain.AuditSampleIngest,
		domain.AuditWarningCreated, domain.AuditAerationStart, domain.AuditFarmLogCreate,
	} {
		if !actions[want] {
			t.Fatalf("audit trail missing action %s", want)
		}
	}
}

func TestOverviewAggregation(t *testing.T) {
	_, _, svc := newTestServices(t)
	zone, buoy := seedZoneAndBuoy(t, svc, "东区")
	_, _ = svc.Ingest.Process(IngestRequest{
		BuoyID: buoy.ID, DO: 2.5, Temperature: 24, Salinity: 31, PH: 8.1, Ammonia: 0.05,
		Timestamp: time.Now().UTC(), Operator: "t",
	})
	ov := svc.Overview.Get()
	if len(ov.Zones) != 1 {
		t.Fatalf("expected 1 zone, got %d", len(ov.Zones))
	}
	zo := ov.Zones[0]
	if zo.Zone.ID != zone.ID || zo.OpenWarningCount != 1 || zo.BuoyCount != 1 {
		t.Fatalf("unexpected zone overview: %+v", zo)
	}
	if ov.Totals.ActiveAerators != 1 {
		t.Fatalf("expected 1 active aerator, got %d", ov.Totals.ActiveAerators)
	}
}

func TestRestoreCheckerRunLoopCancels(t *testing.T) {
	_, _, svc := newTestServices(t)
	ctx, cancel := context.WithCancel(context.Background())
	svc.StartSweepers(ctx)
	cancel()
	// No panic, no leak assertion possible; just ensure it returns cleanly.
	time.Sleep(50 * time.Millisecond)
}
