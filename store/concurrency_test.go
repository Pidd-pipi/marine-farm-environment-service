package store

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"example.com/marine-farm-environment-service/domain"
)

func TestReadInterfacesReturnDeepCopies(t *testing.T) {
	st := NewMemoryStore()
	now := time.Now().UTC()
	zone := domain.NewFarmZone(st.NewID("zone"), "东区", 100, 50000, 4, 3, now)
	ts := now.Add(time.Hour)
	zone.RestoreEligible = true
	zone.RestoreEligibleAt = &ts
	if err := st.Zones().Create(zone); err != nil {
		t.Fatalf("create zone: %v", err)
	}
	buoy := domain.NewBuoy(st.NewID("buoy"), zone.ID, "浮标", "", 0, 0, now)
	if err := st.Buoys().Create(buoy); err != nil {
		t.Fatalf("create buoy: %v", err)
	}
	sample := domain.NewWaterSample(st.NewID("sample"), buoy.ID, zone.ID, 6.0, 24, 31, 8.1, 0.05, now)
	sample.Violations = []string{"do_low"}
	if err := st.Samples().Create(sample, 100); err != nil {
		t.Fatalf("create sample: %v", err)
	}
	warning := &domain.WarningRecord{
		ID: st.NewID("warning"), ZoneID: zone.ID, BuoyID: buoy.ID,
		Type: domain.WarningTypeDOLow, Level: domain.WarningLevelWarning,
		Status: domain.WarningStatusConfirmed, ReportedAt: now,
	}
	vt := now
	warning.VerifiedAt = &vt
	if err := st.Warnings().Create(warning); err != nil {
		t.Fatalf("create warning: %v", err)
	}
	aer, err := domain.NewAerationLog(st.NewID("aeration"), zone.ID, "aerator_"+zone.ID, domain.AerationActionStart, domain.TriggerManual, "", now)
	if err != nil {
		t.Fatalf("new aeration: %v", err)
	}
	ft := now
	aer.FeedbackAt = &ft
	if err := st.Aeration().Create(aer); err != nil {
		t.Fatalf("create aeration: %v", err)
	}

	// Mutating a returned sample must not affect the repository.
	gotSample, err := st.Samples().Get(sample.ID)
	if err != nil {
		t.Fatalf("get sample: %v", err)
	}
	gotSample.Violations[0] = "tampered"
	again, _ := st.Samples().Get(sample.ID)
	if again.Violations[0] != "do_low" {
		t.Fatalf("sample Violations slice aliased the store, got %q", again.Violations[0])
	}

	// Mutating a returned zone pointer must not affect the repository.
	gotZone, _ := st.Zones().Get(zone.ID)
	*gotZone.RestoreEligibleAt = now.Add(48 * time.Hour)
	againZone, _ := st.Zones().Get(zone.ID)
	if !againZone.RestoreEligibleAt.Equal(ts) {
		t.Fatalf("zone RestoreEligibleAt pointer aliased the store")
	}

	// Mutating a returned warning pointer must not affect the repository.
	gotWarning, _ := st.Warnings().Get(warning.ID)
	*gotWarning.VerifiedAt = now.Add(48 * time.Hour)
	againWarning, _ := st.Warnings().Get(warning.ID)
	if !againWarning.VerifiedAt.Equal(vt) {
		t.Fatalf("warning VerifiedAt pointer aliased the store")
	}

	// Mutating a returned aeration pointer must not affect the repository.
	gotAer, _ := st.Aeration().Get(aer.ID)
	*gotAer.FeedbackAt = now.Add(48 * time.Hour)
	againAer, _ := st.Aeration().Get(aer.ID)
	if !againAer.FeedbackAt.Equal(ft) {
		t.Fatalf("aeration FeedbackAt pointer aliased the store")
	}
}

func TestConcurrentReadsAndWrites(t *testing.T) {
	st := NewMemoryStore()
	now := time.Now().UTC()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				zone := domain.NewFarmZone(st.NewID("zone"), "并发区", 100, 50000, 4, 3, now.Add(time.Duration(n)*time.Millisecond))
				_ = st.Zones().Create(zone)
				_, _ = st.Zones().Get(zone.ID)
				_ = st.Zones().List()
				_ = st.Zones().Count()
			}
		}(i)
	}
	wg.Wait()
	if st.Zones().Count() != 8*50 {
		t.Fatalf("expected %d zones, got %d", 8*50, st.Zones().Count())
	}
}

func TestConcurrentSaveSerialization(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "state.json")
	st := NewStore(file)
	now := time.Now().UTC()
	zone := domain.NewFarmZone(st.NewID("zone"), "持久化", 100, 50000, 4, 3, now)
	if err := st.Zones().Create(zone); err != nil {
		t.Fatalf("create: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if err := st.Save(); err != nil {
					t.Errorf("save: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()

	reloaded := NewStore(file)
	if err := reloaded.Load(); err != nil {
		t.Fatalf("reload after concurrent saves: %v", err)
	}
	if reloaded.Zones().Count() != 1 {
		t.Fatalf("expected 1 zone after reload, got %d", reloaded.Zones().Count())
	}
}

func TestCorruptSnapshotBackedUpAndDegraded(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "state.json")
	if err := os.WriteFile(file, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write corrupt snapshot: %v", err)
	}

	st := NewStore(file)
	if err := st.Load(); err != nil {
		t.Fatalf("Load must degrade gracefully, got %v", err)
	}
	if !st.IsEmpty() {
		t.Fatalf("corrupt snapshot must degrade to an empty store")
	}
	if _, err := os.Stat(file + ".bak"); err != nil {
		t.Fatalf("corrupt snapshot must be backed up to .bak: %v", err)
	}
}
