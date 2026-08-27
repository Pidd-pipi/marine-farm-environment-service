package store

import (
	"sort"

	"example.com/marine-farm-environment-service/domain"
)

// AerationStore is the repository of aerator command logs.
type AerationStore struct{ s *Store }

// Create persists a new aeration log.
func (a *AerationStore) Create(log *domain.AerationLog) error {
	a.s.mu.Lock()
	defer a.s.mu.Unlock()
	a.s.state.Aeration = append(a.s.state.Aeration, *log)
	return a.s.saveLocked()
}

// Get returns a copy of the aeration log with the given id.
func (a *AerationStore) Get(id string) (domain.AerationLog, error) {
	a.s.mu.RLock()
	defer a.s.mu.RUnlock()
	for i := range a.s.state.Aeration {
		if a.s.state.Aeration[i].ID == id {
			return cloneAerationLog(a.s.state.Aeration[i]), nil
		}
	}
	return domain.AerationLog{}, domain.NotFound("aeration log", id)
}

// List returns all aeration logs, newest first.
func (a *AerationStore) List(limit int) []domain.AerationLog {
	a.s.mu.RLock()
	defer a.s.mu.RUnlock()
	out := make([]domain.AerationLog, 0, len(a.s.state.Aeration))
	for _, log := range a.s.state.Aeration {
		out = append(out, cloneAerationLog(log))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CommandTime.After(out[j].CommandTime) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// ListByZone returns the aeration logs of a zone, newest first.
func (a *AerationStore) ListByZone(zoneID string, limit int) []domain.AerationLog {
	a.s.mu.RLock()
	defer a.s.mu.RUnlock()
	out := make([]domain.AerationLog, 0)
	for i := range a.s.state.Aeration {
		if a.s.state.Aeration[i].ZoneID == zoneID {
			out = append(out, cloneAerationLog(a.s.state.Aeration[i]))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CommandTime.After(out[j].CommandTime) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// LatestByZone returns the most recent aeration log of a zone, if any.
func (a *AerationStore) LatestByZone(zoneID string) (domain.AerationLog, bool) {
	a.s.mu.RLock()
	defer a.s.mu.RUnlock()
	var latest *domain.AerationLog
	for i := range a.s.state.Aeration {
		log := &a.s.state.Aeration[i]
		if log.ZoneID != zoneID {
			continue
		}
		if latest == nil || log.CommandTime.After(latest.CommandTime) {
			latest = log
		}
	}
	if latest == nil {
		return domain.AerationLog{}, false
	}
	return cloneAerationLog(*latest), true
}

// Update persists a modified aeration log.
func (a *AerationStore) Update(log *domain.AerationLog) error {
	a.s.mu.Lock()
	defer a.s.mu.Unlock()
	for i := range a.s.state.Aeration {
		if a.s.state.Aeration[i].ID == log.ID {
			a.s.state.Aeration[i] = *log
			return a.s.saveLocked()
		}
	}
	return domain.NotFound("aeration log", log.ID)
}

// ListPending returns every command still waiting for terminal feedback
// (none or only acknowledged); these are eligible for timeout handling.
func (a *AerationStore) ListPending() []domain.AerationLog {
	a.s.mu.RLock()
	defer a.s.mu.RUnlock()
	out := make([]domain.AerationLog, 0)
	for i := range a.s.state.Aeration {
		if !a.s.state.Aeration[i].HasTerminalFeedback() {
			out = append(out, cloneAerationLog(a.s.state.Aeration[i]))
		}
	}
	return out
}

// CountActive returns the number of aerators currently energised
// (starting or running).
func (a *AerationStore) CountActive() int {
	a.s.mu.RLock()
	defer a.s.mu.RUnlock()
	n := 0
	for i := range a.s.state.Aeration {
		if a.s.state.Aeration[i].IsActive() {
			n++
		}
	}
	return n
}

// CountByStatus returns the number of logs currently in the given status.
func (a *AerationStore) CountByStatus(status domain.AeratorStatus) int {
	a.s.mu.RLock()
	defer a.s.mu.RUnlock()
	n := 0
	for i := range a.s.state.Aeration {
		if a.s.state.Aeration[i].Status == status {
			n++
		}
	}
	return n
}
