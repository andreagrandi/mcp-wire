package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/andreagrandi/mcp-wire/internal/catalog"
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
	cmd := &cobra.Command{
		Use:   "services",
		Short: "List available service definitions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			source, _ := cmd.Flags().GetString("source")

			if err := validateSource(source); err != nil {
				return err
			}

			cfg, err := loadConfig()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			registryEnabled := cfg.IsFeatureEnabled("registry")

			if !registryEnabled && source != "curated" {
				return fmt.Errorf("--source requires the registry feature (enable with: mcp-wire feature enable registry)")
			}

			if registryEnabled && source != "curated" {
				cat, err := loadCatalog(source, true)
				if err != nil {
					return err
				}

				var entries []catalog.Entry
				switch source {
				case "registry":
					entries = cat.BySource(catalog.SourceRegistry)
				default:
					entries = cat.All()
				}

				printCatalogEntries(cmd.OutOrStdout(), entries, source == "all")
				if statusLine := registrySyncStatusLine(registryEnabled); statusLine != "" {
					fmt.Fprintln(cmd.OutOrStdout())
					fmt.Fprintln(cmd.OutOrStdout(), statusLine)
				}
				return nil
			}

			services, err := loadServices()
			if err != nil {
				return fmt.Errorf("load services: %w", err)
			}

			printServicesList(cmd.OutOrStdout(), services)
			return nil
		},
	}

	cmd.Flags().String("source", "curated", "filter by source: curated, registry, or all")
	cmd.Flag("source").Hidden = true

	parentHelp := cmd.HelpFunc()
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err == nil && cfg.IsFeatureEnabled("registry") {
			c.Flag("source").Hidden = false
		}

		parentHelp(c, args)
	})

	return cmd
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
