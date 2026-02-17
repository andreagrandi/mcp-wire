package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/andreagrandi/mcp-wire/internal/app"
	"github.com/andreagrandi/mcp-wire/internal/catalog"
	"github.com/andreagrandi/mcp-wire/internal/config"
	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
	"github.com/andreagrandi/mcp-wire/internal/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mcp-wire",
	Short: "Install and configure MCP servers across AI coding tools",
	Long: `mcp-wire is a CLI tool that lets you install and configure MCP (Model Context Protocol)
servers across multiple AI coding CLI tools (Claude Code, Codex, Gemini CLI, etc.)
from a single interface.

Services are defined as YAML files -- no code needed to add one.
Targets are the AI tools where services get installed.`,
	Version: app.Version,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runGuidedMainMenu(cmd)
	},
}

func Execute() error {
	if !isCacheCommand(os.Args) {
		maybeStartRegistryBackgroundSync()
	}

	return rootCmd.Execute()
}

func isCacheCommand(args []string) bool {
	if len(args) < 2 {
		return false
	}

	return args[1] == "cache"
}

func runGuidedMainMenu(cmd *cobra.Command) error {
	if canUseInteractiveUI(cmd.InOrStdin(), cmd.OutOrStdout()) {
		cfg, err := loadConfig()
		if err == nil && cfg.IsFeatureEnabled("tui") {
			return tui.Run(tuiCallbacks(cfg), app.Version)
		}

		return runGuidedMainMenuSurvey(cmd)
	}

	return runGuidedMainMenuPlain(cmd)
}

func runGuidedMainMenuPlain(cmd *cobra.Command) error {
	reader := bufio.NewReader(cmd.InOrStdin())
	output := cmd.OutOrStdout()

	for {
		fmt.Fprintln(output, "Main Menu")
		fmt.Fprintln(output, "  1) Install service")
		fmt.Fprintln(output, "  2) Uninstall service")
		fmt.Fprintln(output, "  3) Status")
		fmt.Fprintln(output, "  4) List services")
		fmt.Fprintln(output, "  5) List targets")
		fmt.Fprintln(output, "  6) Exit")

		choice, err := readTrimmedLine(reader, output, "Option [1-6]: ")
		if err != nil {
			return fmt.Errorf("read menu option: %w", err)
		}

		switch strings.ToLower(choice) {
		case "1", "install":
			fmt.Fprintln(output)
			if err := runInstallWizard(cmd, reader, nil, false); err != nil {
				return err
			}
			fmt.Fprintln(output)
		case "2", "uninstall":
			fmt.Fprintln(output)
			if err := runUninstallWizard(cmd, reader, nil); err != nil {
				return err
			}
			fmt.Fprintln(output)
		case "3", "status":
			fmt.Fprintln(output)
			if err := runStatusFlow(output, targetpkg.ConfigScopeEffective); err != nil {
				return err
			}
			fmt.Fprintln(output)
		case "4", "services":
			fmt.Fprintln(output)
			services, err := loadServices()
			if err != nil {
				return fmt.Errorf("load services: %w", err)
			}
			printServicesList(output, services)
			fmt.Fprintln(output)
		case "5", "targets":
			fmt.Fprintln(output)
			printTargetsList(output, allTargets())
			fmt.Fprintln(output)
		case "6", "exit", "q", "quit":
			fmt.Fprintln(output, "Goodbye.")
			return nil
		default:
			fmt.Fprintf(output, "Invalid option %q. Enter 1-6.\n\n", choice)
		}
	}
}

func tuiCallbacks(cfg *config.Config) tui.Callbacks {
	registryEnabled := cfg.IsFeatureEnabled("registry")
	return tui.Callbacks{
		RenderStatus: func(w io.Writer) error {
			return runStatusFlow(w, targetpkg.ConfigScopeEffective)
		},
		RenderServicesList: func(w io.Writer) error {
			services, err := loadServices()
			if err != nil {
				return fmt.Errorf("load services: %w", err)
			}

			printServicesList(w, services)
			return nil
		},
		RenderTargetsList: func(w io.Writer) error {
			printTargetsList(w, allTargets())
			return nil
		},
		LoadCatalog: func(source string) (*catalog.Catalog, error) {
			return loadCatalog(source, registryEnabled)
		},
		RegistrySyncStatus: func() string {
			return registrySyncStatusLine(registryEnabled)
		},
		RefreshRegistryEntry:  refreshRegistryEntry,
		CatalogEntryToService: catalogEntryToService,
		AllTargets:            allTargets,
		RegistryEnabled:       registryEnabled,
	}
}

func readTrimmedLine(reader *bufio.Reader, output io.Writer, prompt string) (string, error) {
	fmt.Fprint(output, prompt)
	line, err := reader.ReadString('\n')
	if err != nil {
		if len(strings.TrimSpace(line)) == 0 {
			return "", err
		}
	}

	return strings.TrimSpace(line), nil
}
