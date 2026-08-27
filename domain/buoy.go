package domain

import "time"

// Buoy is a monitoring buoy (监测浮标) moored inside a farm zone. It reports
// water-quality samples on a fixed period; the most recent reading is kept
// on the buoy for fast overview rendering.
type Buoy struct {
	ID     string `json:"id"`
	ZoneID string `json:"zone_id"`
	Name   string `json:"name"`

	// Position is a human-readable position description.
	Position string `json:"position"`

	// Latitude/Longitude optionally pin the buoy on a map.
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`

	Status BuoyStatus `json:"status"`

	// LastReportAt is the timestamp of the latest accepted sample.
	LastReportAt *time.Time `json:"last_report_at,omitempty"`

	// LastDO is the dissolved oxygen of the latest accepted sample.
	LastDO float64 `json:"last_do"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewBuoy builds a buoy. The default status is active.
func NewBuoy(id, zoneID, name, position string, latitude, longitude float64, now time.Time) *Buoy {
	return &Buoy{
		ID:           id,
		ZoneID:       zoneID,
		Name:         name,
		Position:     position,
		Latitude:     latitude,
		Longitude:    longitude,
		Status:       BuoyStatusActive,
		LastReportAt: nil,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// SetStatus validates and applies a new buoy status.
func (b *Buoy) SetStatus(s BuoyStatus, now time.Time) error {
	if !s.Valid() {
		return InvalidInput("invalid buoy status %q", s)
	}
	b.Status = s
	b.UpdatedAt = now
	return nil
}

// IsReporting reports whether the buoy is currently able to report samples.
func (b *Buoy) IsReporting() bool {
	return b.Status == BuoyStatusActive
}

// RecordReport updates the buoy's latest-report bookkeeping after a sample.
func (b *Buoy) RecordReport(do float64, at time.Time) {
	ts := at
	b.LastReportAt = &ts
	b.LastDO = do
	b.UpdatedAt = at
}

// Stale reports whether the buoy has not reported within `window`.
func (b *Buoy) Stale(window time.Duration, now time.Time) bool {
	if b.LastReportAt == nil {
		return true
	}
	return now.Sub(*b.LastReportAt) > window
}

// String returns a compact description used in audit entries.
func (b *Buoy) String() string {
	return "buoy " + b.ID + " (" + b.Name + ")"
}
