package cli

import (
	"fmt"
	"io"

	"github.com/andreagrandi/mcp-wire/internal/config"
	"github.com/spf13/cobra"
)

var loadConfig = config.Load

func init() {
	featureCmd := &cobra.Command{
		Use:   "feature",
		Short: "Manage feature flags",
	}

	featureCmd.AddCommand(newFeatureEnableCmd())
	featureCmd.AddCommand(newFeatureDisableCmd())
	featureCmd.AddCommand(newFeatureListCmd())
	rootCmd.AddCommand(featureCmd)
}

func newFeatureEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable <feature>",
		Short: "Enable a feature flag",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setFeatureFlag(cmd.OutOrStdout(), args[0], true)
		},
	}
}

func newFeatureDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <feature>",
		Short: "Disable a feature flag",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setFeatureFlag(cmd.OutOrStdout(), args[0], false)
		},
	}
}

func newFeatureListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all feature flags and their status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return listFeatures(cmd.OutOrStdout())
		},
	}
}

func setFeatureFlag(output io.Writer, name string, enabled bool) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := cfg.SetFeature(name, enabled); err != nil {
		return err
	}

	action := "disabled"
	if enabled {
		action = "enabled"
	}

	fmt.Fprintf(output, "Feature %q %s.\n", name, action)

	return nil
}

func listFeatures(output io.Writer) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	features := cfg.Features()
	if len(features) == 0 {
		fmt.Fprintln(output, "No feature flags available.")
		return nil
	}

	fmt.Fprintln(output, "Feature flags:")
	fmt.Fprintln(output)

	maxNameWidth := 0
	for _, f := range features {
		if len(f.Name) > maxNameWidth {
			maxNameWidth = len(f.Name)
		}
	}

	for _, f := range features {
		status := "disabled"
		if f.Enabled {
			status = "enabled"
		}

		fmt.Fprintf(output, "  %-*s  %-8s  %s\n", maxNameWidth, f.Name, status, f.Description)
	}

	return nil
}
