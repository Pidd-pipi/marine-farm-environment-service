package service

import (
	"testing"

	"example.com/marine-farm-environment-service/domain"
)

// TestBootstrapZone3Aerating pins the seed contract: zone 3 is seeded
// through the full state chain and must end up in the aerating state.
func TestBootstrapZone3Aerating(t *testing.T) {
	cfg, st, _ := newTestServices(t)
	boot := NewBootstrap(cfg, st)
	if err := boot.SeedIfEmpty(); err != nil {
		t.Fatalf("seed: %v", err)
	}
	for _, z := range st.Zones().List() {
		if z.Name == "南区·3号养殖区" {
			if z.Status != domain.ZoneStatusAerating {
				t.Fatalf("zone 3 status = %s, want aerating", z.Status)
			}
			return
		}
	}
	t.Fatalf("zone 3 not found")
}
