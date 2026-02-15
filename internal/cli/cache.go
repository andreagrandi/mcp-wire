package cli

import (
	"fmt"

	"github.com/andreagrandi/mcp-wire/internal/registry"
	"github.com/spf13/cobra"
)

var clearRegistryCache = registry.ClearDefaultCache

func init() {
	cacheCmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage local cache data",
	}

	cacheCmd.AddCommand(newCacheClearCmd())
	rootCmd.AddCommand(cacheCmd)
}

func newCacheClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Clear local registry cache",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, removed, err := clearRegistryCache()
			if err != nil {
				return err
			}

			if removed {
				fmt.Fprintf(cmd.OutOrStdout(), "Registry cache cleared: %s\n", path)
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Registry cache already empty: %s\n", path)
			return nil
		},
	}
}
