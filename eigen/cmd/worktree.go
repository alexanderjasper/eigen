package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/alexanderjasper/eigen/internal/worktree"
)

var worktreeCmd = &cobra.Command{
	Use:   "worktree",
	Short: "Manage git worktrees for eigen projects",
}

// ── worktree create ──────────────────────────────────────────────────────────

var worktreeCreateCmd = &cobra.Command{
	Use:   "create <branch-name>",
	Short: "Create a new git worktree for the given branch",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorktreeCreate,
}

func runWorktreeCreate(cmd *cobra.Command, args []string) error {
	branchName := args[0]
	// Derive the worktree name from the branch by replacing "/" with "-".
	name := strings.ReplaceAll(branchName, "/", "-")

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting cwd: %w", err)
	}
	gitRoot, err := worktree.FindGitRoot(cwd)
	if err != nil {
		return fmt.Errorf("finding git root: %w", err)
	}

	worktreePath := filepath.Join(gitRoot, ".claude", "worktrees", name)

	// Check if the path already exists (AC-013).
	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("worktree directory already exists: %s", worktreePath)
	}

	// Create the git worktree with a new branch.
	gitCmd := exec.Command("git", "-C", gitRoot, "worktree", "add", worktreePath, "-b", branchName)
	gitCmd.Stdout = os.Stdout
	gitCmd.Stderr = os.Stderr
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("git worktree add: %w", err)
	}

	// Create .claude/ directory inside the worktree.
	claudeDir := filepath.Join(worktreePath, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("creating .claude dir: %w", err)
	}

	// Copy settings.json if it exists in the main repo (AC-003).
	srcSettings := filepath.Join(gitRoot, ".claude", "settings.json")
	if data, err := os.ReadFile(srcSettings); err == nil {
		dstSettings := filepath.Join(claudeDir, "settings.json")
		if err := os.WriteFile(dstSettings, data, 0644); err != nil {
			return fmt.Errorf("copying settings.json: %w", err)
		}
	}

	// Register the new worktree in .eigen/worktrees.json (AC-002).
	entry := worktree.Entry{
		Name:      name,
		Branch:    branchName,
		Path:      worktreePath,
		CreatedAt: time.Now().UTC(),
	}
	if err := worktree.AddEntry(gitRoot, entry); err != nil {
		return fmt.Errorf("registering worktree: %w", err)
	}

	// Print path and hint (AC-004).
	relPath := filepath.Join(".claude", "worktrees", name)
	fmt.Printf("Created worktree: %s\n", worktreePath)
	fmt.Printf("Next step: claude --project %s\n", relPath)
	return nil
}

// ── worktree list ────────────────────────────────────────────────────────────

var worktreeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered worktrees",
	RunE:  runWorktreeList,
}

func runWorktreeList(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting cwd: %w", err)
	}
	gitRoot, err := worktree.FindGitRoot(cwd)
	if err != nil {
		return fmt.Errorf("finding git root: %w", err)
	}

	reg, err := worktree.ReadRegistry(gitRoot)
	if err != nil {
		return fmt.Errorf("reading registry: %w", err)
	}

	mainBranch, _ := worktree.CurrentBranch(gitRoot)
	mainHash, _ := worktree.GitShortHash(gitRoot)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tBRANCH\tPATH\tHEAD\tSTATUS")

	// Main worktree is always listed first (AC-005).
	fmt.Fprintf(w, "main\t%s\t%s\t%s\tactive\n", mainBranch, gitRoot, mainHash)

	// Then registered entries.
	for _, e := range reg.Entries {
		status := "active"
		if _, err := os.Stat(e.Path); os.IsNotExist(err) {
			status = "orphaned"
		}
		hash, _ := worktree.GitShortHash(e.Path)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", e.Name, e.Branch, e.Path, hash, status)
	}

	return w.Flush()
}

// ── worktree remove ──────────────────────────────────────────────────────────

var worktreeRemoveForce bool

var worktreeRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a registered git worktree",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorktreeRemove,
}

func runWorktreeRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting cwd: %w", err)
	}
	gitRoot, err := worktree.FindGitRoot(cwd)
	if err != nil {
		return fmt.Errorf("finding git root: %w", err)
	}

	reg, err := worktree.ReadRegistry(gitRoot)
	if err != nil {
		return fmt.Errorf("reading registry: %w", err)
	}

	// Find the entry.
	var entry *worktree.Entry
	for i := range reg.Entries {
		if reg.Entries[i].Name == name {
			entry = &reg.Entries[i]
			break
		}
	}
	if entry == nil {
		return fmt.Errorf("worktree %q not found in registry", name)
	}

	// Check for uncommitted changes (AC-007).
	if !worktreeRemoveForce {
		statusCmd := exec.Command("git", "-C", entry.Path, "status", "--porcelain")
		out, err := statusCmd.Output()
		if err == nil && len(strings.TrimSpace(string(out))) > 0 {
			return fmt.Errorf("worktree %q has uncommitted changes; use --force to remove anyway", name)
		}
	}

	// Remove git worktree registration.
	removeArgs := []string{"-C", gitRoot, "worktree", "remove"}
	if worktreeRemoveForce {
		removeArgs = append(removeArgs, "--force")
	}
	removeArgs = append(removeArgs, entry.Path)
	gitCmd := exec.Command("git", removeArgs...)
	gitCmd.Stdout = io.Discard
	gitCmd.Stderr = os.Stderr
	if err := gitCmd.Run(); err != nil {
		// If git worktree remove fails (e.g. path already gone), continue with cleanup.
		fmt.Fprintf(os.Stderr, "warning: git worktree remove: %v\n", err)
	}

	// Belt-and-suspenders: remove directory (AC-006).
	if err := os.RemoveAll(entry.Path); err != nil {
		return fmt.Errorf("removing worktree directory: %w", err)
	}

	// Remove from registry.
	if _, err := worktree.RemoveEntry(gitRoot, name); err != nil {
		return fmt.Errorf("removing registry entry: %w", err)
	}

	fmt.Printf("Removed worktree: %s\n", name)
	return nil
}

func init() {
	worktreeRemoveCmd.Flags().BoolVar(&worktreeRemoveForce, "force", false, "remove even if the worktree has uncommitted changes")

	worktreeCmd.AddCommand(worktreeCreateCmd)
	worktreeCmd.AddCommand(worktreeListCmd)
	worktreeCmd.AddCommand(worktreeRemoveCmd)
}
