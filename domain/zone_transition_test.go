package domain

import "testing"

// TestZoneDangerToAeratingTransition pins the state-machine edge: a danger
// zone must be allowed to move into the aerating state.
func TestZoneDangerToAeratingTransition(t *testing.T) {
	if !CanZoneTransition(ZoneStatusDanger, ZoneStatusAerating) {
		t.Fatalf("danger -> aerating transition should be legal")
	}
}
