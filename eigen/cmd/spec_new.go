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

	changesDir := storage.ChangesPath(specsRoot, path)
	if err := os.MkdirAll(changesDir, 0755); err != nil {
		return fmt.Errorf("creating changes directory: %w", err)
	}

	_, module := splitPath(path)
	ch := spec.Change{
		ID:        "chg-001",
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

	data, err := yaml.Marshal(ch)
	if err != nil {
		return fmt.Errorf("marshaling initial change: %w", err)
	}
	changePath := filepath.Join(changesDir, "001_initial.yaml")
	if err := os.WriteFile(changePath, data, 0644); err != nil {
		return fmt.Errorf("writing initial change: %w", err)
	}

	changes, err := storage.ReadChanges(specsRoot, path)
	if err != nil {
		return fmt.Errorf("reading changes: %w", err)
	}
	// Convert []Change to []*Change
	changePtrs := make([]*spec.Change, len(changes))
	for i := range changes {
		changePtrs[i] = &changes[i]
	}
	s := spec.Project(path, changePtrs)
	if err := storage.WriteSpec(specsRoot, path, s); err != nil {
		return fmt.Errorf("writing spec.yaml: %w", err)
	}

	fmt.Printf("Created %s\n", path)
	fmt.Printf("  %s\n", changePath)
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
