package store

import (
	"testing"
	"time"

	"example.com/marine-farm-environment-service/domain"
)

// TestZoneListReturnsCopy pins the no-aliasing contract for zone lists.
func TestZoneListReturnsCopy(t *testing.T) {
	st := NewMemoryStore()
	now := time.Now().UTC()
	ts := now.Add(time.Hour)
	zone := domain.NewFarmZone(st.NewID("zone"), "东区", 100, 50000, 4, 3, now)
	zone.RestoreEligible = true
	zone.RestoreEligibleAt = &ts
	if err := st.Zones().Create(zone); err != nil {
		t.Fatalf("create zone: %v", err)
	}
	want := ts

	got := st.Zones().List()
	*got[0].RestoreEligibleAt = now.Add(48 * time.Hour)
	again := st.Zones().List()
	if !again[0].RestoreEligibleAt.Equal(want) {
		t.Fatalf("Zone List returned an aliased RestoreEligibleAt pointer")
	}
}

// TestZoneGetReturnsCopy pins the no-aliasing contract for zone reads.
func TestZoneGetReturnsCopy(t *testing.T) {
	st := NewMemoryStore()
	now := time.Now().UTC()
	ts := now.Add(time.Hour)
	zone := domain.NewFarmZone(st.NewID("zone"), "东区", 100, 50000, 4, 3, now)
	zone.RestoreEligible = true
	zone.RestoreEligibleAt = &ts
	if err := st.Zones().Create(zone); err != nil {
		t.Fatalf("create zone: %v", err)
	}
	want := ts

	got, err := st.Zones().Get(zone.ID)
	if err != nil {
		t.Fatalf("get zone: %v", err)
	}
	*got.RestoreEligibleAt = now.Add(48 * time.Hour)
	again, _ := st.Zones().Get(zone.ID)
	if !again.RestoreEligibleAt.Equal(want) {
		t.Fatalf("Zone Get returned an aliased RestoreEligibleAt pointer")
	}
}
