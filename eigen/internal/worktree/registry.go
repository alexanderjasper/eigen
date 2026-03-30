package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Entry represents a discovered worktree.
type Entry struct {
	Name      string    `json:"name"`
	Branch    string    `json:"branch"`
	Path      string    `json:"path"` // absolute
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// FindGitRoot finds the git repository root from startDir by shelling out to git.
// Works from both the main repo and any worktree sub-directory.
func FindGitRoot(startDir string) (string, error) {
	cmd := exec.Command("git", "-C", startDir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("finding git root from %s: %w", startDir, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// WorktreesDir returns the canonical path where eigen/Claude worktrees are created.
func WorktreesDir(gitRoot string) string {
	return filepath.Join(gitRoot, ".claude", "worktrees")
}

// CurrentBranch returns the current branch name for the given directory.
func CurrentBranch(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git branch in %s: %w", dir, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ScanWorktreesDir scans .claude/worktrees/ and returns an Entry for each
// subdirectory that is a valid git worktree (CurrentBranch succeeds).
func ScanWorktreesDir(gitRoot string) ([]Entry, error) {
	dir := WorktreesDir(gitRoot)
	infos, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scanning worktrees dir: %w", err)
	}

	var discovered []Entry
	for _, info := range infos {
		if !info.IsDir() {
			continue
		}
		name := info.Name()
		wtPath := filepath.Join(dir, name)
		branch, err := CurrentBranch(wtPath)
		if err != nil {
			continue // not a git repo / not accessible — skip
		}
		discovered = append(discovered, Entry{
			Name:   name,
			Branch: branch,
			Path:   wtPath,
		})
	}
	return discovered, nil
}
