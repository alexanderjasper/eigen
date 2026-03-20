package cmd

import (
	"fmt"
	"os"
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
	Use:   "new <domain> <module>",
	Short: "Scaffold a new spec module",
	Args:  cobra.ExactArgs(2),
	RunE:  runSpecNew,
}

func runSpecNew(cmd *cobra.Command, args []string) error {
	domain, module := args[0], args[1]

	modulePath := storage.ModulePath(specsRoot, domain, module)
	if _, err := os.Stat(modulePath); err == nil {
		return fmt.Errorf("module %s.%s already exists at %s", domain, module, modulePath)
	}

	eventsDir := storage.EventsPath(specsRoot, domain, module)
	if err := os.MkdirAll(eventsDir, 0755); err != nil {
		return fmt.Errorf("creating events directory: %w", err)
	}

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
	eventPath := storage.EventsPath(specsRoot, domain, module) + "/001_initial.yaml"
	if err := os.WriteFile(eventPath, data, 0644); err != nil {
		return fmt.Errorf("writing initial event: %w", err)
	}

	events, err := storage.ReadEvents(specsRoot, domain, module)
	if err != nil {
		return fmt.Errorf("reading events: %w", err)
	}
	s := spec.Project(domain, module, events)
	if err := storage.WriteSpec(specsRoot, domain, module, s); err != nil {
		return fmt.Errorf("writing spec.yaml: %w", err)
	}

	fmt.Printf("Created %s.%s\n", domain, module)
	fmt.Printf("  %s\n", eventPath)
	fmt.Printf("  %s\n", storage.SpecPath(specsRoot, domain, module))
	return nil
}
