package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed skills/eigen-change-spec.md
var eigenChangeSpecSkill []byte

//go:embed skills/eigen-change-compile.md
var eigenChangeCompileSkill []byte

//go:embed skills/eigen-change.md
var eigenChangeSkill []byte

//go:embed agents/spec-agent.md
var specAgentDef []byte

//go:embed agents/plan-agent.md
var planAgentDef []byte

//go:embed agents/compile-agent.md
var compileAgentDef []byte

var scaffoldForce bool

func init() {
	rootCmd.AddCommand(scaffoldCmd)
	scaffoldCmd.Flags().BoolVarP(&scaffoldForce, "force", "f", false, "overwrite existing skill and agent files")
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
	}

	agents := []struct {
		name    string
		content []byte
	}{
		{"spec-agent", specAgentDef},
		{"plan-agent", planAgentDef},
		{"compile-agent", compileAgentDef},
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

	// AC-005: print created files and hint
	fmt.Println("Scaffolded eigen project:")
	for _, p := range created {
		fmt.Printf("  %s\n", p)
	}
	fmt.Printf("  %s/\n", specsDir)
	fmt.Println("\nNext: eigen spec new <module-name>")
	return nil
}
