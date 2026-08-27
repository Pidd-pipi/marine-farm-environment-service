// Package store implements the persistence layer: an in-memory repository
// with atomic JSON-file snapshots. It has no external service dependency —
// every entity repository lives here and is reachable through the shared
// Store aggregator.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"example.com/marine-farm-environment-service/domain"
)

// stateVersion is bumped whenever the persisted snapshot layout changes.
const stateVersion = 1

// State is the full persisted snapshot of the service.
type State struct {
	Version int               `json:"version"`
	Seq     map[string]uint64 `json:"seq"`

	Zones    []domain.FarmZone      `json:"zones"`
	Buoys    []domain.Buoy          `json:"buoys"`
	Samples  []domain.WaterSample   `json:"samples"`
	Warnings []domain.WarningRecord `json:"warnings"`
	Aeration []domain.AerationLog   `json:"aeration"`
	FarmLogs []domain.FarmLog       `json:"farm_logs"`
	Audit    []domain.AuditEntry    `json:"audit"`
}

// Store is the thread-safe in-memory repository. When `file` is non-empty
// it loads any existing snapshot on startup and flushes every mutation to
// it, giving durable persistence without any external service.
type Store struct {
	mu    sync.RWMutex
	file  string
	state State

	zones    *ZoneStore
	buoys    *BuoyStore
	samples  *SampleStore
	warnings *WarningStore
	aeration *AerationStore
	farmlogs *FarmLogStore
	audit    *AuditStore
}

// NewStore creates a store. When file is non-empty the store loads any
// existing snapshot from it and flushes every mutation to it.
func NewStore(file string) *Store {
	s := &Store{file: file}
	s.state.Version = stateVersion
	s.state.Seq = map[string]uint64{}
	s.zones = &ZoneStore{s: s}
	s.buoys = &BuoyStore{s: s}
	s.samples = &SampleStore{s: s}
	s.warnings = &WarningStore{s: s}
	s.aeration = &AerationStore{s: s}
	s.farmlogs = &FarmLogStore{s: s}
	s.audit = &AuditStore{s: s}
	return s
}

// NewMemoryStore creates a store without any file persistence (tests,
// ephemeral runs).
func NewMemoryStore() *Store { return NewStore("") }

// Zones returns the farm-zone repository.
func (s *Store) Zones() *ZoneStore { return s.zones }

// Buoys returns the buoy repository.
func (s *Store) Buoys() *BuoyStore { return s.buoys }

// Samples returns the water-sample repository.
func (s *Store) Samples() *SampleStore { return s.samples }

// Warnings returns the warning repository.
func (s *Store) Warnings() *WarningStore { return s.warnings }

// Aeration returns the aeration-log repository.
func (s *Store) Aeration() *AerationStore { return s.aeration }

// FarmLogs returns the farm-log repository.
func (s *Store) FarmLogs() *FarmLogStore { return s.farmlogs }

// Audit returns the audit repository.
func (s *Store) Audit() *AuditStore { return s.audit }

// File returns the configured persistence file.
func (s *Store) File() string { return s.file }

// Load reads the JSON snapshot from the persistence file into memory. It
// is a no-op when the file does not exist. A corrupt snapshot is backed up
// to <file>.bak and the service degrades to an empty store with a warning.
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file == "" {
		return nil
	}
	data, err := os.ReadFile(s.file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("store: read %s: %w", s.file, err)
	}
	if err := decodeState(data, &s.state); err != nil {
		if backupErr := backupCorruptFile(s.file, data); backupErr != nil {
			slog.Error("store: corrupt snapshot backup failed", "file", s.file, "error", backupErr)
		} else {
			slog.Warn("store: corrupt snapshot backed up, starting with empty state",
				"file", s.file, "backup", s.file+".bak", "error", err)
		}
		s.state = State{Version: stateVersion, Seq: map[string]uint64{}}
		return nil
	}
	if s.state.Seq == nil {
		s.state.Seq = map[string]uint64{}
	}
	return nil
}

// Save flushes the in-memory state to the persistence file atomically
// (unique temp file + fsync + rename). It is a no-op when persistence is
// disabled. The write lock serialises concurrent Save calls so two writers
// can never interleave on the same temporary file.
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

// saveLocked writes the snapshot; the caller must hold the write lock.
func (s *Store) saveLocked() error {
	if s.file == "" {
		return nil
	}
	data, err := encodeState(&s.state)
	if err != nil {
		return err
	}
	if err := writeFileAtomic(s.file, data, 0o644); err != nil {
		return err
	}
	return nil
}

// nextID generates the next monotonic id for a prefix (e.g. "zone_7").
func (s *Store) nextID(prefix string) string {
	s.state.Seq[prefix]++
	return fmt.Sprintf("%s_%d", prefix, s.state.Seq[prefix])
}

// NewID returns the next monotonic id for a prefix (thread-safe).
func (s *Store) NewID(prefix string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.nextID(prefix)
}

// randomToken returns a short random hex token used for request ids.
func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// NewToken returns a random hex token used for request correlation.
func (s *Store) NewToken(n int) string { return randomToken(n) }

// Count returns the total number of persisted entities of each kind; used
// by the health endpoint and overview aggregation. It takes the read lock so
// it never observes a slice mid-append/mid-realloc from a concurrent writer.
func (s *Store) Count() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]int{
		"zones":    len(s.state.Zones),
		"buoys":    len(s.state.Buoys),
		"samples":  len(s.state.Samples),
		"warnings": len(s.state.Warnings),
		"aeration": len(s.state.Aeration),
		"farmlogs": len(s.state.FarmLogs),
		"audit":    len(s.state.Audit),
	}
}

// Snapshot returns a consistent, fully isolated copy of the whole state for
// read paths that need a multi-entity view (overview aggregation, bootstrap).
//
// It takes the read lock for the duration of the copy, so it never races with
// a concurrent writer that is appending to or reallocating one of the slices
// (the historical symptom was the overview sample count flickering between
// reads taken in the same second). Every slice is deep-copied element by
// element via the clone helpers, so mutating a returned entity or one of its
// nested slices (e.g. WaterSample.Violations) can never leak back into the
// store — callers receive values they fully own.
func (s *Store) Snapshot() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := State{
		Version: s.state.Version,
		Seq:     cloneSeq(s.state.Seq),
	}
	out.Zones = cloneFarmZoneSlice(s.state.Zones)
	out.Buoys = cloneBuoySlice(s.state.Buoys)
	out.Samples = cloneWaterSampleSlice(s.state.Samples)
	out.Warnings = cloneWarningRecordSlice(s.state.Warnings)
	out.Aeration = cloneAerationLogSlice(s.state.Aeration)
	out.FarmLogs = cloneFarmLogSlice(s.state.FarmLogs)
	out.Audit = cloneAuditEntrySlice(s.state.Audit)
	return out
}

// IsEmpty reports whether no business entity has been persisted yet; used
// by the bootstrap seeder.
func (s *Store) IsEmpty() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.state.Zones) == 0 && len(s.state.Buoys) == 0 && len(s.state.Samples) == 0
}

// Reset clears all data (used by bootstrap for idempotent reseeding).
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = State{Version: stateVersion, Seq: map[string]uint64{}}
}
