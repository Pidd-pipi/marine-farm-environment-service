package domain

import (
	"testing"
	"time"
)

func TestAeratorStateMachine(t *testing.T) {
	legal := [][2]AeratorStatus{
		{AeratorStatusStopped, AeratorStatusStarting},
		{AeratorStatusStarting, AeratorStatusRunning},
		{AeratorStatusStarting, AeratorStatusFault},
		{AeratorStatusRunning, AeratorStatusStopping},
		{AeratorStatusRunning, AeratorStatusFault},
		{AeratorStatusStopping, AeratorStatusStopped},
		{AeratorStatusStopping, AeratorStatusFault},
		{AeratorStatusFault, AeratorStatusStopped},
	}
	for _, tr := range legal {
		if !CanAeratorTransition(tr[0], tr[1]) {
			t.Fatalf("transition %s -> %s should be legal", tr[0], tr[1])
		}
	}
	illegal := [][2]AeratorStatus{
		{AeratorStatusStopped, AeratorStatusRunning},
		{AeratorStatusStopped, AeratorStatusFault},
		{AeratorStatusRunning, AeratorStatusStarting},
		{AeratorStatusFault, AeratorStatusRunning},
	}
	for _, tr := range illegal {
		if CanAeratorTransition(tr[0], tr[1]) {
			t.Fatalf("transition %s -> %s should be illegal", tr[0], tr[1])
		}
	}
}

func TestAerationStartFeedbackLifecycle(t *testing.T) {
	now := time.Now().UTC()
	log, err := NewAerationLog("aer_1", "z1", "aerator_z1", AerationActionStart, TriggerAuto, "auto", now)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if log.Status != AeratorStatusStarting || log.Feedback != FeedbackNone {
		t.Fatalf("start command should be starting/none, got %s/%s", log.Status, log.Feedback)
	}
	status, err := log.ApplyFeedback(FeedbackStarted, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("started feedback: %v", err)
	}
	if status != AeratorStatusRunning || log.Status != AeratorStatusRunning {
		t.Fatalf("expected running, got %s", log.Status)
	}
	if _, err := log.ApplyFeedback(FeedbackStopped, now.Add(2*time.Minute)); err == nil {
		t.Fatal("double feedback must be rejected")
	}
}

func TestAerationTimeout(t *testing.T) {
	now := time.Now().UTC()
	log, _ := NewAerationLog("aer_1", "z1", "aerator_z1", AerationActionStart, TriggerAuto, "auto", now)
	if log.TimedOut(time.Minute, now.Add(30*time.Second)) {
		t.Fatal("must not time out before the window")
	}
	if !log.TimedOut(time.Minute, now.Add(2*time.Minute)) {
		t.Fatal("must time out after the window")
	}
	if err := log.MarkTimeout(now.Add(2 * time.Minute)); err != nil {
		t.Fatalf("mark timeout: %v", err)
	}
	if log.Status != AeratorStatusFault || log.Feedback != FeedbackTimeout {
		t.Fatalf("timeout must move to fault/timeout, got %s/%s", log.Status, log.Feedback)
	}
}

func TestAerationStopFeedback(t *testing.T) {
	now := time.Now().UTC()
	log, _ := NewAerationLog("aer_2", "z1", "aerator_z1", AerationActionStop, TriggerRestore, "stop", now)
	if log.Status != AeratorStatusStopping {
		t.Fatalf("stop command should be stopping, got %s", log.Status)
	}
	status, err := log.ApplyFeedback(FeedbackStopped, now.Add(time.Minute))
	if err != nil || status != AeratorStatusStopped {
		t.Fatalf("expected stopped, got %s err=%v", status, err)
	}
}

func TestInvalidFeedbackForAction(t *testing.T) {
	now := time.Now().UTC()
	log, _ := NewAerationLog("aer_3", "z1", "aerator_z1", AerationActionStart, TriggerAuto, "", now)
	if _, err := log.ApplyFeedback(FeedbackStopped, now); err == nil {
		t.Fatal("stopped feedback on a start command must be rejected")
	}
}
