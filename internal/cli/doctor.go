package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/andreagrandi/mcp-wire/internal/app"
	"github.com/andreagrandi/mcp-wire/internal/config"
	"github.com/andreagrandi/mcp-wire/internal/credential"
	"github.com/andreagrandi/mcp-wire/internal/registry"
	"github.com/andreagrandi/mcp-wire/internal/target"
	"github.com/spf13/cobra"
)

// doctorPathProbe describes a filesystem path that doctor reports on.
type doctorPathProbe struct {
	label string
	path  string
}

// doctorDeps wires the data sources doctor reads, so tests can substitute them.
type doctorDeps struct {
	loadConfig        func() (*config.Config, error)
	allTargets        func() []target.Target
	registryCachePath func() string
	credentialsPath   func() string
	userServicesPath  func() string
	version           string
	stat              func(name string) (os.FileInfo, error)
}

func defaultDoctorDeps() doctorDeps {
	return doctorDeps{
		loadConfig:        loadConfig,
		allTargets:        allTargets,
		registryCachePath: registry.DefaultCachePath,
		credentialsPath:   defaultCredentialsFilePath,
		userServicesPath:  defaultUserServicesPath,
		version:           app.Version,
		stat:              os.Stat,
	}
}

func init() {
	rootCmd.AddCommand(newDoctorCmd())
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Print read-only diagnostic information",
		Long: `doctor prints detected targets, config paths, feature flag state,
and likely setup problems.

It is read-only: it never writes to target config files or credentials.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDoctor(cmd.OutOrStdout(), defaultDoctorDeps())
		},
	}
}

func runDoctor(output io.Writer, deps doctorDeps) error {
	fmt.Fprintf(output, "mcp-wire %s\n\n", deps.version)

	writeDoctorTargets(output, deps)
	writeDoctorFeatures(output, deps)
	writeDoctorPaths(output, deps)
	writeDoctorHints(output, deps)

	return nil
}

func writeDoctorTargets(output io.Writer, deps doctorDeps) {
	fmt.Fprintln(output, "Targets:")

	targets := deps.allTargets()
	if len(targets) == 0 {
		fmt.Fprintln(output, "  (no targets registered)")
		fmt.Fprintln(output)
		return
	}

	for _, t := range targets {
		installed := "no"
		if t.IsInstalled() {
			installed = "yes"
		}

		fmt.Fprintf(output, "  %s (%s)\n", t.Name(), t.Slug())
		fmt.Fprintf(output, "    Installed:    %s\n", installed)

		configPath, hasConfigPath := targetConfigPath(t)
		if !hasConfigPath {
			fmt.Fprintln(output, "    Config path:  (not reported)")
			continue
		}

		fmt.Fprintf(output, "    Config path:  %s\n", configPath)
		fmt.Fprintf(output, "    Config:       %s\n", describePathStatus(configPath, deps.stat))
	}

	fmt.Fprintln(output)
}

func writeDoctorFeatures(output io.Writer, deps doctorDeps) {
	fmt.Fprintln(output, "Feature flags:")

	cfg, err := deps.loadConfig()
	if err != nil {
		fmt.Fprintf(output, "  (failed to load config: %v)\n\n", err)
		return
	}

	features := cfg.Features()
	if len(features) == 0 {
		fmt.Fprintln(output, "  (none registered)")
		fmt.Fprintln(output)
		return
	}

	maxNameWidth := 0
	for _, feature := range features {
		if len(feature.Name) > maxNameWidth {
			maxNameWidth = len(feature.Name)
		}
	}

	for _, feature := range features {
		state := "disabled"
		if feature.Enabled {
			state = "enabled"
		}

		fmt.Fprintf(output, "  %-*s  %-8s  %s\n", maxNameWidth, feature.Name, state, feature.Description)
	}

	fmt.Fprintln(output)
}

func writeDoctorPaths(output io.Writer, deps doctorDeps) {
	fmt.Fprintln(output, "Paths:")

	probes := []doctorPathProbe{
		{label: "mcp-wire config", path: defaultMCPWireConfigPath()},
		{label: "Credentials file", path: deps.credentialsPath()},
		{label: "User services dir", path: deps.userServicesPath()},
		{label: "Registry cache", path: deps.registryCachePath()},
	}

	maxLabelWidth := 0
	for _, probe := range probes {
		if len(probe.label) > maxLabelWidth {
			maxLabelWidth = len(probe.label)
		}
	}

	for _, probe := range probes {
		fmt.Fprintf(output, "  %-*s  %s  (%s)\n",
			maxLabelWidth, probe.label, probe.path, describePathStatus(probe.path, deps.stat))
	}

	fmt.Fprintln(output)
}

func writeDoctorHints(output io.Writer, deps doctorDeps) {
	hints := buildDoctorHints(deps)
	if len(hints) == 0 {
		return
	}

	fmt.Fprintln(output, "Hints:")
	for _, hint := range hints {
		fmt.Fprintf(output, "  - %s\n", hint)
	}

	fmt.Fprintln(output)
}

func buildDoctorHints(deps doctorDeps) []string {
	var hints []string

	for _, t := range deps.allTargets() {
		if t.IsInstalled() {
			continue
		}

		hints = append(hints, fmt.Sprintf(
			"%s (%s) is not detected on this system. Install it to enable installs into this target.",
			t.Name(), t.Slug()))
	}

	cfg, err := deps.loadConfig()
	if err == nil && !cfg.IsFeatureEnabled("registry") {
		hints = append(hints, "Registry feature is disabled. Enable with `mcp-wire feature enable registry` to install services from the MCP Registry.")
	}

	return hints
}

func targetConfigPath(t target.Target) (string, bool) {
	provider, ok := t.(target.ConfigPathProvider)
	if !ok {
		return "", false
	}

	return strings.TrimSpace(provider.ConfigPath()), true
}

func describePathStatus(path string, stat func(name string) (os.FileInfo, error)) string {
	if strings.TrimSpace(path) == "" {
		return "unknown"
	}

	if stat == nil {
		stat = os.Stat
	}

	info, err := stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "missing"
		}

		return fmt.Sprintf("error: %v", err)
	}

	if info.IsDir() {
		return "exists (directory)"
	}

	return "exists"
}

func defaultMCPWireConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "mcp-wire", "config.json")
	}

	return filepath.Join(homeDir, ".config", "mcp-wire", "config.json")
}

func defaultCredentialsFilePath() string {
	source := credential.NewFileSource("")
	return source.Path()
}

func defaultUserServicesPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "mcp-wire", "services")
	}

	return filepath.Join(homeDir, ".config", "mcp-wire", "services")
}
