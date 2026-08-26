package domain

import "testing"

func TestEnumValidity(t *testing.T) {
	for _, s := range AllZoneStatuses() {
		if !s.Valid() {
			t.Fatalf("zone status %s must be valid", s)
		}
		if s.Label() == "" {
			t.Fatalf("zone status %s needs a label", s)
		}
	}
	if ZoneStatus("bogus").Valid() {
		t.Fatal("bogus zone status must be invalid")
	}
	for _, w := range AllWarningTypes() {
		if !w.Valid() || w.Label() == "" {
			t.Fatalf("warning type %s invalid or unlabelled", w)
		}
	}
	for _, s := range []WarningStatus{WarningStatusPending, WarningStatusConfirmed, WarningStatusResolved} {
		if !s.Valid() || s.Label() == "" {
			t.Fatalf("warning status %s invalid or unlabelled", s)
		}
	}
	for _, s := range AllAeratorStatuses() {
		if !s.Valid() || s.Label() == "" {
			t.Fatalf("aerator status %s invalid or unlabelled", s)
		}
	}
	for _, s := range []BuoyStatus{BuoyStatusActive, BuoyStatusOffline, BuoyStatusMaintenance} {
		if !s.Valid() {
			t.Fatalf("buoy status %s invalid", s)
		}
	}
	for _, f := range []FeedbackStatus{FeedbackNone, FeedbackAcknowledged, FeedbackStarted, FeedbackStopped, FeedbackFault, FeedbackTimeout} {
		if !f.Valid() {
			t.Fatalf("feedback %s invalid", f)
		}
	}
}

func TestAuditActionValidity(t *testing.T) {
	actions := []AuditAction{
		AuditZoneCreate, AuditBuoyCreate, AuditSampleIngest, AuditWarningCreated,
		AuditWarningVerify, AuditWarningResolve, AuditAerationStart, AuditAerationStop,
		AuditAerationFeedback, AuditAerationTimeout, AuditZoneRestore, AuditFarmLogCreate,
		AuditRestoreCheck, AuditHTTPRequest,
	}
	for _, a := range actions {
		if !a.Valid() {
			t.Fatalf("audit action %s must be valid", a)
		}
	}
}
