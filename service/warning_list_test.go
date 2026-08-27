package service

import (
	"testing"
	"time"

	"example.com/marine-farm-environment-service/domain"
	"example.com/marine-farm-environment-service/store"
)

// TestWarningServiceListNoDuplicates pins the list contract: the service
// must return each warning exactly once.
func TestWarningServiceListNoDuplicates(t *testing.T) {
	_, st, svc := newTestServices(t)
	now := time.Now().UTC()
	for i := 0; i < 2; i++ {
		rec := &domain.WarningRecord{
			ID: st.NewID("warning"), ZoneID: "zone_1", BuoyID: "buoy_1",
			Type: domain.WarningTypeDOLow, Level: domain.WarningLevelWarning,
			Status: domain.WarningStatusConfirmed, ReportedAt: now.Add(time.Duration(i) * time.Minute),
		}
		_ = st.Warnings().Create(rec)
	}
	got := svc.Warnings.List(store.WarningFilter{})
	if len(got) != 2 {
		t.Fatalf("warning list = %d entries, want 2 (no duplicates)", len(got))
	}
}
