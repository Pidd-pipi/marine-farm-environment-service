package domain

import (
	"testing"
	"time"
)

// TestCrossValidationDoesNotMutateInput pins the no-side-effect contract:
// evaluating a cross validation must not reorder the caller's sample slice.
func TestCrossValidationDoesNotMutateInput(t *testing.T) {
	now := time.Now().UTC()
	samples := []WaterSample{
		{ID: "s_new", BuoyID: "b2", ZoneID: "z1", DO: 6.0, Timestamp: now},
		{ID: "s_old", BuoyID: "b2", ZoneID: "z1", DO: 6.0, Timestamp: now.Add(-2 * time.Minute)},
	}
	_ = EvaluateCrossValidation(NeighbourSamples{
		ZoneID: "z1", BuoyID: "b1", From: now.Add(-5 * time.Minute), To: now, Samples: samples,
	}, 4.0)

	if samples[0].ID != "s_new" || samples[1].ID != "s_old" {
		t.Fatalf("EvaluateCrossValidation reordered the caller slice: %+v", samples)
	}
}
