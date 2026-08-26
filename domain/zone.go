package domain

import (
	"fmt"
	"time"
)

// FarmZone is a marine-farm farming zone (养殖区). Each zone owns one or
// more monitoring buoys, a dissolved-oxygen threshold configuration and a
// lifecycle status driven by the zone state machine.
type FarmZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	// Area is the farming area in mu (亩).
	Area float64 `json:"area"`

	// Stock is the current stock count of the zone (存塘量), used by the
	// death-abnormal rule of farm logs.
	Stock int `json:"stock"`

	// DOWarnThreshold is the dissolved-oxygen level below which the zone
	// enters warning (default 4 mg/L).
	DOWarnThreshold float64 `json:"do_warn_threshold"`

	// DODangerThreshold is the dissolved-oxygen level below which the zone
	// enters danger (default 3 mg/L).
	DODangerThreshold float64 `json:"do_danger_threshold"`

	// Status is the current lifecycle status of the zone.
	Status ZoneStatus `json:"status"`

	// StatusSince records when the current status was entered.
	StatusSince time.Time `json:"status_since"`

	// RestoreEligible is set by the background restore checker when the
	// dissolved-oxygen recovery condition has been sustained long enough
	// and the operator is allowed to confirm restore.
	RestoreEligible bool `json:"restore_eligible"`

	// RestoreEligibleAt records when the zone became restore-eligible.
	RestoreEligibleAt *time.Time `json:"restore_eligible_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewFarmZone builds a farm zone with sensible defaults.
func NewFarmZone(id, name string, area float64, stock int, warnThreshold, dangerThreshold float64, now time.Time) *FarmZone {
	return &FarmZone{
		ID:                id,
		Name:              name,
		Area:              area,
		Stock:             stock,
		DOWarnThreshold:   warnThreshold,
		DODangerThreshold: dangerThreshold,
		Status:            ZoneStatusNormal,
		StatusSince:       now,
		RestoreEligible:   false,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// SetStatus moves the zone to a new status, validating the transition
// against the state machine. The status timestamp is refreshed on every
// successful transition.
func (z *FarmZone) SetStatus(to ZoneStatus, now time.Time) error {
	if !to.Valid() {
		return InvalidInput("invalid zone status %q", to)
	}
	if z.Status == to {
		return nil
	}
	if !CanZoneTransition(z.Status, to) {
		return Conflict("illegal zone state transition %s -> %s", z.Status, to)
	}
	z.Status = to
	z.StatusSince = now
	z.UpdatedAt = now
	return nil
}

// Touch refreshes the updated timestamp without changing the status.
func (z *FarmZone) Touch(now time.Time) {
	z.UpdatedAt = now
}

// ClearRestoreEligibility resets the restore-eligible flag.
func (z *FarmZone) ClearRestoreEligibility() {
	z.RestoreEligible = false
	z.RestoreEligibleAt = nil
}

// MarkRestoreEligible sets the restore-eligible flag and timestamp.
func (z *FarmZone) MarkRestoreEligible(now time.Time) {
	if !z.RestoreEligible {
		z.RestoreEligible = true
		ts := now
		z.RestoreEligibleAt = &ts
	}
	z.UpdatedAt = now
}

// TargetStatusFromDO computes the zone status a dissolved-oxygen reading
// should drive the zone towards:
//
//	do < dangerThreshold : danger
//	do < warnThreshold  : warning
//	otherwise           : normal
func (z *FarmZone) TargetStatusFromDO(do float64) ZoneStatus {
	switch {
	case do < z.DODangerThreshold:
		return ZoneStatusDanger
	case do < z.DOWarnThreshold:
		return ZoneStatusWarning
	default:
		return ZoneStatusNormal
	}
}

// zoneTransitionTable encodes the legal transitions of the farm-zone state
// machine. Self-transitions are always legal.
//
//	normal   → warning, danger
//	warning  → normal, danger
//	danger   → warning, aerating
//	aerating → danger, restored
//	restored → normal, warning, danger
var zoneTransitionTable = map[ZoneStatus]map[ZoneStatus]bool{
	ZoneStatusNormal: {
		ZoneStatusWarning: true,
		ZoneStatusDanger:  true,
	},
	ZoneStatusWarning: {
		ZoneStatusNormal: true,
		ZoneStatusDanger: true,
	},
	ZoneStatusDanger: {
		ZoneStatusWarning:  true,
		ZoneStatusAerating: true,
	},
	ZoneStatusAerating: {
		ZoneStatusDanger:   true,
		ZoneStatusRestored: true,
	},
	ZoneStatusRestored: {
		ZoneStatusNormal:  true,
		ZoneStatusWarning: true,
		ZoneStatusDanger:  true,
	},
}

// CanZoneTransition reports whether moving from `from` to `to` is legal in
// the farm-zone state machine. Staying in the same state is always allowed.
func CanZoneTransition(from, to ZoneStatus) bool {
	if from == to {
		return true
	}
	return zoneTransitionTable[from][to]
}

// AllowedZoneTransitionsFrom returns the legal next states of `from`.
func AllowedZoneTransitionsFrom(from ZoneStatus) []ZoneStatus {
	if from == "" {
		from = ZoneStatusNormal
	}
	out := make([]ZoneStatus, 0, len(zoneTransitionTable[from]))
	for s := range zoneTransitionTable[from] {
		out = append(out, s)
	}
	return out
}

// StatusAge returns how long the zone has been in its current status.
func (z *FarmZone) StatusAge(now time.Time) time.Duration {
	if now.Before(z.StatusSince) {
		return 0
	}
	return now.Sub(z.StatusSince)
}

// String returns a compact description used in audit entries.
func (z *FarmZone) String() string {
	return fmt.Sprintf("zone %s (%s)", z.ID, z.Name)
}
