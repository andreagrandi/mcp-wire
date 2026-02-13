package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/andreagrandi/mcp-wire/internal/credential"
	"github.com/andreagrandi/mcp-wire/internal/service"
	"github.com/andreagrandi/mcp-wire/internal/target"
	"github.com/spf13/cobra"
)

var listInstalledTargets = target.InstalledTargets
var lookupTarget = target.FindTarget
var newCredentialEnvSource = func() credential.Source { return credential.NewEnvSource() }
var newCredentialFileSource = func(path string) credential.Source { return credential.NewFileSource(path) }
var newCredentialResolver = func(sources ...credential.Source) *credential.Resolver {
	return credential.NewResolver(sources...)
}

func init() {
	rootCmd.AddCommand(newInstallCmd())
}

func newInstallCmd() *cobra.Command {
	var targetSlugs []string
	var noPrompt bool

	cmd := &cobra.Command{
		Use:   "install <service>",
		Short: "Install a service into one or more targets",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return runInstallWizard(cmd, bufio.NewReader(cmd.InOrStdin()), targetSlugs, noPrompt)
			}

			requestedServiceName := strings.TrimSpace(args[0])
			if requestedServiceName == "" {
				return errors.New("service name is required")
			}

			services, err := loadServices()
			if err != nil {
				return fmt.Errorf("load services: %w", err)
			}

			svc, err := findServiceDefinitionByName(services, requestedServiceName)
			if err != nil {
				return err
			}

			targetDefinitions, err := resolveInstallTargets(targetSlugs)
			if err != nil {
				return err
			}

			return executeInstall(cmd, svc, targetDefinitions, noPrompt)
		},
	}

	cmd.Flags().StringArrayVar(&targetSlugs, "target", nil, "Install to specific target slug(s); can be repeated")
	cmd.Flags().BoolVar(&noPrompt, "no-prompt", false, "Fail when required credentials are missing instead of prompting")

	return cmd
}

func findServiceDefinitionByName(services map[string]service.Service, name string) (service.Service, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return service.Service{}, errors.New("service name is required")
	}

	if svc, found := services[trimmedName]; found {
		return svc, nil
	}

	for key, svc := range services {
		if strings.EqualFold(key, trimmedName) {
			return svc, nil
		}
	}

	availableServiceNames := make([]string, 0, len(services))
	for key := range services {
		availableServiceNames = append(availableServiceNames, key)
	}
	sort.Strings(availableServiceNames)

	if len(availableServiceNames) == 0 {
		return service.Service{}, fmt.Errorf("service %q not found (no service definitions available)", trimmedName)
	}

	return service.Service{}, fmt.Errorf("service %q not found (available: %s)", trimmedName, strings.Join(availableServiceNames, ", "))
}

func resolveInstallTargets(targetSlugs []string) ([]target.Target, error) {
	normalizedTargetSlugs := make([]string, 0, len(targetSlugs))
	for _, rawSlug := range targetSlugs {
		slug := strings.ToLower(strings.TrimSpace(rawSlug))
		if slug == "" {
			continue
		}

		normalizedTargetSlugs = append(normalizedTargetSlugs, slug)
	}

	if len(normalizedTargetSlugs) == 0 {
		targetDefinitions := listInstalledTargets()
		if len(targetDefinitions) == 0 {
			return nil, errors.New("no installed targets found")
		}

		return targetDefinitions, nil
	}

	targetDefinitions := make([]target.Target, 0, len(normalizedTargetSlugs))
	seenTargets := make(map[string]struct{})

	for _, slug := range normalizedTargetSlugs {
		if _, seen := seenTargets[slug]; seen {
			continue
		}

		targetDefinition, found := lookupTarget(slug)
		if !found {
			return nil, fmt.Errorf("target %q is not known", slug)
		}

		if !targetDefinition.IsInstalled() {
			return nil, fmt.Errorf("target %q is not installed", slug)
		}

		targetDefinitions = append(targetDefinitions, targetDefinition)
		seenTargets[slug] = struct{}{}
	}

	if len(targetDefinitions) == 0 {
		return nil, errors.New("no targets selected")
	}

	return targetDefinitions, nil
}

func printInstallPlan(output io.Writer, targetDefinitions []target.Target) {
	names := make([]string, 0, len(targetDefinitions))
	for _, targetDefinition := range targetDefinitions {
		names = append(names, targetDefinition.Name())
	}

	fmt.Fprintf(output, "Installing to: %s\n", strings.Join(names, ", "))
}

func executeInstall(cmd *cobra.Command, svc service.Service, targetDefinitions []target.Target, noPrompt bool) error {
	envSource := newCredentialEnvSource()
	fileSource := newCredentialFileSource("")
	resolver := newCredentialResolver(envSource, fileSource)

	resolvedEnv, err := resolveServiceCredentials(svc, resolver, interactiveCredentialOptions{
		noPrompt:   noPrompt,
		input:      cmd.InOrStdin(),
		output:     cmd.OutOrStdout(),
		fileSource: fileSource,
	})
	if err != nil {
		return err
	}

	printInstallPlan(cmd.OutOrStdout(), targetDefinitions)

	installErrors := make([]error, 0)
	for _, targetDefinition := range targetDefinitions {
		err := targetDefinition.Install(svc, resolvedEnv)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s: failed (%v)\n", targetDefinition.Name(), err)
			installErrors = append(installErrors, fmt.Errorf("target %q: %w", targetDefinition.Slug(), err))
			continue
		}

		fmt.Fprintf(cmd.OutOrStdout(), "  %s: configured\n", targetDefinition.Name())
	}

	if len(installErrors) > 0 {
		return fmt.Errorf("failed to install service %q on one or more targets: %w", svc.Name, errors.Join(installErrors...))
	}

	return nil
}
