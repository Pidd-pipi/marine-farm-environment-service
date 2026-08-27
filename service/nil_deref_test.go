package service

import (
	"testing"
	"time"

	"example.com/marine-farm-environment-service/domain"
)

// TestIngestFirstReportSafe pins the first-report path: a buoy that has
// never reported before must be accepted, not panic on a nil timestamp.
func TestIngestFirstReportSafe(t *testing.T) {
	_, _, svc := newTestServices(t)
	_, buoy := seedZoneAndBuoy(t, svc, "东区")
	_, err := svc.Ingest.Process(IngestRequest{
		BuoyID: buoy.ID, DO: 6.2, Temperature: 24, Salinity: 31, PH: 8.1, Ammonia: 0.05,
		Timestamp: time.Now().UTC(), Operator: "test",
	})
	if err != nil {
		t.Fatalf("first report should succeed: %v", err)
	}
}

// TestOverviewZoneWithoutSampleSafe pins the zero-sample path: a zone with
// no samples must render with LatestDO=-1 rather than dereferencing nil.
func TestOverviewZoneWithoutSampleSafe(t *testing.T) {
	_, st, svc := newTestServices(t)
	_, err := svc.Zones.Create(CreateZoneRequest{Name: "东区", Area: 100, Stock: 50000})
	if err != nil {
		t.Fatalf("create zone: %v", err)
	}
	_ = st
	ov := svc.Overview.Get()
	if len(ov.Zones) != 1 {
		t.Fatalf("expected 1 zone, got %d", len(ov.Zones))
	}
	if ov.Zones[0].LatestDO != -1 {
		t.Fatalf("zone without sample should have LatestDO=-1, got %v", ov.Zones[0].LatestDO)
	}
}

// TestOverviewZoneWithoutAerationSafe pins the zero-aeration path: a zone
// with samples but no aerator command must not dereference a nil aerator.
func TestOverviewZoneWithoutAerationSafe(t *testing.T) {
	_, st, svc := newTestServices(t)
	zone, err := svc.Zones.Create(CreateZoneRequest{Name: "东区", Area: 100, Stock: 50000})
	if err != nil {
		t.Fatalf("create zone: %v", err)
	}
	buoy, err := svc.Buoys.Create(CreateBuoyRequest{ZoneID: zone.ID, Name: "浮标"})
	if err != nil {
		t.Fatalf("create buoy: %v", err)
	}
	now := time.Now().UTC()
	sample := domain.NewWaterSample(st.NewID("sample"), buoy.ID, zone.ID, 6.0, 24, 31, 8.1, 0.05, now)
	if err := st.Samples().Create(sample, 100); err != nil {
		t.Fatalf("create sample: %v", err)
	}
	ov := svc.Overview.Get()
	if len(ov.Zones) != 1 {
		t.Fatalf("expected 1 zone, got %d", len(ov.Zones))
	}
	if ov.Zones[0].LatestDO != 6.0 {
		t.Fatalf("latest DO = %v, want 6.0", ov.Zones[0].LatestDO)
	}
}
