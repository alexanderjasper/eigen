package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/alexanderjasper/eigen/internal/storage"
)

func init() {
	specCmd.AddCommand(specShowCmd)
}

var specShowCmd = &cobra.Command{
	Use:   "show <domain> <module>",
	Short: "Print the current spec.yaml for a module",
	Args:  cobra.ExactArgs(2),
	RunE:  runSpecShow,
}

func runSpecShow(cmd *cobra.Command, args []string) error {
	domain, module := args[0], args[1]

	path := storage.SpecPath(specsRoot, domain, module)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading spec.yaml for %s.%s: %w", domain, module, err)
	}
	fmt.Print(string(data))
	return nil
}
