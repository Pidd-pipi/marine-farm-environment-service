package service

import (
	"log/slog"
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
//
// Audit trail writes are best-effort: a failure here must never roll back
// the business action that triggered it. But the failure is surfaced — both
// logged, so it shows up in operations, and returned, so callers that *do*
// care (batch ingestion, durability checks) can see it instead of being
// told everything succeeded.
func (a *AuditService) Record(action domain.AuditAction, targetType, targetID, operator, detail, requestID string, at time.Time) (domain.AuditEntry, error) {
	entry := domain.NewAuditEntry(a.store.NewID("audit"), action, targetType, targetID, operator, detail, at)
	entry.RequestID = requestID
	if cerr := a.store.Audit().Create(entry, maxAuditEntries); cerr != nil {
		slog.Error("audit: record failed",
			"action", string(action), "target_type", targetType, "target_id", targetID,
			"operator", operator, "request_id", requestID, "error", cerr)
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
