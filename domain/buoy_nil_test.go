package domain

import (
	"testing"
	"time"
)

// TestBuoyStaleNeverReportedSafe pins the zero-value path: a buoy that has
// never reported must be considered stale rather than panicking.
func TestBuoyStaleNeverReportedSafe(t *testing.T) {
	buoy := NewBuoy("buoy_1", "zone_1", "浮标", "", 0, 0, time.Now().UTC())
	// Must not panic: a never-reported buoy has a nil LastReportAt.
	if !buoy.Stale(time.Minute, time.Now().UTC()) {
		t.Fatalf("never-reported buoy should be stale")
	}
}
