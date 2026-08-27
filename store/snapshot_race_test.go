package store

import (
	"sync"
	"testing"
	"time"

	"example.com/marine-farm-environment-service/domain"
)

// TestSnapshotConcurrentCreateNoRace exercises the read path against
// concurrent writes; with the race detector enabled any unsynchronised
// access between Snapshot and SampleStore.Create is reported.
func TestSnapshotConcurrentCreateNoRace(t *testing.T) {
	st := NewMemoryStore()
	now := time.Now().UTC()
	zone := domain.NewFarmZone(st.NewID("zone"), "东区", 100, 50000, 4, 3, now)
	if err := st.Zones().Create(zone); err != nil {
		t.Fatalf("create zone: %v", err)
	}
	buoy := domain.NewBuoy(st.NewID("buoy"), zone.ID, "浮标", "", 0, 0, now)
	if err := st.Buoys().Create(buoy); err != nil {
		t.Fatalf("create buoy: %v", err)
	}

	const writers = 4
	const readers = 4
	const perWriter = 120
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			<-start
			for j := 0; j < perWriter; j++ {
				sample := domain.NewWaterSample(
					st.NewID("sample"), buoy.ID, zone.ID,
					6.0, 24, 31, 8.1, 0.05, now.Add(time.Duration(j)*time.Second),
				)
				_ = st.Samples().Create(sample, 2000)
			}
		}(i)
	}
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < perWriter; j++ {
				_ = st.Snapshot()
			}
		}()
	}

	close(start)
	wg.Wait()

	got := st.Snapshot()
	if len(got.Samples) != writers*perWriter {
		t.Fatalf("sample count = %d, want %d", len(got.Samples), writers*perWriter)
	}
}

// TestCountConcurrentCreateNoRace exercises Count against concurrent writes.
func TestCountConcurrentCreateNoRace(t *testing.T) {
	st := NewMemoryStore()
	now := time.Now().UTC()
	zone := domain.NewFarmZone(st.NewID("zone"), "东区", 100, 50000, 4, 3, now)
	if err := st.Zones().Create(zone); err != nil {
		t.Fatalf("create zone: %v", err)
	}
	buoy := domain.NewBuoy(st.NewID("buoy"), zone.ID, "浮标", "", 0, 0, now)
	if err := st.Buoys().Create(buoy); err != nil {
		t.Fatalf("create buoy: %v", err)
	}

	const writers = 4
	const readers = 4
	const perWriter = 120
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < perWriter; j++ {
				sample := domain.NewWaterSample(
					st.NewID("sample"), buoy.ID, zone.ID,
					6.0, 24, 31, 8.1, 0.05, now.Add(time.Duration(j)*time.Second),
				)
				_ = st.Samples().Create(sample, 2000)
			}
		}()
	}
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < perWriter; j++ {
				_ = st.Count()
			}
		}()
	}

	close(start)
	wg.Wait()

	got := st.Count()
	if got["samples"] != writers*perWriter {
		t.Fatalf("sample count = %d, want %d", got["samples"], writers*perWriter)
	}
}

// TestLatestByZoneReturnsCopy pins the no-aliasing contract for the
// latest-sample read path.
func TestLatestByZoneReturnsCopy(t *testing.T) {
	st := NewMemoryStore()
	now := time.Now().UTC()
	zone := domain.NewFarmZone(st.NewID("zone"), "东区", 100, 50000, 4, 3, now)
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

	got, ok := st.Samples().LatestByZone(zone.ID)
	if !ok {
		t.Fatalf("latest sample missing")
	}
	got.Violations[0] = "tampered"
	again, _ := st.Samples().LatestByZone(zone.ID)
	if again.Violations[0] != "do_low" {
		t.Fatalf("LatestByZone returned an aliased Violations slice: %q", again.Violations[0])
	}
}

// TestListByZoneReturnsCopy pins the no-aliasing contract for the list
// read path.
func TestListByZoneReturnsCopy(t *testing.T) {
	st := NewMemoryStore()
	now := time.Now().UTC()
	zone := domain.NewFarmZone(st.NewID("zone"), "东区", 100, 50000, 4, 3, now)
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

	got := st.Samples().ListByZone(zone.ID, 10)
	if len(got) != 1 {
		t.Fatalf("expected 1 sample, got %d", len(got))
	}
	got[0].Violations[0] = "tampered"
	again := st.Samples().ListByZone(zone.ID, 10)
	if again[0].Violations[0] != "do_low" {
		t.Fatalf("ListByZone returned an aliased Violations slice: %q", again[0].Violations[0])
	}
}
