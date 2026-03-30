package worktree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanWorktreesDir_AbsentDir(t *testing.T) {
	dir := t.TempDir()
	entries, err := ScanWorktreesDir(dir)
	if err != nil {
		t.Fatalf("expected no error for absent worktrees dir, got: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty entries, got %d", len(entries))
	}
}

func TestScanWorktreesDir_SkipsNonGitDirs(t *testing.T) {
	gitRoot := t.TempDir()
	wtDir := filepath.Join(gitRoot, ".claude", "worktrees")
	if err := os.MkdirAll(wtDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create a non-git subdirectory.
	if err := os.Mkdir(filepath.Join(wtDir, "not-a-repo"), 0755); err != nil {
		t.Fatal(err)
	}
	entries, err := ScanWorktreesDir(gitRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The non-git dir should be skipped (CurrentBranch fails).
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}
