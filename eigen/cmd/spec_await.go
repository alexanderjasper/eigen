package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/alexanderjasper/eigen/internal/storage"
)

var (
	awaitTimeout  time.Duration
	awaitInterval time.Duration
)

func init() {
	specCmd.AddCommand(specAwaitApprovalCmd)
	specAwaitApprovalCmd.Flags().DurationVar(&awaitTimeout, "timeout", 5*time.Minute, "maximum time to wait for approval")
	specAwaitApprovalCmd.Flags().DurationVar(&awaitInterval, "interval", 3*time.Second, "poll interval")
}

var specAwaitApprovalCmd = &cobra.Command{
	Use:   "await-approval <module-path>",
	Short: "Poll change files until all are approved or one is rejected",
	Args:  cobra.ExactArgs(1),
	RunE:  runSpecAwaitApproval,
}

func runSpecAwaitApproval(cmd *cobra.Command, args []string) error {
	modulePath := args[0]

	deadline := time.After(awaitTimeout)
	ticker := time.NewTicker(awaitInterval)
	defer ticker.Stop()

	consecutiveFailures := 0

	for {
		changes, err := storage.ReadChanges(specsRoot, modulePath)
		if err != nil {
			consecutiveFailures++
			fmt.Fprintf(os.Stderr, "error reading changes (attempt %d/5): %v\n", consecutiveFailures, err)
			if consecutiveFailures >= 5 {
				return fmt.Errorf("giving up after %d consecutive read failures: %v", consecutiveFailures, err)
			}
		} else {
			consecutiveFailures = 0

			// Check for any change with a review_comment (rejection)
			hasComment := false
			for _, c := range changes {
				if c.ReviewComment != "" {
					hasComment = true
					break
				}
			}
			if hasComment {
				data, _ := json.Marshal(changes)
				fmt.Println("REJECTED")
				fmt.Println(string(data))
				return nil
			}

			// Check if no draft changes remain
			hasDraft := false
			for _, c := range changes {
				if c.Status == "" || c.Status == "draft" {
					hasDraft = true
					break
				}
			}
			if !hasDraft {
				data, _ := json.Marshal(changes)
				fmt.Println("APPROVED")
				fmt.Println(string(data))
				return nil
			}
		}

		select {
		case <-deadline:
			return fmt.Errorf("timed out after %s waiting for approval", awaitTimeout)
		case <-ticker.C:
			// next iteration
		}
	}
}
