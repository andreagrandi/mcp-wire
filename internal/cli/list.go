package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/andreagrandi/mcp-wire/internal/service"
	"github.com/spf13/cobra"
)

var loadServices = service.LoadServices

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available services and targets",
	}

	listCmd.AddCommand(newListServicesCmd())
	rootCmd.AddCommand(listCmd)
}

func newListServicesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "services",
		Short: "List available service definitions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			services, err := loadServices()
			if err != nil {
				return fmt.Errorf("load services: %w", err)
			}

			printServicesList(cmd.OutOrStdout(), services)
			return nil
		},
	}
}

func printServicesList(output io.Writer, services map[string]service.Service) {
	fmt.Fprintln(output, "Available services:")
	fmt.Fprintln(output)

	if len(services) == 0 {
		fmt.Fprintln(output, "  (none)")
		return
	}

	names := make([]string, 0, len(services))
	maxNameWidth := 0
	for name := range services {
		names = append(names, name)
		if len(name) > maxNameWidth {
			maxNameWidth = len(name)
		}
	}

	sort.Strings(names)

	for _, name := range names {
		description := strings.TrimSpace(services[name].Description)
		if description == "" {
			fmt.Fprintf(output, "  %s\n", name)
			continue
		}

		fmt.Fprintf(output, "  %-*s  %s\n", maxNameWidth, name, description)
	}
}
