package store

import (
	"testing"
	"time"

	"example.com/marine-farm-environment-service/domain"
)

func mkWarning(st *Store, typ domain.WarningType, status domain.WarningStatus, at time.Time) {
	rec := &domain.WarningRecord{
		ID: st.NewID("warning"), ZoneID: "zone_1", BuoyID: "buoy_1",
		Type: typ, Level: domain.WarningLevelWarning, Status: status,
		ReportedAt: at,
	}
	_ = st.Warnings().Create(rec)
}

// TestWarningFilterDoesNotPollute pins the no-aliasing contract: filtering
// must not overwrite the store's backing array.
func TestWarningFilterDoesNotPollute(t *testing.T) {
	st := NewMemoryStore()
	now := time.Now().UTC()
	mkWarning(st, domain.WarningTypeTempShock, domain.WarningStatusConfirmed, now)
	mkWarning(st, domain.WarningTypeDOLow, domain.WarningStatusConfirmed, now.Add(time.Minute))

	got := st.Warnings().List(WarningFilter{Type: string(domain.WarningTypeDOLow)})
	if len(got) != 1 {
		t.Fatalf("do_low filter = %d warnings, want 1", len(got))
	}
	again := st.Warnings().List(WarningFilter{Type: string(domain.WarningTypeTempShock)})
	if len(again) != 1 {
		t.Fatalf("temp_shock count = %d, want 1 (filter polluted store)", len(again))
	}
}

// TestWarningListReturnsCopy pins the pointer no-aliasing contract.
func TestWarningListReturnsCopy(t *testing.T) {
	st := NewMemoryStore()
	now := time.Now().UTC()
	vt := now.Add(time.Minute)
	rec := &domain.WarningRecord{
		ID: st.NewID("warning"), ZoneID: "zone_1", BuoyID: "buoy_1",
		Type: domain.WarningTypeDOLow, Level: domain.WarningLevelWarning,
		Status: domain.WarningStatusConfirmed, ReportedAt: now,
	}
	rec.VerifiedAt = &vt
	_ = st.Warnings().Create(rec)
	want := vt

	got := st.Warnings().List(WarningFilter{})
	*got[0].VerifiedAt = now.Add(48 * time.Hour)
	again := st.Warnings().List(WarningFilter{})
	if !again[0].VerifiedAt.Equal(want) {
		t.Fatalf("Warning List returned an aliased VerifiedAt pointer")
	}
}
