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

// cloneSeq returns a fresh map so a caller mutating the returned seq map can
// never race with or alias the store's internal seq table.
func cloneSeq(in map[string]uint64) map[string]uint64 {
	if in == nil {
		return map[string]uint64{}
	}
	out := make(map[string]uint64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// cloneFarmZoneSlice returns a fully isolated copy of a zone slice; every
// element is run through cloneFarmZone so the returned slice and all of its
// nested pointers are independent of the source.
func cloneFarmZoneSlice(in []domain.FarmZone) []domain.FarmZone {
	if in == nil {
		return nil
	}
	out := make([]domain.FarmZone, len(in))
	for i := range in {
		out[i] = cloneFarmZone(in[i])
	}
	return out
}

func cloneBuoySlice(in []domain.Buoy) []domain.Buoy {
	if in == nil {
		return nil
	}
	out := make([]domain.Buoy, len(in))
	for i := range in {
		out[i] = cloneBuoy(in[i])
	}
	return out
}

func cloneWaterSampleSlice(in []domain.WaterSample) []domain.WaterSample {
	if in == nil {
		return nil
	}
	out := make([]domain.WaterSample, len(in))
	for i := range in {
		out[i] = cloneWaterSample(in[i])
	}
	return out
}

func cloneWarningRecordSlice(in []domain.WarningRecord) []domain.WarningRecord {
	if in == nil {
		return nil
	}
	out := make([]domain.WarningRecord, len(in))
	for i := range in {
		out[i] = cloneWarningRecord(in[i])
	}
	return out
}

func cloneAerationLogSlice(in []domain.AerationLog) []domain.AerationLog {
	if in == nil {
		return nil
	}
	out := make([]domain.AerationLog, len(in))
	for i := range in {
		out[i] = cloneAerationLog(in[i])
	}
	return out
}

func cloneFarmLogSlice(in []domain.FarmLog) []domain.FarmLog {
	if in == nil {
		return nil
	}
	out := make([]domain.FarmLog, len(in))
	for i := range in {
		out[i] = cloneFarmLog(in[i])
	}
	return out
}

func cloneAuditEntrySlice(in []domain.AuditEntry) []domain.AuditEntry {
	if in == nil {
		return nil
	}
	out := make([]domain.AuditEntry, len(in))
	for i := range in {
		out[i] = cloneAuditEntry(in[i])
	}
	return out
}
