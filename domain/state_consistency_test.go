package domain

import (
	"testing"
	"time"
)

// TestCanAeratorRunningToStopping pins the aerator transition table: a
// running aerator must be able to move into stopping.
func TestCanAeratorRunningToStopping(t *testing.T) {
	if !CanAeratorTransition(AeratorStatusRunning, AeratorStatusStopping) {
		t.Fatalf("running -> stopping transition should be legal")
	}
}

// TestStopFeedbackReachesStopped pins the feedback contract: a stop
// command's stopped feedback must reach the stopped state.
func TestStopFeedbackReachesStopped(t *testing.T) {
	now := time.Now().UTC()
	log, err := NewAerationLog("aer_1", "zone_1", "aerator_1", AerationActionStop, TriggerManual, "", now)
	if err != nil {
		t.Fatalf("new aeration: %v", err)
	}
	status, err := log.ApplyFeedback(FeedbackStopped, now)
	if err != nil {
		t.Fatalf("apply stopped: %v", err)
	}
	if status != AeratorStatusStopped {
		t.Fatalf("status = %s, want stopped", status)
	}
}

// TestDeathAbnormalBoundary pins the abnormal-death threshold: exactly 1%
// of stock is not abnormal (must be strictly greater).
func TestDeathAbnormalBoundary(t *testing.T) {
	if EvaluateDeathAbnormal(500, 50000, 0.01) {
		t.Fatalf("death exactly at 1%% should not be abnormal")
	}
}
