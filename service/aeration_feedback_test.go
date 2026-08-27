package service

import (
	"testing"

	"example.com/marine-farm-environment-service/domain"
)

// TestFeedbackReachesRunning pins the feedback contract: applying the
// started feedback must persist the running state, not a stale pre-feedback
// copy.
func TestFeedbackReachesRunning(t *testing.T) {
	_, _, svc := newTestServices(t)
	zone, _ := seedZoneAndBuoy(t, svc, "东区")
	log, err := svc.Aeration.Start(zone.ID, domain.TriggerManual, "test", "req")
	if err != nil {
		t.Fatalf("start aeration: %v", err)
	}
	_, err = svc.Aeration.Feedback(log.ID, domain.FeedbackStarted, "test", "req")
	if err != nil {
		t.Fatalf("apply started feedback: %v", err)
	}
	got, err := svc.Store.Aeration().Get(log.ID)
	if err != nil {
		t.Fatalf("get aeration: %v", err)
	}
	if got.Status != domain.AeratorStatusRunning {
		t.Fatalf("aeration status = %s, want running (feedback lost)", got.Status)
	}
}
