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

func init() {
	specCmd.AddCommand(specEventCmd)
}

var specEventCmd = &cobra.Command{
	Use:   "event <path>",
	Short: "Record a new change event for a spec module",
	Args:  cobra.ExactArgs(1),
	RunE:  runSpecEvent,
}

func runSpecEvent(cmd *cobra.Command, args []string) error {
	path := args[0]

	modulePath := storage.ModulePath(specsRoot, path)
	if _, err := os.Stat(modulePath); os.IsNotExist(err) {
		return fmt.Errorf("module %q does not exist", path)
	}

	seq, err := storage.NextSequence(specsRoot, path)
	if err != nil {
		return err
	}

	template := buildEventTemplate(seq)

	tmpFile, err := os.CreateTemp("", "eigen-event-*.yaml")
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

	var ev spec.ChangeEvent
	if err := yaml.Unmarshal(data, &ev); err != nil {
		return fmt.Errorf("parsing event: %w", err)
	}
	if ev.ID == "" {
		return fmt.Errorf("event id is required")
	}
	ev.Sequence = seq

	slug := slugify(ev.Summary)
	if slug == "" {
		slug = "event"
	}

	filename := fmt.Sprintf("%03d_%s.yaml", seq, slug)
	eventPath := filepath.Join(storage.EventsPath(specsRoot, path), filename)
	if err := os.WriteFile(eventPath, data, 0644); err != nil {
		return fmt.Errorf("writing event file: %w", err)
	}

	events, err := storage.ReadEvents(specsRoot, path)
	if err != nil {
		return fmt.Errorf("reading events: %w", err)
	}
	s := spec.Project(path, events)
	if err := storage.WriteSpec(specsRoot, path, s); err != nil {
		return fmt.Errorf("writing spec.yaml: %w", err)
	}

	fmt.Printf("Recorded event %d for %s\n", seq, path)
	return nil
}

func buildEventTemplate(seq int) string {
	return fmt.Sprintf(`id: evt-%03d
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
