package worktree

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Entry represents a single registered worktree.
type Entry struct {
	Name      string    `json:"name"`
	Branch    string    `json:"branch"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
}

// Registry holds the list of registered worktrees.
type Registry struct {
	Entries []Entry `json:"entries"`
}

// FindGitRoot finds the git repository root from startDir by shelling out to git.
// It works from both the main repo and any worktree sub-directory.
func FindGitRoot(startDir string) (string, error) {
	cmd := exec.Command("git", "-C", startDir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("finding git root from %s: %w", startDir, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// RegistryPath returns the path to the worktrees.json registry file.
func RegistryPath(gitRoot string) string {
	return filepath.Join(gitRoot, ".eigen", "worktrees.json")
}

// ReadRegistry reads and decodes the registry from disk.
// If the file does not exist, it returns an empty Registry without error.
func ReadRegistry(gitRoot string) (Registry, error) {
	path := RegistryPath(gitRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Registry{}, nil
		}
		return Registry{}, fmt.Errorf("reading registry: %w", err)
	}
	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return Registry{}, fmt.Errorf("parsing registry: %w", err)
	}
	return reg, nil
}

// WriteRegistry atomically writes the registry to disk.
// It creates the .eigen/ directory if it does not exist.
func WriteRegistry(gitRoot string, reg Registry) error {
	eigenDir := filepath.Join(gitRoot, ".eigen")
	if err := os.MkdirAll(eigenDir, 0755); err != nil {
		return fmt.Errorf("creating .eigen dir: %w", err)
	}

	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling registry: %w", err)
	}

	regPath := RegistryPath(gitRoot)
	tmp := regPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("writing registry temp file: %w", err)
	}
	if err := os.Rename(tmp, regPath); err != nil {
		return fmt.Errorf("renaming registry temp file: %w", err)
	}
	return nil
}

// AddEntry reads the registry, appends the entry, and writes it back.
func AddEntry(gitRoot string, e Entry) error {
	reg, err := ReadRegistry(gitRoot)
	if err != nil {
		return err
	}
	reg.Entries = append(reg.Entries, e)
	return WriteRegistry(gitRoot, reg)
}

// RemoveEntry reads the registry, removes the entry with the given name, and writes it back.
// Returns true if an entry was removed, false if not found.
func RemoveEntry(gitRoot string, name string) (bool, error) {
	reg, err := ReadRegistry(gitRoot)
	if err != nil {
		return false, err
	}
	var kept []Entry
	found := false
	for _, e := range reg.Entries {
		if e.Name == name {
			found = true
			continue
		}
		kept = append(kept, e)
	}
	if !found {
		return false, nil
	}
	reg.Entries = kept
	return true, WriteRegistry(gitRoot, reg)
}

// GitShortHash returns the short git commit hash for the given directory.
func GitShortHash(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git short hash in %s: %w", dir, err)
	}
	return strings.TrimSpace(string(out)), nil
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

// WorktreesDir returns the canonical path where eigen/Claude worktrees are created.
func WorktreesDir(gitRoot string) string {
	return filepath.Join(gitRoot, ".claude", "worktrees")
}

// ScanWorktreesDir scans .claude/worktrees/ and returns an Entry for each
// subdirectory that looks like a valid worktree (has a specs/ dir or is a git
// worktree). Entries already present in the registry (by name) are skipped so
// the registry takes precedence for metadata.
func ScanWorktreesDir(gitRoot string, reg Registry) ([]Entry, error) {
	dir := WorktreesDir(gitRoot)
	infos, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scanning worktrees dir: %w", err)
	}

	registered := make(map[string]bool, len(reg.Entries))
	for _, e := range reg.Entries {
		registered[e.Name] = true
	}

	var discovered []Entry
	for _, info := range infos {
		if !info.IsDir() {
			continue
		}
		name := info.Name()
		if registered[name] {
			continue // registry entry takes precedence
		}
		wtPath := filepath.Join(dir, name)
		branch, err := CurrentBranch(wtPath)
		if err != nil {
			continue // not a git repo / not accessible — skip
		}
		discovered = append(discovered, Entry{
			Name:      name,
			Branch:    branch,
			Path:      wtPath,
			CreatedAt: time.Time{}, // unknown for auto-discovered worktrees
		})
	}
	return discovered, nil
}
