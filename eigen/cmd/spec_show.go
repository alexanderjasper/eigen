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
	Use:   "show <path>",
	Short: "Print the current spec.yaml for a module",
	Args:  cobra.ExactArgs(1),
	RunE:  runSpecShow,
}

func runSpecShow(cmd *cobra.Command, args []string) error {
	path := args[0]
	data, err := os.ReadFile(storage.SpecPath(specsRoot, path))
	if err != nil {
		return fmt.Errorf("reading spec.yaml for %q: %w", path, err)
	}
	fmt.Print(string(data))
	return nil
}
