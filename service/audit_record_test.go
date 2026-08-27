package service

import (
	"testing"
	"time"

	"example.com/marine-farm-environment-service/config"
	"example.com/marine-farm-environment-service/domain"
	"example.com/marine-farm-environment-service/store"
)

// TestRecordErrorPreserved pins the error contract: a persistence failure
// while writing the audit trail must be surfaced, not silently swallowed.
func TestRecordErrorPreserved(t *testing.T) {
	cfg := config.Default()
	cfg.DataFile = "/dev/null/marine/x.json" // unwritable: /dev/null is a file
	st := store.NewStore(cfg.DataFile)
	svc := New(cfg, st)

	_, err := svc.Audit.Record(domain.AuditZoneCreate, "zone", "z1", "op", "d", "req", time.Now().UTC())
	if err == nil {
		t.Fatalf("audit persistence error was swallowed")
	}
}
