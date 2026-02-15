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
var shouldAutoAuthenticate = func(cmd *cobra.Command) bool {
	return canUseInteractiveUI(cmd.InOrStdin(), cmd.OutOrStdout())
}

func init() {
	rootCmd.AddCommand(newInstallCmd())
}

func newInstallCmd() *cobra.Command {
	var targetSlugs []string
	var noPrompt bool
	var scopeValue string

	cmd := &cobra.Command{
		Use:   "install <service>",
		Short: "Install a service into one or more targets",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, err := parseInstallUninstallScope(scopeValue)
			if err != nil {
				return err
			}

			scopeSet := cmd.Flags().Changed("scope")

			if len(args) == 0 {
				return runInstallWizardWithScope(cmd, bufio.NewReader(cmd.InOrStdin()), targetSlugs, noPrompt, scope, scopeSet)
			}

			requestedServiceName := strings.TrimSpace(args[0])
			if requestedServiceName == "" {
				return errors.New("service name is required")
			}

			svc, err := resolveServiceByName(requestedServiceName)
			if err != nil {
				return err
			}

			targetDefinitions, err := resolveInstallTargets(targetSlugs)
			if err != nil {
				return err
			}

			return executeInstall(cmd, svc, targetDefinitions, noPrompt, scope)
		},
	}

	cmd.Flags().StringArrayVar(&targetSlugs, "target", nil, "Install to specific target slug(s); can be repeated")
	cmd.Flags().BoolVar(&noPrompt, "no-prompt", false, "Fail when required credentials are missing instead of prompting")
	cmd.Flags().StringVar(&scopeValue, "scope", string(target.ConfigScopeUser), "Config scope for supported targets: user or project")

	return cmd
}

func resolveServiceByName(name string) (service.Service, error) {
	services, err := loadServices()
	if err != nil {
		return service.Service{}, fmt.Errorf("load services: %w", err)
	}

	svc, err := findServiceDefinitionByName(services, name)
	if err == nil {
		return svc, nil
	}

	cfg, cfgErr := loadConfig()
	if cfgErr != nil || !cfg.IsFeatureEnabled("registry") {
		return service.Service{}, err
	}

	cat, catErr := loadCatalog("registry", true)
	if catErr != nil {
		return service.Service{}, err
	}

	entry, found := cat.Find(name)
	if !found {
		return service.Service{}, err
	}

	entry = refreshRegistryEntry(entry)

	resolved, ok := catalogEntryToService(entry)
	if !ok {
		return service.Service{}, fmt.Errorf("registry service %q has no supported install method", name)
	}

	return resolved, nil
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

func executeInstall(cmd *cobra.Command, svc service.Service, targetDefinitions []target.Target, noPrompt bool, scope target.ConfigScope) error {
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

	applyRegistrySubstitutions(&svc, resolvedEnv)

	printInstallPlan(cmd.OutOrStdout(), targetDefinitions)
	autoAuthenticate := shouldAutoAuthenticate(cmd) && serviceUsesOAuth(svc)

	installErrors := make([]error, 0)
	authenticationErrors := make([]error, 0)
	for _, targetDefinition := range targetDefinitions {
		var err error
		scopedTarget, supportsScopes := targetDefinition.(target.ScopedTarget)
		if supportsScopes && targetSupportsScope(targetDefinition, scope) {
			err = scopedTarget.InstallWithScope(svc, resolvedEnv, scope)
		} else {
			err = targetDefinition.Install(svc, resolvedEnv)
		}

		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s: failed (%v)\n", targetDefinition.Name(), err)
			installErrors = append(installErrors, fmt.Errorf("target %q: %w", targetDefinition.Slug(), err))
			continue
		}

		fmt.Fprintf(cmd.OutOrStdout(), "  %s: configured\n", targetDefinition.Name())

		if !autoAuthenticate {
			continue
		}

		authTarget, supportsAuth := targetDefinition.(target.AuthTarget)
		if !supportsAuth {
			manualAuthHint := oauthManualAuthHint(targetDefinition)
			if manualAuthHint != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  [!] Next step: %s\n", manualAuthHint)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s: authentication skipped (automatic OAuth is not supported by this target)\n", targetDefinition.Name())
			}

			continue
		}

		fmt.Fprintf(cmd.OutOrStdout(), "  %s: starting OAuth authentication...\n", targetDefinition.Name())
		err = authTarget.Authenticate(svc.Name, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s: authentication failed (%v)\n", targetDefinition.Name(), err)
			authenticationErrors = append(authenticationErrors, fmt.Errorf("target %q: %w", targetDefinition.Slug(), err))
			continue
		}

		fmt.Fprintf(cmd.OutOrStdout(), "  %s: authenticated\n", targetDefinition.Name())
	}

	if len(installErrors) > 0 {
		return fmt.Errorf("failed to install service %q on one or more targets: %w", svc.Name, errors.Join(installErrors...))
	}

	if len(authenticationErrors) > 0 {
		return fmt.Errorf("configured service %q but failed OAuth authentication on one or more targets: %w", svc.Name, errors.Join(authenticationErrors...))
	}

	return nil
}

func serviceUsesOAuth(svc service.Service) bool {
	authType := strings.ToLower(strings.TrimSpace(svc.Auth))
	if authType != "" {
		return authType == "oauth"
	}

	if strings.ToLower(strings.TrimSpace(svc.Transport)) != "sse" {
		return false
	}

	if strings.Contains(strings.ToLower(svc.Description), "oauth") {
		return true
	}

	url := strings.ToLower(strings.TrimSpace(svc.URL))
	return strings.Contains(url, "/mcp/oauth")
}

func oauthManualAuthHint(targetDefinition target.Target) string {
	targetSlug := strings.ToLower(strings.TrimSpace(targetDefinition.Slug()))

	switch targetSlug {
	case "claude":
		return "In Claude Code, run /mcp to complete OAuth authentication."
	default:
		return ""
	}
}

func applyRegistrySubstitutions(svc *service.Service, resolvedEnv map[string]string) {
	svc.URL = substituteVars(svc.URL, resolvedEnv)

	for name, tmpl := range svc.Headers {
		svc.Headers[name] = substituteVars(tmpl, resolvedEnv)
	}

	for i, arg := range svc.Args {
		svc.Args[i] = substituteVars(arg, resolvedEnv)
	}
}

func substituteVars(template string, values map[string]string) string {
	result := template
	for name, value := range values {
		placeholder := "{" + name + "}"
		if strings.Contains(result, placeholder) {
			result = strings.ReplaceAll(result, placeholder, value)
		}
	}

	return result
}
