package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/alexanderjasper/eigen/internal/storage"
)

func init() {
	specCmd.AddCommand(specListCmd)
}

var specListCmd = &cobra.Command{
	Use:   "list [domain]",
	Short: "List all spec modules",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSpecList,
}

var statusColors = map[string]func(...interface{}) string{
	"draft":      color.New(color.FgYellow).SprintFunc(),
	"stable":     color.New(color.FgGreen).SprintFunc(),
	"deprecated": color.New(color.FgRed).SprintFunc(),
}

func runSpecList(cmd *cobra.Command, args []string) error {
	domainFilter := ""
	if len(args) == 1 {
		domainFilter = args[0]
	}

	refs, err := storage.WalkModules(specsRoot, domainFilter)
	if err != nil {
		return fmt.Errorf("listing modules: %w", err)
	}

	if len(refs) == 0 {
		fmt.Println("No modules found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tOWNER\tSTATUS\tTITLE")
	fmt.Fprintln(w, "──\t─────\t──────\t─────")

	for _, ref := range refs {
		s, err := storage.ReadSpec(specsRoot, ref.Domain, ref.Module)
		if err != nil {
			fmt.Fprintf(w, "%s.%s\t?\t?\t(error reading spec)\n", ref.Domain, ref.Module)
			continue
		}
		statusStr := s.Status
		if colorFn, ok := statusColors[s.Status]; ok {
			statusStr = colorFn(s.Status)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.ID, s.Owner, statusStr, s.Title)
	}
	w.Flush()
	return nil
}
