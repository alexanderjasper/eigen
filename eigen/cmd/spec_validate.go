package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/alexanderjasper/eigen/internal/spec"
	"github.com/alexanderjasper/eigen/internal/storage"
)

func init() {
	specCmd.AddCommand(specValidateCmd)
}

var specValidateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Validate spec modules",
	Long:  "Validate a specific module by path, or all modules if no path is given.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSpecValidate,
}

func runSpecValidate(cmd *cobra.Command, args []string) error {
	var paths []string

	if len(args) == 1 {
		paths = []string{args[0]}
	} else {
		refs, err := storage.WalkModules(specsRoot, "")
		if err != nil {
			return err
		}
		for _, r := range refs {
			paths = append(paths, r.Path)
		}
	}

	errColor := color.New(color.FgRed).SprintFunc()
	okColor := color.New(color.FgGreen).SprintFunc()

	totalErrors := 0
	for _, path := range paths {
		s, err := storage.ReadSpec(specsRoot, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s %s: %s\n", errColor("✗"), path, err)
			totalErrors++
			continue
		}

		errs := spec.Validate(s, specsRoot)
		if len(errs) == 0 {
			fmt.Printf("%s %s\n", okColor("✓"), path)
			continue
		}

		fmt.Fprintf(os.Stderr, "%s %s\n", errColor("✗"), path)
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "    %s\n", e.Error())
		}
		totalErrors += len(errs)
	}

	if totalErrors > 0 {
		os.Exit(1)
	}
	return nil
}
