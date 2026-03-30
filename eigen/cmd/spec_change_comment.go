package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/alexanderjasper/eigen/internal/storage"
)

func init() {
	specCmd.AddCommand(specChangeCommentCmd)
}

var specChangeCommentCmd = &cobra.Command{
	Use:   "change-comment <module-path> <filename> <comment>",
	Short: "Set the review_comment of a change file",
	Args:  cobra.ExactArgs(3),
	RunE:  runSpecChangeComment,
}

func runSpecChangeComment(cmd *cobra.Command, args []string) error {
	modulePath := args[0]
	filename := args[1]
	comment := args[2]

	if err := storage.SetChangeComment(specsRoot, modulePath, filename, comment); err != nil {
		return err
	}

	fmt.Printf("Set review_comment of %s to %q\n", filename, comment)
	return nil
}
