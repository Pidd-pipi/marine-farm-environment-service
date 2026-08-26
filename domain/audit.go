package domain

import "time"

// AuditAction is a stable action name recorded in the audit trail.
type AuditAction string

const (
	AuditZoneCreate       AuditAction = "zone.create"
	AuditBuoyCreate       AuditAction = "buoy.create"
	AuditSampleIngest     AuditAction = "sample.ingest"
	AuditWarningCreated   AuditAction = "warning.created"
	AuditWarningVerify    AuditAction = "warning.verify"
	AuditWarningResolve   AuditAction = "warning.resolve"
	AuditAerationStart    AuditAction = "aeration.start"
	AuditAerationStop     AuditAction = "aeration.stop"
	AuditAerationFeedback AuditAction = "aeration.feedback"
	AuditAerationTimeout  AuditAction = "aeration.timeout"
	AuditZoneRestore      AuditAction = "zone.restore"
	AuditFarmLogCreate    AuditAction = "farmlog.create"
	AuditRestoreCheck     AuditAction = "restore.check"
	AuditHTTPRequest      AuditAction = "http.request"
)

// Valid reports whether the action is known.
func (a AuditAction) Valid() bool {
	switch a {
	case AuditZoneCreate, AuditBuoyCreate, AuditSampleIngest, AuditWarningCreated,
		AuditWarningVerify, AuditWarningResolve, AuditAerationStart, AuditAerationStop,
		AuditAerationFeedback, AuditAerationTimeout, AuditZoneRestore, AuditFarmLogCreate,
		AuditRestoreCheck, AuditHTTPRequest:
		return true
	}
	return false
}

// AuditEntry is one operation-audit record. The prompt requires warning
// verification, aeration linkage, restore confirmation and farm-log entry
// to be fully traceable through handler → service → audit store.
type AuditEntry struct {
	ID         string      `json:"id"`
	Action     AuditAction `json:"action"`
	TargetType string      `json:"target_type"`
	TargetID   string      `json:"target_id"`
	Operator   string      `json:"operator"`
	Detail     string      `json:"detail"`
	RequestID  string      `json:"request_id,omitempty"`
	At         time.Time   `json:"at"`
}

// NewAuditEntry builds an audit entry.
func NewAuditEntry(id string, action AuditAction, targetType, targetID, operator, detail string, at time.Time) *AuditEntry {
	return &AuditEntry{
		ID:         id,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Operator:   operator,
		Detail:     detail,
		At:         at,
	}
}
