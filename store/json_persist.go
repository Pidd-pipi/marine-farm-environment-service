package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// encodeState serialises a state snapshot to indented JSON.
func encodeState(s *State) ([]byte, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("store: marshal state: %w", err)
	}
	return append(data, '\n'), nil
}

// decodeState parses a state snapshot from JSON.
func decodeState(data []byte, s *State) error {
	if err := json.Unmarshal(data, s); err != nil {
		return err
	}
	if s.Version != stateVersion {
		// Future migrations can upgrade older snapshots here.
		s.Version = stateVersion
	}
	if s.Seq == nil {
		s.Seq = map[string]uint64{}
	}
	return nil
}

// writeFileAtomic writes data to path using the standard crash-safe
// sequence: create a unique temporary file in the target directory, write
// the payload, fsync it to stable storage, rename it over the destination
// and fsync the directory so the rename itself is durable.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("store: mkdir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("store: create tmp: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}

	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("store: write tmp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("store: fsync tmp: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		cleanup()
		return fmt.Errorf("store: chmod tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("store: close tmp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("store: rename: %w", err)
	}

	// fsync the parent directory so the rename survives a power loss.
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

// backupCorruptFile copies the unreadable snapshot to path.bak so it can be
// inspected later. It never overwrites an existing backup from an earlier
// incident.
func backupCorruptFile(path string, data []byte) error {
	backup := path + ".bak"
	if _, err := os.Stat(backup); err == nil {
		return fmt.Errorf("backup %s already exists", backup)
	}
	return writeFileAtomic(backup, data, 0o644)
}

// SaveSnapshotTo is a test helper that serialises a store's state to a
// path without changing the configured persistence file.
func (s *Store) SaveSnapshotTo(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.file
	s.file = path
	err := s.saveLocked()
	s.file = old
	return err
}
