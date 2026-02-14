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
	var scopeValue string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show service status across installed targets",
		RunE: func(cmd *cobra.Command, _ []string) error {
			scope, err := parseStatusScope(scopeValue)
			if err != nil {
				return err
			}

			return runStatusFlow(cmd.OutOrStdout(), scope)
		},
	}

	cmd.Flags().StringVar(&scopeValue, "scope", string(target.ConfigScopeEffective), "Status scope for supported targets: effective, user, or project")

	return cmd
}

func runStatusFlow(output io.Writer, scope target.ConfigScope) error {
	services, err := loadServices()
	if err != nil {
		return fmt.Errorf("load services: %w", err)
	}

	installedTargets := listInstalledTargets()
	if len(installedTargets) == 0 {
		printStatusNoTargets(output, scope)
		return nil
	}

	targetStatuses, err := listConfiguredServicesByTarget(installedTargets, scope)
	if err != nil {
		return err
	}

	serviceNames := make([]string, 0, len(services))
	for name := range services {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)

	if len(serviceNames) == 0 {
		printStatusNoServices(output, scope)
		return nil
	}

	printStatusMatrix(output, serviceNames, installedTargets, targetStatuses, scope)
	return nil
}

func listConfiguredServicesByTarget(targetDefinitions []target.Target, scope target.ConfigScope) (map[string]map[string]struct{}, error) {
	targetStatuses := make(map[string]map[string]struct{}, len(targetDefinitions))

	for _, targetDefinition := range targetDefinitions {
		var (
			configuredServices []string
			err                error
		)

		scopedTarget, supportsScopes := targetDefinition.(target.ScopedTarget)
		if supportsScopes && targetSupportsScope(targetDefinition, scope) {
			configuredServices, err = scopedTarget.ListWithScope(scope)
		} else {
			configuredServices, err = targetDefinition.List()
		}

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

func printStatusNoTargets(output io.Writer, scope target.ConfigScope) {
	fmt.Fprintf(output, "Status (%s scope):\n", scopeDescription(scope))
	fmt.Fprintln(output)
	fmt.Fprintln(output, "  (no installed targets found)")
}

func printStatusNoServices(output io.Writer, scope target.ConfigScope) {
	fmt.Fprintf(output, "Status (%s scope):\n", scopeDescription(scope))
	fmt.Fprintln(output)
	fmt.Fprintln(output, "  (no services found)")
}

func printStatusMatrix(
	output io.Writer,
	serviceNames []string,
	targetDefinitions []target.Target,
	targetStatuses map[string]map[string]struct{},
	scope target.ConfigScope,
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

	fmt.Fprintf(output, "Status (%s scope):\n", scopeDescription(scope))
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
