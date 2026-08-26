package domain

import (
	"testing"
	"time"
)

func TestZoneStateMachineTransitions(t *testing.T) {
	legal := [][2]ZoneStatus{
		{ZoneStatusNormal, ZoneStatusNormal},
		{ZoneStatusNormal, ZoneStatusWarning},
		{ZoneStatusNormal, ZoneStatusDanger},
		{ZoneStatusWarning, ZoneStatusNormal},
		{ZoneStatusWarning, ZoneStatusDanger},
		{ZoneStatusDanger, ZoneStatusWarning},
		{ZoneStatusDanger, ZoneStatusAerating},
		{ZoneStatusAerating, ZoneStatusDanger},
		{ZoneStatusAerating, ZoneStatusRestored},
		{ZoneStatusRestored, ZoneStatusNormal},
		{ZoneStatusRestored, ZoneStatusWarning},
		{ZoneStatusRestored, ZoneStatusDanger},
	}
	for _, tr := range legal {
		if !CanZoneTransition(tr[0], tr[1]) {
			t.Fatalf("transition %s -> %s should be legal", tr[0], tr[1])
		}
	}
	illegal := [][2]ZoneStatus{
		{ZoneStatusNormal, ZoneStatusAerating},
		{ZoneStatusNormal, ZoneStatusRestored},
		{ZoneStatusWarning, ZoneStatusAerating},
		{ZoneStatusWarning, ZoneStatusRestored},
		{ZoneStatusDanger, ZoneStatusRestored},
		{ZoneStatusDanger, ZoneStatusNormal},
		{ZoneStatusAerating, ZoneStatusNormal},
		{ZoneStatusAerating, ZoneStatusWarning},
		{ZoneStatusRestored, ZoneStatusAerating},
	}
	for _, tr := range illegal {
		if CanZoneTransition(tr[0], tr[1]) {
			t.Fatalf("transition %s -> %s should be illegal", tr[0], tr[1])
		}
	}
}

func TestZoneSetStatus(t *testing.T) {
	now := time.Now().UTC()
	z := NewFarmZone("zone_1", "东区", 100, 50000, 4, 3, now)
	if z.Status != ZoneStatusNormal {
		t.Fatalf("new zone should be normal, got %s", z.Status)
	}
	if err := z.SetStatus(ZoneStatusAerating, now); err == nil {
		t.Fatal("normal -> aerating must be rejected")
	}
	if err := z.SetStatus(ZoneStatusWarning, now); err != nil {
		t.Fatalf("normal -> warning should succeed: %v", err)
	}
	if err := z.SetStatus(ZoneStatusDanger, now); err != nil {
		t.Fatalf("warning -> danger should succeed: %v", err)
	}
	if err := z.SetStatus(ZoneStatusAerating, now); err != nil {
		t.Fatalf("danger -> aerating should succeed: %v", err)
	}
	if err := z.SetStatus(ZoneStatusRestored, now); err != nil {
		t.Fatalf("aerating -> restored should succeed: %v", err)
	}
	if err := z.SetStatus(ZoneStatusNormal, now); err != nil {
		t.Fatalf("restored -> normal should succeed: %v", err)
	}
}

func TestTargetStatusFromDO(t *testing.T) {
	now := time.Now().UTC()
	z := NewFarmZone("zone_1", "东区", 100, 50000, 4, 3, now)
	cases := []struct {
		do   float64
		want ZoneStatus
	}{
		{6.5, ZoneStatusNormal},
		{4.0, ZoneStatusNormal},
		{3.9, ZoneStatusWarning},
		{3.0, ZoneStatusWarning},
		{2.9, ZoneStatusDanger},
		{0.5, ZoneStatusDanger},
	}
	for _, c := range cases {
		if got := z.TargetStatusFromDO(c.do); got != c.want {
			t.Fatalf("TargetStatusFromDO(%v) = %s, want %s", c.do, got, c.want)
		}
	}
}

func TestStatusAge(t *testing.T) {
	now := time.Now().UTC()
	z := NewFarmZone("zone_1", "东区", 100, 50000, 4, 3, now.Add(-time.Hour))
	age := z.StatusAge(now)
	if age != time.Hour {
		t.Fatalf("expected 1h age, got %v", age)
	}
}

func TestRestoreEligibility(t *testing.T) {
	now := time.Now().UTC()
	z := NewFarmZone("zone_1", "东区", 100, 50000, 4, 3, now)
	if z.RestoreEligible {
		t.Fatal("new zone must not be restore-eligible")
	}
	z.MarkRestoreEligible(now)
	if !z.RestoreEligible || z.RestoreEligibleAt == nil {
		t.Fatal("MarkRestoreEligible must set the flag and timestamp")
	}
	z.ClearRestoreEligibility()
	if z.RestoreEligible || z.RestoreEligibleAt != nil {
		t.Fatal("ClearRestoreEligibility must clear the flag")
	}
}
