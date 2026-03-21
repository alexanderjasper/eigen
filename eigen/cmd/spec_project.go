package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alexanderjasper/eigen/internal/spec"
	"github.com/alexanderjasper/eigen/internal/storage"
)

func init() {
	specCmd.AddCommand(specProjectCmd)
}

var specProjectCmd = &cobra.Command{
	Use:   "project [path]",
	Short: "Reproject spec.yaml from changes",
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

// reprojectModule reads all changes for path, validates for no-op changes,
// projects, and writes spec.yaml.
func reprojectModule(path string) error {
	changes, err := storage.ReadChanges(specsRoot, path)
	if err != nil {
		return fmt.Errorf("reading changes for %s: %w", path, err)
	}
	// Convert []Change to []*Change
	changePtrs := make([]*spec.Change, len(changes))
	for i := range changes {
		changePtrs[i] = &changes[i]
	}
	if errs := spec.ValidateChangeLog(path, changePtrs); len(errs) > 0 {
		var msgs []string
		for _, e := range errs {
			msgs = append(msgs, "  "+e.Error())
		}
		return fmt.Errorf("no-op changes detected in %s:\n%s", path, strings.Join(msgs, "\n"))
	}
	s := spec.Project(path, changePtrs)
	if err := storage.WriteSpec(specsRoot, path, s); err != nil {
		return fmt.Errorf("writing spec.yaml for %s: %w", path, err)
	}
	fmt.Printf("Projected %s\n", storage.SpecPath(specsRoot, path))
	return nil
}
