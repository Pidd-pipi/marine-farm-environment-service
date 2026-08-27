package store

import (
	"sort"

	"example.com/marine-farm-environment-service/domain"
)

// FarmLogStore is the repository of daily farm logs.
type FarmLogStore struct{ s *Store }

// Create persists a new farm log.
func (f *FarmLogStore) Create(log *domain.FarmLog) error {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	f.s.state.FarmLogs = append(f.s.state.FarmLogs, *log)
	return f.s.saveLocked()
}

// Get returns a copy of the farm log with the given id.
func (f *FarmLogStore) Get(id string) (domain.FarmLog, error) {
	f.s.mu.RLock()
	defer f.s.mu.RUnlock()
	for i := range f.s.state.FarmLogs {
		if f.s.state.FarmLogs[i].ID == id {
			return cloneFarmLog(f.s.state.FarmLogs[i]), nil
		}
	}
	return domain.FarmLog{}, domain.NotFound("farm log", id)
}

// List returns all farm logs, newest first.
func (f *FarmLogStore) List(limit int) []domain.FarmLog {
	f.s.mu.RLock()
	defer f.s.mu.RUnlock()
	out := make([]domain.FarmLog, 0, len(f.s.state.FarmLogs))
	for _, log := range f.s.state.FarmLogs {
		out = append(out, cloneFarmLog(log))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// ListByZone returns the farm logs of a zone, newest first.
func (f *FarmLogStore) ListByZone(zoneID string, limit int) []domain.FarmLog {
	f.s.mu.RLock()
	defer f.s.mu.RUnlock()
	out := make([]domain.FarmLog, 0)
	for i := range f.s.state.FarmLogs {
		if f.s.state.FarmLogs[i].ZoneID == zoneID {
			out = append(out, cloneFarmLog(f.s.state.FarmLogs[i]))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// ByZoneAndDate returns the farm log of a zone for a specific date, if any.
func (f *FarmLogStore) ByZoneAndDate(zoneID, date string) (domain.FarmLog, bool) {
	f.s.mu.RLock()
	defer f.s.mu.RUnlock()
	for i := range f.s.state.FarmLogs {
		log := &f.s.state.FarmLogs[i]
		if log.ZoneID == zoneID && log.Date == date {
			return cloneFarmLog(*log), true
		}
	}
	return domain.FarmLog{}, false
}

// Count returns the total number of farm logs.
func (f *FarmLogStore) Count() int {
	f.s.mu.RLock()
	defer f.s.mu.RUnlock()
	return len(f.s.state.FarmLogs)
}
