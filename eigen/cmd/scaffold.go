package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed skills/eigen-change-spec.md
var eigenChangeSpecSkill []byte

//go:embed skills/eigen-change-compile.md
var eigenChangeCompileSkill []byte

//go:embed skills/eigen-change.md
var eigenChangeSkill []byte

//go:embed skills/eigen-change-review.md
var eigenChangeReviewSkill []byte

//go:embed agents/spec-agent.md
var specAgentDef []byte

//go:embed agents/plan-agent.md
var planAgentDef []byte

//go:embed agents/compile-agent.md
var compileAgentDef []byte

//go:embed agents/review-agent.md
var reviewAgentDef []byte

var scaffoldForce bool
var scaffoldNoHooks bool

func init() {
	rootCmd.AddCommand(scaffoldCmd)
	scaffoldCmd.Flags().BoolVarP(&scaffoldForce, "force", "f", false, "overwrite existing skill and agent files")
	scaffoldCmd.Flags().BoolVar(&scaffoldNoHooks, "no-hooks", false, "skip git pre-commit hook installation")
}

const hookBegin = "# BEGIN eigen scaffold"
const hookEnd = "# END eigen scaffold"

func installHook(target string) (string, error) {
	hookPath := filepath.Join(target, ".git", "hooks", "pre-commit")
	if err := os.MkdirAll(filepath.Dir(hookPath), 0755); err != nil {
		return "", fmt.Errorf("creating hooks directory: %w", err)
	}

	hookBlock := hookBegin + `
STAGED_SPECS=$(git diff --cached --name-only | grep '^specs/')
if [ -n "$STAGED_SPECS" ]; then
  if ! eigen spec project-all; then
    echo "eigen: spec project-all failed; commit rejected" >&2
    exit 1
  fi
  if ! eigen spec validate; then
    echo "eigen: spec validate failed; commit rejected" >&2
    exit 1
  fi
fi
` + hookEnd

	existing, err := os.ReadFile(hookPath)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("reading pre-commit hook: %w", err)
	}

	var content string
	if os.IsNotExist(err) {
		content = "#!/bin/sh\n" + hookBlock
	} else {
		if strings.Contains(string(existing), hookBegin) {
			// already present, idempotent
			if err := os.Chmod(hookPath, 0755); err != nil {
				return "", fmt.Errorf("chmod pre-commit hook: %w", err)
			}
			return hookPath, nil
		}
		content = string(existing) + "\n" + hookBlock
	}

	if err := os.WriteFile(hookPath, []byte(content), 0755); err != nil {
		return "", fmt.Errorf("writing pre-commit hook: %w", err)
	}
	if err := os.Chmod(hookPath, 0755); err != nil {
		return "", fmt.Errorf("chmod pre-commit hook: %w", err)
	}
	return hookPath, nil
}

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold [path]",
	Short: "Initialize a new project with eigen Claude skills",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runScaffold,
}

func runScaffold(cmd *cobra.Command, args []string) error {
	target := "."
	if len(args) == 1 {
		target = args[0]
	}

	skills := []struct {
		name    string
		content []byte
	}{
		{"eigen-change-spec", eigenChangeSpecSkill},
		{"eigen-change-compile", eigenChangeCompileSkill},
		{"eigen-change", eigenChangeSkill},
		{"eigen-change-review", eigenChangeReviewSkill},
	}

	agents := []struct {
		name    string
		content []byte
	}{
		{"spec-agent", specAgentDef},
		{"plan-agent", planAgentDef},
		{"compile-agent", compileAgentDef},
		{"review-agent", reviewAgentDef},
	}

	// AC-004: check for existing files before writing anything
	var existing []string
	for _, s := range skills {
		p := filepath.Join(target, ".claude", "skills", s.name, "SKILL.md")
		if _, err := os.Stat(p); err == nil {
			existing = append(existing, p)
		}
	}
	for _, a := range agents {
		p := filepath.Join(target, ".claude", "agents", a.name+".md")
		if _, err := os.Stat(p); err == nil {
			existing = append(existing, p)
		}
	}
	if !scaffoldForce {
		if len(existing) > 0 {
			for _, p := range existing {
				fmt.Fprintf(os.Stderr, "already exists: %s\n", p)
			}
			return fmt.Errorf("skill/agent files already exist; remove them first to re-scaffold")
		}
	} else {
		// AC-006: remove existing files so they are cleanly rewritten
		for _, s := range skills {
			p := filepath.Join(target, ".claude", "skills", s.name, "SKILL.md")
			if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("removing existing skill file: %w", err)
			}
		}
		for _, a := range agents {
			p := filepath.Join(target, ".claude", "agents", a.name+".md")
			if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("removing existing agent file: %w", err)
			}
		}
	}

	// AC-001: write skill files
	var created []string
	for _, s := range skills {
		p := filepath.Join(target, ".claude", "skills", s.name, "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			return fmt.Errorf("creating skill directory: %w", err)
		}
		if err := os.WriteFile(p, s.content, 0644); err != nil {
			return fmt.Errorf("writing skill file: %w", err)
		}
		created = append(created, p)
	}

	// write agent definition files
	for _, a := range agents {
		p := filepath.Join(target, ".claude", "agents", a.name+".md")
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			return fmt.Errorf("creating agents directory: %w", err)
		}
		if err := os.WriteFile(p, a.content, 0644); err != nil {
			return fmt.Errorf("writing agent file: %w", err)
		}
		created = append(created, p)
	}

	// AC-002: create specs/ directory
	specsDir := filepath.Join(target, "specs")
	if err := os.MkdirAll(specsDir, 0755); err != nil {
		return fmt.Errorf("creating specs directory: %w", err)
	}

	// AC-008/AC-015: install pre-commit hook by default
	var hookPath string
	if !scaffoldNoHooks {
		var err error
		hookPath, err = installHook(target)
		if err != nil {
			return fmt.Errorf("installing pre-commit hook: %w", err)
		}
	}

	// AC-005/AC-013: print created files and hint
	fmt.Println("Scaffolded eigen project:")
	for _, p := range created {
		fmt.Printf("  %s\n", p)
	}
	fmt.Printf("  %s/\n", specsDir)
	if hookPath != "" {
		fmt.Printf("  %s\n", hookPath)
	}
	fmt.Println("\nNext: eigen spec new <module-name>")
	return nil
}
