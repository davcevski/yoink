package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// A pre-existing, world-readable log (as launchd or a v0.0.1 install would leave
// it) must be tightened to owner-only without losing its contents.
func TestEnsureLogFileTightensExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "yoinkd.log")
	if err := os.WriteFile(path, []byte("prior log\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ensureLogFile(path); err != nil {
		t.Fatalf("ensureLogFile: %v", err)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Errorf("perm = %v, want 0600", got)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "prior log\n" {
		t.Errorf("content = %q, want preserved (append, not truncate)", b)
	}
}

// A fresh install creates the log already locked down.
func TestEnsureLogFileCreatesOwnerOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "yoinkd.log")

	if err := ensureLogFile(path); err != nil {
		t.Fatalf("ensureLogFile: %v", err)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Errorf("perm = %v, want 0600", got)
	}
}
