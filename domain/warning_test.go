package domain

import (
	"testing"
	"time"
)

func TestWarningVerifyResolve(t *testing.T) {
	now := time.Now().UTC()
	s := NewWaterSample("s1", "b1", "z1", 2.5, 24.5, 31.0, 8.1, 0.06, now)
	rec := NewWarningRecord("w1", "z1", "b1", WarningTypeDOLow, WarningLevelDanger, s, "low", now)
	if rec.Status != WarningStatusConfirmed {
		t.Fatalf("new warning should be confirmed, got %s", rec.Status)
	}
	rec.Pending()
	if rec.Status != WarningStatusPending {
		t.Fatal("Pending should set pending status")
	}
	if err := rec.Verify(now); err != nil {
		t.Fatalf("verify should succeed: %v", err)
	}
	if err := rec.Verify(now); err == nil {
		t.Fatal("double verify must be rejected")
	}
	if err := rec.Resolve(now); err != nil {
		t.Fatalf("resolve should succeed: %v", err)
	}
	if rec.Status != WarningStatusResolved || rec.ResolvedAt == nil {
		t.Fatal("resolve must set resolved status and timestamp")
	}
}

func TestCrossValidationContradicted(t *testing.T) {
	now := time.Now().UTC()
	samples := []WaterSample{
		{ID: "n1", BuoyID: "b2", ZoneID: "z1", DO: 6.0, Timestamp: now.Add(-2 * time.Minute)},
		{ID: "n2", BuoyID: "b2", ZoneID: "z1", DO: 5.8, Timestamp: now.Add(-8 * time.Minute)},
	}
	res := EvaluateCrossValidation(NeighbourSamples{
		ZoneID: "z1", BuoyID: "b1",
		From: now.Add(-15 * time.Minute), To: now,
		Samples: samples,
	}, 4.0)
	if !res.Contradicted || !res.Checked {
		t.Fatalf("expected contradicted cross-check, got %+v", res)
	}
	if res.EvidenceSampleID != "n1" {
		t.Fatalf("expected newest evidence n1, got %s", res.EvidenceSampleID)
	}
}

func TestCrossValidationNoEvidence(t *testing.T) {
	now := time.Now().UTC()
	res := EvaluateCrossValidation(NeighbourSamples{
		ZoneID: "z1", BuoyID: "b1",
		From: now.Add(-15 * time.Minute), To: now,
		Samples: []WaterSample{
			{ID: "l1", BuoyID: "b2", ZoneID: "z1", DO: 2.8, Timestamp: now.Add(-2 * time.Minute)},
		},
	}, 4.0)
	if res.Contradicted {
		t.Fatalf("low neighbour DO must not contradict, got %+v", res)
	}
}

func TestCrossValidationIgnoresSelf(t *testing.T) {
	now := time.Now().UTC()
	res := EvaluateCrossValidation(NeighbourSamples{
		ZoneID: "z1", BuoyID: "b1",
		From: now.Add(-15 * time.Minute), To: now,
		Samples: []WaterSample{
			{ID: "self1", BuoyID: "b1", ZoneID: "z1", DO: 6.0, Timestamp: now.Add(-2 * time.Minute)},
		},
	}, 4.0)
	if res.Contradicted {
		t.Fatal("the reporting buoy's own sample must not count as evidence")
	}
}

func TestCrossValidationOutOfWindow(t *testing.T) {
	now := time.Now().UTC()
	res := EvaluateCrossValidation(NeighbourSamples{
		ZoneID: "z1", BuoyID: "b1",
		From: now.Add(-15 * time.Minute), To: now,
		Samples: []WaterSample{
			{ID: "old", BuoyID: "b2", ZoneID: "z1", DO: 6.0, Timestamp: now.Add(-30 * time.Minute)},
		},
	}, 4.0)
	if res.Contradicted {
		t.Fatal("out-of-window evidence must not count")
	}
}
