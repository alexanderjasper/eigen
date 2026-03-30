package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/alexanderjasper/eigen/internal/spec"
	"github.com/alexanderjasper/eigen/internal/storage"
)

var specChangeEdit bool

func init() {
	specChangeCmd.Flags().BoolVar(&specChangeEdit, "edit", false, "open $EDITOR after writing the template")
	specCmd.AddCommand(specChangeCmd)
}

var specChangeCmd = &cobra.Command{
	Use:   "change <path>",
	Short: "Record a new change for a spec module",
	Args:  cobra.ExactArgs(1),
	RunE:  runSpecChange,
}

func runSpecChange(cmd *cobra.Command, args []string) error {
	path := args[0]

	modulePath := storage.ModulePath(specsRoot, path)
	if _, err := os.Stat(modulePath); os.IsNotExist(err) {
		return fmt.Errorf("module %q does not exist", path)
	}

	seq, err := storage.NextSequence(specsRoot, path)
	if err != nil {
		return err
	}

	template := buildChangeTemplate(seq)

	if !specChangeEdit {
		return writeChangeDirect(path, seq, template)
	}
	return writeChangeViaEditor(path, seq, template)
}

// writeChangeDirect writes the template straight to changes/ and reprojects.
func writeChangeDirect(path string, seq int, template string) error {
	filename := fmt.Sprintf("%03d_initial.yaml", seq)
	changePath := filepath.Join(storage.ChangesPath(specsRoot, path), filename)
	if err := os.WriteFile(changePath, []byte(template), 0644); err != nil {
		return fmt.Errorf("writing change file: %w", err)
	}
	if err := reprojectModule(path); err != nil {
		return err
	}
	fmt.Println(changePath)
	return nil
}

// writeChangeViaEditor opens $EDITOR, then writes and reprojects on save.
func writeChangeViaEditor(path string, seq int, template string) error {
	tmpFile, err := os.CreateTemp("", "eigen-change-*.yaml")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(template); err != nil {
		tmpFile.Close()
		return fmt.Errorf("writing template: %w", err)
	}
	tmpFile.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	editorCmd := exec.Command(editor, tmpPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr
	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("editor exited with error: %w", err)
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("reading edited file: %w", err)
	}

	var ch spec.Change
	if err := yaml.Unmarshal(data, &ch); err != nil {
		return fmt.Errorf("parsing change: %w", err)
	}
	if ch.ID == "" {
		return fmt.Errorf("change id is required")
	}
	ch.Sequence = seq

	slug := slugify(ch.Summary)
	if slug == "" {
		slug = "change"
	}

	filename := fmt.Sprintf("%03d_%s.yaml", seq, slug)
	changePath := filepath.Join(storage.ChangesPath(specsRoot, path), filename)
	if err := os.WriteFile(changePath, data, 0644); err != nil {
		return fmt.Errorf("writing change file: %w", err)
	}

	if err := reprojectModule(path); err != nil {
		return err
	}
	fmt.Printf("Recorded change %d for %s\n", seq, path)
	return nil
}

func buildChangeTemplate(seq int) string {
	return fmt.Sprintf(`format: eigen/v1
id: chg-%03d
sequence: %d
timestamp: %s
author: ""
type: updated
summary: ""
reason: |
  TODO: explain why this change is being made.
changes:
  # Include only fields that are changing.
  # title: ""
  # owner: ""
  # status: draft
  # description: |
  #   ...
  # behavior: |
  #   ...
  # acceptance_criteria:
  #   - id: AC-001
  #     description: ""
  #     given: ""
  #     when: ""
  #     then: ""
`, seq, seq, time.Now().UTC().Format(time.RFC3339))
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlnum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}
