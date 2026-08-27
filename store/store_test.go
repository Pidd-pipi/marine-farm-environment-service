package store

import (
	"path/filepath"
	"testing"
	"time"

	"example.com/marine-farm-environment-service/domain"
)

func TestZoneStoreCRUD(t *testing.T) {
	st := NewMemoryStore()
	now := time.Now().UTC()
	z := domain.NewFarmZone(st.NewID("zone"), "东区", 100, 50000, 4, 3, now)
	if err := st.Zones().Create(z); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := st.Zones().Get(z.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "东区" {
		t.Fatalf("unexpected zone: %+v", got)
	}
	if _, err := st.Zones().Get("missing"); err == nil {
		t.Fatal("missing zone must return not found")
	}
	if st.Zones().Count() != 1 {
		t.Fatalf("expected 1 zone, got %d", st.Zones().Count())
	}
}

func TestSampleStoreTrim(t *testing.T) {
	st := NewMemoryStore()
	now := time.Now().UTC()
	z := domain.NewFarmZone(st.NewID("zone"), "东区", 100, 50000, 4, 3, now)
	_ = st.Zones().Create(z)
	b := domain.NewBuoy(st.NewID("buoy"), z.ID, "浮标", "", 0, 0, now)
	_ = st.Buoys().Create(b)

	for i := 0; i < 10; i++ {
		s := domain.NewWaterSample(st.NewID("sample"), b.ID, z.ID, 6.0, 24, 31, 8.1, 0.05, now.Add(time.Duration(i)*time.Minute))
		if err := st.Samples().Create(s, 5); err != nil {
			t.Fatalf("create sample: %v", err)
		}
	}
	list := st.Samples().ListByBuoy(b.ID, 0)
	if len(list) != 5 {
		t.Fatalf("expected trim to 5 samples, got %d", len(list))
	}
	// Newest first.
	if !list[0].Timestamp.After(list[len(list)-1].Timestamp) {
		t.Fatal("samples must be sorted newest first")
	}
}

func TestWarningStoreFilter(t *testing.T) {
	st := NewMemoryStore()
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		rec := &domain.WarningRecord{
			ID: st.NewID("warning"), ZoneID: "z1", BuoyID: "b1",
			Type: domain.WarningTypeDOLow, Level: domain.WarningLevelDanger,
			Status: domain.WarningStatusConfirmed, ReportedAt: now.Add(time.Duration(i) * time.Minute),
		}
		if err := st.Warnings().Create(rec); err != nil {
			t.Fatalf("create: %v", err)
		}
	}
	pend := &domain.WarningRecord{
		ID: st.NewID("warning"), ZoneID: "z2", BuoyID: "b2",
		Type: domain.WarningTypeTempShock, Level: domain.WarningLevelWarning,
		Status: domain.WarningStatusPending, ReportedAt: now,
	}
	if err := st.Warnings().Create(pend); err != nil {
		t.Fatalf("create: %v", err)
	}
	if n := st.Warnings().CountOpen(); n != 4 {
		t.Fatalf("expected 4 open warnings, got %d", n)
	}
	if n := st.Warnings().CountByStatus(domain.WarningStatusPending); n != 1 {
		t.Fatalf("expected 1 pending, got %d", n)
	}
	filtered := st.Warnings().List(WarningFilter{ZoneID: "z1", Status: "confirmed"})
	if len(filtered) != 3 {
		t.Fatalf("expected 3 confirmed in z1, got %d", len(filtered))
	}
}

func TestPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "state.json")
	st := NewStore(file)
	now := time.Now().UTC()
	z := domain.NewFarmZone(st.NewID("zone"), "东区", 100, 50000, 4, 3, now)
	if err := st.Zones().Create(z); err != nil {
		t.Fatalf("create: %v", err)
	}
	bu := domain.NewBuoy(st.NewID("buoy"), z.ID, "浮标", "", 0, 0, now)
	if err := st.Buoys().Create(bu); err != nil {
		t.Fatalf("create: %v", err)
	}

	st2 := NewStore(file)
	if err := st2.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	got, err := st2.Zones().Get(z.ID)
	if err != nil {
		t.Fatalf("reload get: %v", err)
	}
	if got.Name != "东区" {
		t.Fatalf("reloaded zone mismatch: %+v", got)
	}
	if st2.Buoys().Count() != 1 {
		t.Fatalf("reloaded buoy count mismatch")
	}
	// IDs must continue from the persisted sequence.
	z2 := domain.NewFarmZone(st2.NewID("zone"), "西区", 50, 10000, 4, 3, now)
	if z2.ID == z.ID {
		t.Fatal("new id must not collide with persisted id")
	}
}

func TestStoreIsEmptyAndReset(t *testing.T) {
	st := NewMemoryStore()
	if !st.IsEmpty() {
		t.Fatal("fresh store must be empty")
	}
	now := time.Now().UTC()
	z := domain.NewFarmZone(st.NewID("zone"), "东区", 100, 50000, 4, 3, now)
	_ = st.Zones().Create(z)
	if st.IsEmpty() {
		t.Fatal("store with a zone must not be empty")
	}
	st.Reset()
	if !st.IsEmpty() {
		t.Fatal("reset must empty the store")
	}
}
