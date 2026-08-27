package service

import (
	"context"
	"testing"
	"time"
)

// TestRestoreCheckerStopsOnCancel pins the lifecycle contract: the
// background checker goroutine must exit once its context is cancelled.
func TestRestoreCheckerStopsOnCancel(t *testing.T) {
	_, _, svc := newTestServices(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		svc.Restore.Run(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// pass
	case <-time.After(3 * time.Second):
		t.Fatalf("restore checker did not stop after context cancellation")
	}
}

// TestStartSweepersStopsOnCancel pins the sweeper lifecycle contract at the
// service level: the background checker must stop when its context is
// cancelled.
func TestStartSweepersStopsOnCancel(t *testing.T) {
	_, _, svc := newTestServices(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := svc.StartSweepers(ctx)

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// pass
	case <-time.After(3 * time.Second):
		t.Fatalf("sweeper did not stop after context cancellation")
	}
}
