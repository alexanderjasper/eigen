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
