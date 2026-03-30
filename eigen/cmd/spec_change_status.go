package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alexanderjasper/eigen/internal/storage"
)

var commitFlags []string

func init() {
	specCmd.AddCommand(specChangeStatusCmd)
	specChangeStatusCmd.Flags().StringArrayVar(&commitFlags, "commit", nil,
		"commit hash to record in compiled_commits (repeatable)")
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

	var commits []string
	if status == "compiled" {
		if len(commitFlags) > 0 {
			commits = commitFlags
		} else if hash, err := gitHeadHash(); err == nil && hash != "" {
			commits = []string{hash}
		}
		// err non-nil (not a git repo) → commits stays nil → field omitted
	}

	if err := storage.SetChangeStatus(specsRoot, modulePath, filename, status, commits); err != nil {
		return err
	}

	fmt.Printf("Set status of %s to %q\n", filename, status)
	return nil
}

func gitHeadHash() (string, error) {
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
