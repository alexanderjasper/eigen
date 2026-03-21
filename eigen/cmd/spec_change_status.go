package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/alexanderjasper/eigen/internal/storage"
)

func init() {
	specCmd.AddCommand(specChangeStatusCmd)
}

var specChangeStatusCmd = &cobra.Command{
	Use:   "change-status <module-path> <filename> <status>",
	Short: "Set the status of a change file",
	Args:  cobra.ExactArgs(3),
	RunE:  runSpecChangeStatus,
}

var validStatuses = map[string]bool{
	"draft":    true,
	"approved": true,
	"compiled": true,
}

func runSpecChangeStatus(cmd *cobra.Command, args []string) error {
	modulePath := args[0]
	filename := args[1]
	status := args[2]

	if !validStatuses[status] {
		return fmt.Errorf("invalid status %q: must be one of draft, approved, compiled", status)
	}

	if err := storage.SetChangeStatus(specsRoot, modulePath, filename, status); err != nil {
		return err
	}

	fmt.Printf("Set status of %s to %q\n", filename, status)
	return nil
}
