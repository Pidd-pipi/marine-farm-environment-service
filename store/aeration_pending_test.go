package store

import (
	"testing"
	"time"

	"example.com/marine-farm-environment-service/domain"
)

// TestListPendingIncludesStarting pins the pending contract: an aerator in
// the starting state still awaits terminal feedback and must be listed.
func TestListPendingIncludesStarting(t *testing.T) {
	st := NewMemoryStore()
	now := time.Now().UTC()
	log, err := domain.NewAerationLog(st.NewID("aeration"), "zone_1", "aerator_1",
		domain.AerationActionStart, domain.TriggerAuto, "", now)
	if err != nil {
		t.Fatalf("new aeration: %v", err)
	}
	if err := st.Aeration().Create(log); err != nil {
		t.Fatalf("create aeration: %v", err)
	}
	pending := st.Aeration().ListPending()
	if len(pending) != 1 {
		t.Fatalf("pending aeration count = %d, want 1 (starting state omitted)", len(pending))
	}
}
