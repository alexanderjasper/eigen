package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/alexanderjasper/eigen/internal/spec"
	"github.com/alexanderjasper/eigen/internal/storage"
)

func init() {
	specCmd.AddCommand(specNewCmd)
}

var specNewCmd = &cobra.Command{
	Use:   "new <path>",
	Short: "Scaffold a new spec module",
	Long:  "Create a new spec module at the given path (e.g. spec-cli or spec-cli/cmd-new).",
	Args:  cobra.ExactArgs(1),
	RunE:  runSpecNew,
}

func runSpecNew(cmd *cobra.Command, args []string) error {
	path := args[0]

	modulePath := storage.ModulePath(specsRoot, path)
	if _, err := os.Stat(modulePath); err == nil {
		return fmt.Errorf("module %q already exists at %s", path, modulePath)
	}

	eventsDir := storage.EventsPath(specsRoot, path)
	if err := os.MkdirAll(eventsDir, 0755); err != nil {
		return fmt.Errorf("creating events directory: %w", err)
	}

	_, module := splitPath(path)
	ev := spec.ChangeEvent{
		ID:        "evt-001",
		Sequence:  1,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Author:    "",
		Type:      "created",
		Summary:   "Initial spec for " + module,
		Reason:    "TODO: describe why this module exists.\n",
		Changes: spec.ChangeSet{
			Title:       module,
			Owner:       "",
			Status:      "draft",
			Description: "TODO: describe what this module does.\n",
			Behavior:    "TODO: describe how this module behaves.\n",
			AcceptanceCriteria: []spec.AcceptanceCriterion{
				{
					ID:          "AC-001",
					Description: "TODO: describe what this criterion verifies",
					Given:       "TODO",
					When:        "TODO",
					Then:        "TODO",
				},
			},
		},
	}

	data, err := yaml.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshaling initial event: %w", err)
	}
	eventPath := filepath.Join(eventsDir, "001_initial.yaml")
	if err := os.WriteFile(eventPath, data, 0644); err != nil {
		return fmt.Errorf("writing initial event: %w", err)
	}

	events, err := storage.ReadEvents(specsRoot, path)
	if err != nil {
		return fmt.Errorf("reading events: %w", err)
	}
	s := spec.Project(path, events)
	if err := storage.WriteSpec(specsRoot, path, s); err != nil {
		return fmt.Errorf("writing spec.yaml: %w", err)
	}

	fmt.Printf("Created %s\n", path)
	fmt.Printf("  %s\n", eventPath)
	fmt.Printf("  %s\n", storage.SpecPath(specsRoot, path))
	return nil
}

// splitPath returns the first and last segments of a slash path.
func splitPath(path string) (first, last string) {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i], path[i+1:]
		}
	}
	return path, path
}
