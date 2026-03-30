package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/alexanderjasper/eigen/internal/storage"
)

var specListAllFlag bool

func init() {
	specCmd.AddCommand(specListCmd)
	specListCmd.Flags().BoolVar(&specListAllFlag, "all", false, "Include removed modules")
}

var specListCmd = &cobra.Command{
	Use:   "list [prefix]",
	Short: "List all spec modules",
	Long:  "List all spec modules, optionally filtered by path prefix (e.g. spec-cli).",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSpecList,
}

var statusColors = map[string]func(...interface{}) string{
	"draft":      color.New(color.FgYellow).SprintFunc(),
	"stable":     color.New(color.FgGreen).SprintFunc(),
	"approved":   color.New(color.FgGreen).SprintFunc(),
	"compiled":   color.New(color.FgCyan).SprintFunc(),
	"deprecated": color.New(color.FgRed).SprintFunc(),
	"removed":    color.New(color.FgHiBlack).SprintFunc(),
}

func runSpecList(cmd *cobra.Command, args []string) error {
	prefix := ""
	if len(args) == 1 {
		prefix = args[0]
	}

	refs, err := storage.WalkModules(specsRoot, prefix)
	if err != nil {
		return fmt.Errorf("listing modules: %w", err)
	}

	if len(refs) == 0 {
		fmt.Println("No modules found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PATH\tOWNER\tSTATUS\tTITLE")
	fmt.Fprintln(w, "────\t─────\t──────\t─────")

	for _, ref := range refs {
		s, err := storage.ReadSpec(specsRoot, ref.Path)
		if err != nil {
			fmt.Fprintf(w, "%s\t?\t?\t(error reading spec)\n", ref.Path)
			continue
		}
		if s.Status == "removed" && !specListAllFlag {
			continue
		}
		displayStatus := s.Status
		if s.Status == "deprecated" {
			displayStatus = s.Status + " [deprecated]"
		}
		if colorFn, ok := statusColors[s.Status]; ok {
			displayStatus = colorFn(displayStatus)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", ref.Path, s.Owner, displayStatus, s.Title)
	}
	w.Flush()
	return nil
}
