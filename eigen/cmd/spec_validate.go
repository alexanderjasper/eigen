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
	Use:   "validate [domain] [module]",
	Short: "Validate spec modules",
	Args:  cobra.MaximumNArgs(2),
	RunE:  runSpecValidate,
}

func runSpecValidate(cmd *cobra.Command, args []string) error {
	type target struct {
		domain, module string
	}

	var targets []target

	switch len(args) {
	case 2:
		targets = []target{{args[0], args[1]}}
	case 1:
		refs, err := storage.WalkModules(specsRoot, args[0])
		if err != nil {
			return err
		}
		for _, r := range refs {
			targets = append(targets, target{r.Domain, r.Module})
		}
	default:
		refs, err := storage.WalkModules(specsRoot, "")
		if err != nil {
			return err
		}
		for _, r := range refs {
			targets = append(targets, target{r.Domain, r.Module})
		}
	}

	errColor := color.New(color.FgRed).SprintFunc()
	okColor := color.New(color.FgGreen).SprintFunc()

	totalErrors := 0
	for _, t := range targets {
		s, err := storage.ReadSpec(specsRoot, t.domain, t.module)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s %s.%s: %s\n", errColor("✗"), t.domain, t.module, err)
			totalErrors++
			continue
		}

		errs := spec.Validate(s, specsRoot)
		if len(errs) == 0 {
			fmt.Printf("%s %s.%s\n", okColor("✓"), t.domain, t.module)
			continue
		}

		fmt.Fprintf(os.Stderr, "%s %s.%s\n", errColor("✗"), t.domain, t.module)
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
