package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/alexanderjasper/eigen/internal/spec"
	"github.com/alexanderjasper/eigen/internal/storage"
)

func init() {
	specCmd.AddCommand(specProjectCmd)
}

var specProjectCmd = &cobra.Command{
	Use:   "project [path]",
	Short: "Reproject spec.yaml from events",
	Long:  "Recompute and overwrite spec.yaml for one module (or all modules when no path is given).",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSpecProject,
}

func runSpecProject(cmd *cobra.Command, args []string) error {
	if len(args) == 1 {
		return reprojectModule(args[0])
	}

	refs, err := storage.WalkModules(specsRoot, "")
	if err != nil {
		return fmt.Errorf("listing modules: %w", err)
	}
	for _, ref := range refs {
		if err := reprojectModule(ref.Path); err != nil {
			return err
		}
	}
	return nil
}

// reprojectModule reads all events for path, projects, and writes spec.yaml.
func reprojectModule(path string) error {
	events, err := storage.ReadEvents(specsRoot, path)
	if err != nil {
		return fmt.Errorf("reading events for %s: %w", path, err)
	}
	s := spec.Project(path, events)
	if err := storage.WriteSpec(specsRoot, path, s); err != nil {
		return fmt.Errorf("writing spec.yaml for %s: %w", path, err)
	}
	fmt.Printf("Projected %s\n", storage.SpecPath(specsRoot, path))
	return nil
}
