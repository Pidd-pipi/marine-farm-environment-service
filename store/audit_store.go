package store

import (
	"sort"

	"example.com/marine-farm-environment-service/domain"
)

// AuditStore is the repository of operation-audit entries.
type AuditStore struct{ s *Store }

// Create persists a new audit entry. The audit trail is capped to
// maxEntries (oldest entries are dropped) to bound memory growth.
func (a *AuditStore) Create(entry *domain.AuditEntry, maxEntries int) error {
	a.s.mu.Lock()
	defer a.s.mu.Unlock()
	a.s.state.Audit = append(a.s.state.Audit, *entry)
	if maxEntries > 0 && len(a.s.state.Audit) > maxEntries {
		a.s.state.Audit = a.s.state.Audit[:maxEntries]
	}
	return a.s.saveLocked()
}

// List returns the most recent `limit` audit entries, newest first.
func (a *AuditStore) List(limit int) []domain.AuditEntry {
	a.s.mu.RLock()
	defer a.s.mu.RUnlock()
	out := make([]domain.AuditEntry, 0, len(a.s.state.Audit))
	for _, entry := range a.s.state.Audit {
		out = append(out, cloneAuditEntry(entry))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// ListByTarget returns audit entries touching a target, newest first.
func (a *AuditStore) ListByTarget(targetType, targetID string, limit int) []domain.AuditEntry {
	a.s.mu.RLock()
	defer a.s.mu.RUnlock()
	out := make([]domain.AuditEntry, 0)
	for i := range a.s.state.Audit {
		e := cloneAuditEntry(a.s.state.Audit[i])
		if e.TargetType == targetType && e.TargetID == targetID {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// Count returns the total number of audit entries.
func (a *AuditStore) Count() int {
	a.s.mu.RLock()
	defer a.s.mu.RUnlock()
	return len(a.s.state.Audit)
}
