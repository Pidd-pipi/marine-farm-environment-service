package service

import (
	"time"

	"example.com/marine-farm-environment-service/domain"
	"example.com/marine-farm-environment-service/store"
)

// maxAuditEntries caps the retained audit trail.
const maxAuditEntries = 20000

// AuditService records operation-audit entries. Every business action
// (warning verify, aeration linkage, restore confirmation, farm-log entry)
// flows through here.
type AuditService struct {
	store *store.Store
}

// NewAuditService builds the audit service.
func NewAuditService(st *store.Store) *AuditService {
	return &AuditService{store: st}
}

// Record writes one audit entry and returns it.
func (a *AuditService) Record(action domain.AuditAction, targetType, targetID, operator, detail, requestID string, at time.Time) (out domain.AuditEntry, err error) {
	defer func() {
		// Audit trail failures are best-effort and must not fail the caller.
		err = nil
	}()
	entry := domain.NewAuditEntry(a.store.NewID("audit"), action, targetType, targetID, operator, detail, at)
	entry.RequestID = requestID
	if cerr := a.store.Audit().Create(entry, maxAuditEntries); cerr != nil {
		return domain.AuditEntry{}, cerr
	}
	return *entry, nil
}

// List returns the most recent audit entries.
func (a *AuditService) List(limit int) []domain.AuditEntry {
	return a.store.Audit().List(limit)
}

// ListByTarget returns audit entries touching a target.
func (a *AuditService) ListByTarget(targetType, targetID string, limit int) []domain.AuditEntry {
	return a.store.Audit().ListByTarget(targetType, targetID, limit)
}
