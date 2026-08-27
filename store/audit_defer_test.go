package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"example.com/marine-farm-environment-service/domain"
)

// TestAuditCapKeepsNewest pins the cap contract: the audit trail must keep
// the most recent entries when it grows past the configured maximum.
func TestAuditCapKeepsNewest(t *testing.T) {
	st := NewMemoryStore()
	now := time.Now().UTC()
	const cap = 3
	var newest string
	for i := 0; i < cap+2; i++ {
		e := domain.NewAuditEntry(st.NewID("audit"), domain.AuditHTTPRequest, "http", "/x", "op", "d", now.Add(time.Duration(i)*time.Second))
		if err := st.Audit().Create(e, cap); err != nil {
			t.Fatalf("create audit: %v", err)
		}
		newest = e.ID
	}
	if st.Audit().Count() != cap {
		t.Fatalf("audit count = %d, want %d", st.Audit().Count(), cap)
	}
	all := st.Audit().List(0)
	for _, e := range all {
		if e.ID == newest {
			return
		}
	}
	t.Fatalf("newest audit entry was dropped (oldest retained instead)")
}

// TestBackupCorruptFileDoesNotOverwrite pins the backup contract: an
// existing backup must never be silently overwritten.
func TestBackupCorruptFileDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")
	if err := os.WriteFile(path+".bak", []byte("old"), 0o644); err != nil {
		t.Fatalf("write existing backup: %v", err)
	}
	if err := backupCorruptFile(path, []byte("new")); err != nil {
		t.Fatalf("backup to numbered slot failed: %v", err)
	}
	got, err := os.ReadFile(path + ".bak")
	if err != nil || string(got) != "old" {
		t.Fatalf("original .bak was overwritten: got %q err=%v", got, err)
	}
	numbered, err := os.ReadFile(path + ".bak.1")
	if err != nil || string(numbered) != "new" {
		t.Fatalf("corrupt snapshot not written to .bak.1: got %q err=%v", numbered, err)
	}
}

// TestWriteFileAtomicCleansUpOnRenameError pins the release contract: a
// failed atomic rename must not leave the temporary file behind.
func TestWriteFileAtomicCleansUpOnRenameError(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "dest")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := writeFileAtomic(target, []byte("payload"), 0o644); err == nil {
		t.Fatalf("rename onto a directory should fail")
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "dest.tmp-*"))
	if len(matches) != 0 {
		t.Fatalf("temporary file leaked after failed rename: %v", matches)
	}
}
