package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alexanderjasper/eigen/internal/spec"
	"github.com/alexanderjasper/eigen/internal/storage"
)

var projectAll bool

func init() {
	specProjectCmd.Flags().BoolVar(&projectAll, "all", false, "Reproject all modules")
	specCmd.AddCommand(specProjectCmd)
	specCmd.AddCommand(specProjectAllCmd)
}

var specProjectCmd = &cobra.Command{
	Use:   "project [path]",
	Short: "Reproject spec.yaml from changes",
	Long:  "Recompute and overwrite spec.yaml for one module (or all modules when --all is given).",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSpecProject,
}

var specProjectAllCmd = &cobra.Command{
	Use:   "project-all",
	Short: "Reproject spec.yaml for every module",
	Long:  "Recompute and overwrite spec.yaml for every module found under the specs root.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectAll = true
		return runSpecProject(cmd, args)
	},
}

func runSpecProject(cmd *cobra.Command, args []string) error {
	// Determine scope.
	var paths []string
	if projectAll {
		refs, err := storage.WalkModules(specsRoot, "")
		if err != nil {
			return fmt.Errorf("listing modules: %w", err)
		}
		for _, ref := range refs {
			paths = append(paths, ref.Path)
		}
	} else if len(args) == 1 {
		paths = []string{args[0]}
	} else {
		return fmt.Errorf("provide a module path or use --all to reproject every module")
	}

	// Pre-validate all change files across the full scope before projecting anything.
	var lintErrs []spec.LintError
	for _, p := range paths {
		errs, err := lintModule(p)
		if err != nil {
			return err
		}
		lintErrs = append(lintErrs, errs...)
	}
	if len(lintErrs) > 0 {
		for _, e := range lintErrs {
			fmt.Fprintln(os.Stderr, e.Error())
		}
		return fmt.Errorf("pre-validation failed: %d error(s) found in change files; no spec.yaml files written", len(lintErrs))
	}

	// All files clean — project each module.
	for _, p := range paths {
		if err := reprojectModule(p); err != nil {
			return err
		}
	}
	return nil
}

// lintModule reads all .yaml files from a module's changes/ directory and runs
// LintChangeFile on each, returning all lint errors found.
func lintModule(path string) ([]spec.LintError, error) {
	dir := storage.ChangesPath(specsRoot, path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading changes for %s: %w", path, err)
	}

	var errs []spec.LintError
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		filePath := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("reading change file %s: %w", filePath, err)
		}
		errs = append(errs, spec.LintChangeFile(filePath, data)...)
	}
	return errs, nil
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
	s, err := spec.Project(path, changePtrs)
	if err != nil {
		return fmt.Errorf("projecting %s: %w", path, err)
	}
	if err := storage.WriteSpec(specsRoot, path, s); err != nil {
		return fmt.Errorf("writing spec.yaml for %s: %w", path, err)
	}
	fmt.Printf("Projected %s\n", storage.SpecPath(specsRoot, path))
	return nil
}
