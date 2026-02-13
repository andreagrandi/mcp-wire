package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/andreagrandi/mcp-wire/internal/service"
	"github.com/andreagrandi/mcp-wire/internal/target"
	"github.com/spf13/cobra"
)

var loadServices = service.LoadServices
var allTargets = target.AllTargets

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available services and targets",
	}

	listCmd.AddCommand(newListServicesCmd())
	listCmd.AddCommand(newListTargetsCmd())
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

func newListTargetsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "targets",
		Short: "List known targets and their install status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			printTargetsList(cmd.OutOrStdout(), allTargets())
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

func printTargetsList(output io.Writer, targets []target.Target) {
	fmt.Fprintln(output, "Targets:")
	fmt.Fprintln(output)

	if len(targets) == 0 {
		fmt.Fprintln(output, "  (none)")
		return
	}

	targetStatuses := make([]targetStatusRow, 0, len(targets))
	maxSlugWidth := 0
	maxNameWidth := 0

	for _, targetDefinition := range targets {
		slug := strings.TrimSpace(targetDefinition.Slug())
		name := strings.TrimSpace(targetDefinition.Name())
		installed := targetDefinition.IsInstalled()

		targetStatuses = append(targetStatuses, targetStatusRow{
			slug:      slug,
			name:      name,
			installed: installed,
		})

		if len(slug) > maxSlugWidth {
			maxSlugWidth = len(slug)
		}

		if len(name) > maxNameWidth {
			maxNameWidth = len(name)
		}
	}

	sort.Slice(targetStatuses, func(i int, j int) bool {
		return targetStatuses[i].slug < targetStatuses[j].slug
	})

	for _, targetStatus := range targetStatuses {
		status := "not found"
		if targetStatus.installed {
			status = "installed"
		}

		fmt.Fprintf(output, "  %-*s  %-*s  %s\n", maxSlugWidth, targetStatus.slug, maxNameWidth, targetStatus.name, status)
	}
}

type targetStatusRow struct {
	slug      string
	name      string
	installed bool
}
