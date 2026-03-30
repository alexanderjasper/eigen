package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/alexanderjasper/eigen/internal/server"
	"github.com/alexanderjasper/eigen/internal/worktree"
)

var servePort int
var serveOpen bool

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the spec navigator UI",
	Long:  "Start a local HTTP server serving the spec navigator browser UI.",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 7171, "port to listen on")
	serveCmd.Flags().BoolVar(&serveOpen, "open", true, "open browser on start")
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting cwd: %w", err)
	}
	gitRoot, err := worktree.FindGitRoot(cwd)
	if err != nil {
		// If we can't find a git root, use an empty string (server handles gracefully).
		gitRoot = ""
	}
	fmt.Printf("eigen serve → http://localhost:%d\n", servePort)
	return server.Start(gitRoot, specsRoot, servePort, serveOpen)
}
