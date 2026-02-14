package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/andreagrandi/mcp-wire/internal/credential"
	"github.com/andreagrandi/mcp-wire/internal/service"
	"github.com/andreagrandi/mcp-wire/internal/target"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var newCredentialFileSourceForCleanup = credential.NewFileSource
var isTerminalReader = func(reader io.Reader) bool {
	file, ok := reader.(*os.File)
	if !ok {
		return false
	}

	return term.IsTerminal(int(file.Fd()))
}

func init() {
	rootCmd.AddCommand(newUninstallCmd())
}

func newUninstallCmd() *cobra.Command {
	var targetSlugs []string
	var scopeValue string

	cmd := &cobra.Command{
		Use:   "uninstall <service>",
		Short: "Remove a service from one or more targets",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, err := parseInstallUninstallScope(scopeValue)
			if err != nil {
				return err
			}

			scopeSet := cmd.Flags().Changed("scope")

			if len(args) == 0 {
				return runUninstallWizardWithScope(cmd, bufio.NewReader(cmd.InOrStdin()), targetSlugs, scope, scopeSet)
			}

			serviceName := strings.TrimSpace(args[0])
			if serviceName == "" {
				return errors.New("service name is required")
			}

			targetDefinitions, err := resolveInstallTargets(targetSlugs)
			if err != nil {
				return err
			}

			printUninstallPlan(cmd.OutOrStdout(), targetDefinitions)

			uninstallErrors := make([]error, 0)
			for _, targetDefinition := range targetDefinitions {
				var err error
				scopedTarget, supportsScopes := targetDefinition.(target.ScopedTarget)
				if supportsScopes && targetSupportsScope(targetDefinition, scope) {
					err = scopedTarget.UninstallWithScope(serviceName, scope)
				} else {
					err = targetDefinition.Uninstall(serviceName)
				}

				if err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s: failed (%v)\n", targetDefinition.Name(), err)
					uninstallErrors = append(uninstallErrors, fmt.Errorf("target %q: %w", targetDefinition.Slug(), err))
					continue
				}

				fmt.Fprintf(cmd.OutOrStdout(), "  %s: removed\n", targetDefinition.Name())
			}

			if len(uninstallErrors) > 0 {
				return fmt.Errorf("failed to uninstall service %q from one or more targets: %w", serviceName, errors.Join(uninstallErrors...))
			}

			return maybeRemoveStoredCredentials(cmd, serviceName)
		},
	}

	cmd.Flags().StringArrayVar(&targetSlugs, "target", nil, "Uninstall from specific target slug(s); can be repeated")
	cmd.Flags().StringVar(&scopeValue, "scope", string(target.ConfigScopeUser), "Config scope for supported targets: user or project")

	return cmd
}

func printUninstallPlan(output io.Writer, targetDefinitions []target.Target) {
	names := make([]string, 0, len(targetDefinitions))
	for _, targetDefinition := range targetDefinitions {
		names = append(names, targetDefinition.Name())
	}

	fmt.Fprintf(output, "Uninstalling from: %s\n", strings.Join(names, ", "))
}

func maybeRemoveStoredCredentials(cmd *cobra.Command, serviceName string) error {
	input := cmd.InOrStdin()
	if !isTerminalReader(input) {
		return nil
	}

	services, err := loadServices()
	if err != nil {
		return nil
	}

	serviceDefinition, err := findServiceDefinitionByName(services, serviceName)
	if err != nil {
		return nil
	}

	envNames := serviceEnvNames(serviceDefinition)
	if len(envNames) == 0 {
		return nil
	}

	reader := bufio.NewReader(input)
	shouldRemove, err := askYesNo(reader, cmd.OutOrStdout(), "\nRemove stored credentials for this service? [y/N]: ", false)
	if err != nil {
		return fmt.Errorf("read credential removal confirmation: %w", err)
	}

	if !shouldRemove {
		return nil
	}

	fileSource := newCredentialFileSourceForCleanup("")
	removedCount, err := removeStoredCredentials(fileSource, envNames)
	if err != nil {
		return fmt.Errorf("remove stored credentials: %w", err)
	}

	if removedCount == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No stored credentials found.")
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Stored credentials removed.")
	return nil
}

func serviceEnvNames(serviceDefinition service.Service) []string {
	seen := make(map[string]struct{})
	envNames := make([]string, 0, len(serviceDefinition.Env))

	for _, envVar := range serviceDefinition.Env {
		envName := strings.TrimSpace(envVar.Name)
		if envName == "" {
			continue
		}

		if _, exists := seen[envName]; exists {
			continue
		}

		seen[envName] = struct{}{}
		envNames = append(envNames, envName)
	}

	return envNames
}

func removeStoredCredentials(fileSource *credential.FileSource, envNames []string) (int, error) {
	if fileSource == nil {
		return 0, errors.New("credential file source is nil")
	}

	if len(envNames) == 0 {
		return 0, nil
	}

	matchedEnvNames := make([]string, 0, len(envNames))
	for _, envName := range envNames {
		if _, found := fileSource.Get(envName); !found {
			continue
		}

		matchedEnvNames = append(matchedEnvNames, envName)
	}

	if len(matchedEnvNames) == 0 {
		return 0, nil
	}

	if err := fileSource.DeleteMany(matchedEnvNames...); err != nil {
		return 0, err
	}

	return len(matchedEnvNames), nil
}
