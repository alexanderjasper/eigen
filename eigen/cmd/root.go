package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var specsRoot string

var rootCmd = &cobra.Command{
	Use:   "eigen",
	Short: "Eigen — spec-driven development CLI",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentPreRunE = resolveSpecsRoot
	rootCmd.PersistentFlags().StringVar(&specsRoot, "specs", "", "path to specs directory (default: EIGEN_SPECS env or ../specs)")

	rootCmd.AddCommand(specCmd)
}

func resolveSpecsRoot(cmd *cobra.Command, args []string) error {
	if specsRoot != "" {
		return nil
	}
	if env := os.Getenv("EIGEN_SPECS"); env != "" {
		specsRoot = env
		return nil
	}
	// Walk up from CWD looking for a specs/ directory.
	cwd, err := os.Getwd()
	if err == nil {
		for dir := cwd; ; dir = filepath.Dir(dir) {
			candidate := filepath.Join(dir, "specs")
			if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
				specsRoot = candidate
				return nil
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
	// Fallback: ../specs relative to the executable.
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}
	specsRoot = filepath.Join(filepath.Dir(exe), "..", "specs")
	return nil
}

var specCmd = &cobra.Command{
	Use:   "spec",
	Short: "Manage specs",
}
