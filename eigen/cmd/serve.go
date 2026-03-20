package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/alexanderjasper/eigen/internal/server"
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
	fmt.Printf("eigen serve → http://localhost:%d\n", servePort)
	return server.Start(specsRoot, servePort, serveOpen)
}
