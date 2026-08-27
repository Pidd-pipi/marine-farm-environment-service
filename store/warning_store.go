package store

import (
	"sort"

	"example.com/marine-farm-environment-service/domain"
)

// WarningFilter narrows the warning list query.
type WarningFilter struct {
	ZoneID string
	Status string // "", pending, confirmed, resolved
	Type   string
	Limit  int
}

// WarningStore is the repository of warning records.
type WarningStore struct{ s *Store }

// Create persists a new warning record.
func (w *WarningStore) Create(record *domain.WarningRecord) error {
	w.s.mu.Lock()
	defer w.s.mu.Unlock()
	w.s.state.Warnings = append(w.s.state.Warnings, *record)
	return w.s.saveLocked()
}

// Get returns a copy of the warning with the given id.
func (w *WarningStore) Get(id string) (domain.WarningRecord, error) {
	w.s.mu.RLock()
	defer w.s.mu.RUnlock()
	for i := range w.s.state.Warnings {
		if w.s.state.Warnings[i].ID == id {
			return cloneWarningRecord(w.s.state.Warnings[i]), nil
		}
	}
	return domain.WarningRecord{}, domain.NotFound("warning record", id)
}

// List returns warnings matching the filter, newest first.
func (w *WarningStore) List(f WarningFilter) []domain.WarningRecord {
	w.s.mu.RLock()
	defer w.s.mu.RUnlock()
	out := w.s.state.Warnings[:0]
	for i := range w.s.state.Warnings {
		rec := w.s.state.Warnings[i]
		if f.ZoneID != "" && rec.ZoneID != f.ZoneID {
			continue
		}
		if f.Status != "" && string(rec.Status) != f.Status {
			continue
		}
		if f.Type != "" && string(rec.Type) != f.Type {
			continue
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ReportedAt.After(out[j].ReportedAt) })
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out
}

// ListByZone returns the warnings of a zone, newest first.
func (w *WarningStore) ListByZone(zoneID string, limit int) []domain.WarningRecord {
	return w.List(WarningFilter{ZoneID: zoneID, Limit: limit})
}

// Update persists a modified warning record.
func (w *WarningStore) Update(record *domain.WarningRecord) error {
	w.s.mu.Lock()
	defer w.s.mu.Unlock()
	for i := range w.s.state.Warnings {
		if w.s.state.Warnings[i].ID == record.ID {
			w.s.state.Warnings[i] = *record
			return w.s.saveLocked()
		}
	}
	return domain.NotFound("warning record", record.ID)
}

// CountByStatus counts warnings of the given status.
func (w *WarningStore) CountByStatus(status domain.WarningStatus) int {
	w.s.mu.RLock()
	defer w.s.mu.RUnlock()
	n := 0
	for i := range w.s.state.Warnings {
		if w.s.state.Warnings[i].Status == status {
			n++
		}
	}
	return n
}

// CountByZoneAndStatus counts open (pending/confirmed) warnings of a zone.
func (w *WarningStore) CountOpenByZone(zoneID string) int {
	w.s.mu.RLock()
	defer w.s.mu.RUnlock()
	n := 0
	for i := range w.s.state.Warnings {
		rec := &w.s.state.Warnings[i]
		if rec.ZoneID == zoneID && rec.IsOpen() {
			n++
		}
	}
	return n
}

// CountOpen returns the total number of open warnings.
func (w *WarningStore) CountOpen() int {
	w.s.mu.RLock()
	defer w.s.mu.RUnlock()
	n := 0
	for i := range w.s.state.Warnings {
		if w.s.state.Warnings[i].IsOpen() {
			n++
		}
	}
	return n
}
