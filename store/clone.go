package store

import (
	"time"

	"example.com/marine-farm-environment-service/domain"
)

// Deep-copy helpers: repository read paths must never hand out slices or
// pointers that alias the store's internal state. A caller mutating a
// returned entity (for example appending to WaterSample.Violations or
// dereferencing a *time.Time) must never affect the repository.

func cloneTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	cp := *t
	return &cp
}

func cloneStrings(in []string) []string {
	if in == nil {
		return nil
	}
	return append([]string(nil), in...)
}

func cloneFarmZone(z domain.FarmZone) domain.FarmZone {
	z.RestoreEligibleAt = cloneTime(z.RestoreEligibleAt)
	return z
}

func cloneBuoy(b domain.Buoy) domain.Buoy {
	b.LastReportAt = cloneTime(b.LastReportAt)
	return b
}

func cloneWaterSample(s domain.WaterSample) domain.WaterSample {
	s.Violations = cloneStrings(s.Violations)
	return s
}

func cloneWarningRecord(w domain.WarningRecord) domain.WarningRecord {
	w.VerifiedAt = cloneTime(w.VerifiedAt)
	w.ResolvedAt = cloneTime(w.ResolvedAt)
	return w
}

func cloneAerationLog(a domain.AerationLog) domain.AerationLog {
	a.FeedbackAt = cloneTime(a.FeedbackAt)
	return a
}

func cloneFarmLog(f domain.FarmLog) domain.FarmLog {
	return f
}

func cloneAuditEntry(a domain.AuditEntry) domain.AuditEntry {
	return a
}
