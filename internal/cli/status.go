package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/andreagrandi/mcp-wire/internal/target"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newStatusCmd())
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show service status across installed targets",
		RunE: func(cmd *cobra.Command, _ []string) error {
			services, err := loadServices()
			if err != nil {
				return fmt.Errorf("load services: %w", err)
			}

			installedTargets := listInstalledTargets()
			if len(installedTargets) == 0 {
				printStatusNoTargets(cmd.OutOrStdout())
				return nil
			}

			targetStatuses, err := listConfiguredServicesByTarget(installedTargets)
			if err != nil {
				return err
			}

			serviceNames := make([]string, 0, len(services))
			for name := range services {
				serviceNames = append(serviceNames, name)
			}
			sort.Strings(serviceNames)

			if len(serviceNames) == 0 {
				printStatusNoServices(cmd.OutOrStdout())
				return nil
			}

			printStatusMatrix(cmd.OutOrStdout(), serviceNames, installedTargets, targetStatuses)
			return nil
		},
	}
}

func listConfiguredServicesByTarget(targetDefinitions []target.Target) (map[string]map[string]struct{}, error) {
	targetStatuses := make(map[string]map[string]struct{}, len(targetDefinitions))

	for _, targetDefinition := range targetDefinitions {
		configuredServices, err := targetDefinition.List()
		if err != nil {
			return nil, fmt.Errorf("list configured services for target %q: %w", targetDefinition.Slug(), err)
		}

		configuredSet := make(map[string]struct{}, len(configuredServices))
		for _, serviceName := range configuredServices {
			trimmedName := strings.TrimSpace(serviceName)
			if trimmedName == "" {
				continue
			}

			configuredSet[trimmedName] = struct{}{}
		}

		targetStatuses[targetDefinition.Slug()] = configuredSet
	}

	return targetStatuses, nil
}

func printStatusNoTargets(output io.Writer) {
	fmt.Fprintln(output, "Status:")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "  (no installed targets found)")
}

func printStatusNoServices(output io.Writer) {
	fmt.Fprintln(output, "Status:")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "  (no services found)")
}

func printStatusMatrix(
	output io.Writer,
	serviceNames []string,
	targetDefinitions []target.Target,
	targetStatuses map[string]map[string]struct{},
) {
	sortedTargets := make([]target.Target, len(targetDefinitions))
	copy(sortedTargets, targetDefinitions)
	sort.Slice(sortedTargets, func(i int, j int) bool {
		return sortedTargets[i].Slug() < sortedTargets[j].Slug()
	})

	serviceColumnWidth := len("service")
	for _, serviceName := range serviceNames {
		if len(serviceName) > serviceColumnWidth {
			serviceColumnWidth = len(serviceName)
		}
	}

	targetColumnWidths := make([]int, len(sortedTargets))
	for i, targetDefinition := range sortedTargets {
		columnWidth := len(targetDefinition.Name())
		if columnWidth < len("yes") {
			columnWidth = len("yes")
		}

		targetColumnWidths[i] = columnWidth
	}

	fmt.Fprintln(output, "Status:")
	fmt.Fprintln(output)

	fmt.Fprintf(output, "  %-*s", serviceColumnWidth, "service")
	for i, targetDefinition := range sortedTargets {
		fmt.Fprintf(output, "  %-*s", targetColumnWidths[i], targetDefinition.Name())
	}
	fmt.Fprintln(output)

	for _, serviceName := range serviceNames {
		fmt.Fprintf(output, "  %-*s", serviceColumnWidth, serviceName)

		for i, targetDefinition := range sortedTargets {
			statusValue := "no"
			if _, configured := targetStatuses[targetDefinition.Slug()][serviceName]; configured {
				statusValue = "yes"
			}

			fmt.Fprintf(output, "  %-*s", targetColumnWidths[i], statusValue)
		}

		fmt.Fprintln(output)
	}
}
