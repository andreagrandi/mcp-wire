package cli

import (
	"bufio"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/andreagrandi/mcp-wire/internal/service"
	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
	"github.com/spf13/cobra"
)

func runInstallWizard(cmd *cobra.Command, reader *bufio.Reader, targetSlugs []string, noPrompt bool) error {
	if canUseInteractiveUI(cmd.InOrStdin(), cmd.OutOrStdout()) {
		return runInstallWizardSurvey(cmd, targetSlugs, noPrompt, targetpkg.ConfigScopeUser, false)
	}

	return runInstallWizardPlain(cmd, reader, targetSlugs, noPrompt, targetpkg.ConfigScopeUser, false)
}

func runInstallWizardWithScope(
	cmd *cobra.Command,
	reader *bufio.Reader,
	targetSlugs []string,
	noPrompt bool,
	scope targetpkg.ConfigScope,
	scopeSet bool,
) error {
	if canUseInteractiveUI(cmd.InOrStdin(), cmd.OutOrStdout()) {
		return runInstallWizardSurvey(cmd, targetSlugs, noPrompt, scope, scopeSet)
	}

	return runInstallWizardPlain(cmd, reader, targetSlugs, noPrompt, scope, scopeSet)
}

func runInstallWizardPlain(
	cmd *cobra.Command,
	reader *bufio.Reader,
	targetSlugs []string,
	noPrompt bool,
	requestedScope targetpkg.ConfigScope,
	scopeSet bool,
) error {
	output := cmd.OutOrStdout()
	fmt.Fprintln(output, "Install Wizard")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Step 1/4: Service")

	services, err := loadServices()
	if err != nil {
		return fmt.Errorf("load services: %w", err)
	}

	svc, err := pickServiceInteractive(output, reader, services)
	if err != nil {
		return err
	}

	fmt.Fprintln(output)
	fmt.Fprintln(output, "Step 2/4: Targets")

	targetDefinitions, err := resolveTargetsForWizard(output, reader, targetSlugs)
	if err != nil {
		return err
	}

	selectedScope, err := resolveScopeForPlainWizard(output, reader, targetDefinitions, requestedScope, scopeSet, "Install")
	if err != nil {
		return err
	}

	confirmed, err := confirmInstallSelection(output, reader, svc, targetDefinitions, noPrompt, selectedScope)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(output, "Install cancelled.")
		return nil
	}

	fmt.Fprintln(output)
	fmt.Fprintln(output, "Step 4/4: Apply")

	if err := executeInstall(cmd, svc, targetDefinitions, noPrompt, selectedScope); err != nil {
		return err
	}

	printEquivalentCommand(output, buildEquivalentInstallCommand(svc.Name, targetDefinitions, selectedScope))
	return nil
}

func runUninstallWizard(cmd *cobra.Command, reader *bufio.Reader, targetSlugs []string) error {
	if canUseInteractiveUI(cmd.InOrStdin(), cmd.OutOrStdout()) {
		return runUninstallWizardSurvey(cmd, targetSlugs, targetpkg.ConfigScopeUser, false)
	}

	return runUninstallWizardPlain(cmd, reader, targetSlugs, targetpkg.ConfigScopeUser, false)
}

func runUninstallWizardWithScope(
	cmd *cobra.Command,
	reader *bufio.Reader,
	targetSlugs []string,
	scope targetpkg.ConfigScope,
	scopeSet bool,
) error {
	if canUseInteractiveUI(cmd.InOrStdin(), cmd.OutOrStdout()) {
		return runUninstallWizardSurvey(cmd, targetSlugs, scope, scopeSet)
	}

	return runUninstallWizardPlain(cmd, reader, targetSlugs, scope, scopeSet)
}

func runUninstallWizardPlain(
	cmd *cobra.Command,
	reader *bufio.Reader,
	targetSlugs []string,
	requestedScope targetpkg.ConfigScope,
	scopeSet bool,
) error {
	output := cmd.OutOrStdout()
	fmt.Fprintln(output, "Uninstall Wizard")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Step 1/4: Service")

	services, err := loadServices()
	if err != nil {
		return fmt.Errorf("load services: %w", err)
	}

	svc, err := pickServiceInteractive(output, reader, services)
	if err != nil {
		return err
	}

	fmt.Fprintln(output)
	fmt.Fprintln(output, "Step 2/4: Targets")

	targetDefinitions, err := resolveTargetsForWizard(output, reader, targetSlugs)
	if err != nil {
		return err
	}

	selectedScope, err := resolveScopeForPlainWizard(output, reader, targetDefinitions, requestedScope, scopeSet, "Uninstall")
	if err != nil {
		return err
	}

	confirmed, err := confirmUninstallSelection(output, reader, svc, targetDefinitions, selectedScope)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(output, "Uninstall cancelled.")
		return nil
	}

	fmt.Fprintln(output)
	fmt.Fprintln(output, "Step 4/4: Apply")

	printUninstallPlan(output, targetDefinitions)

	uninstallErrors := make([]error, 0)
	for _, targetDefinition := range targetDefinitions {
		var err error
		scopedTarget, supportsScopes := targetDefinition.(targetpkg.ScopedTarget)
		if supportsScopes && targetSupportsScope(targetDefinition, selectedScope) {
			err = scopedTarget.UninstallWithScope(svc.Name, selectedScope)
		} else {
			err = targetDefinition.Uninstall(svc.Name)
		}

		if err != nil {
			fmt.Fprintf(output, "  %s: failed (%v)\n", targetDefinition.Name(), err)
			uninstallErrors = append(uninstallErrors, fmt.Errorf("target %q: %w", targetDefinition.Slug(), err))
			continue
		}

		fmt.Fprintf(output, "  %s: removed\n", targetDefinition.Name())
	}

	if len(uninstallErrors) > 0 {
		return fmt.Errorf("failed to uninstall service %q from one or more targets: %w", svc.Name, errors.Join(uninstallErrors...))
	}

	if err := maybeRemoveStoredCredentials(cmd, svc.Name); err != nil {
		return err
	}

	printEquivalentCommand(output, buildEquivalentUninstallCommand(svc.Name, targetDefinitions, selectedScope))
	return nil
}

func resolveTargetsForWizard(output ioWriter, reader *bufio.Reader, targetSlugs []string) ([]targetpkg.Target, error) {
	if len(targetSlugs) > 0 {
		return resolveInstallTargets(targetSlugs)
	}

	return pickTargetsInteractive(output, reader)
}

type ioWriter interface {
	Write(p []byte) (n int, err error)
}

func printEquivalentCommand(output ioWriter, command string) {
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Equivalent command:")
	fmt.Fprintf(output, "  %s\n", command)
	fmt.Fprintln(output)
}

func pickServiceInteractive(output ioWriter, reader *bufio.Reader, services map[string]service.Service) (service.Service, error) {
	if len(services) == 0 {
		return service.Service{}, errors.New("no service definitions available")
	}

	rows := make([]service.Service, 0, len(services))
	for _, svc := range services {
		rows = append(rows, svc)
	}

	sort.Slice(rows, func(i int, j int) bool {
		return rows[i].Name < rows[j].Name
	})

	for {
		search, err := readTrimmedLine(reader, output, "Search (name/description, Enter=all): ")
		if err != nil {
			return service.Service{}, fmt.Errorf("read service search: %w", err)
		}

		matches := filterServices(rows, search)
		if len(matches) == 0 {
			fmt.Fprintf(output, "No services match %q.\n", search)
			continue
		}

		fmt.Fprintln(output, "Matches:")
		for i, svc := range matches {
			display := strings.TrimSpace(svc.Description)
			if display == "" {
				display = svc.Name
			}
			fmt.Fprintf(output, "  %d) %s (%s)\n", i+1, svc.Name, display)
		}

		selection, err := readTrimmedLine(reader, output, "Service number: ")
		if err != nil {
			return service.Service{}, fmt.Errorf("read service selection: %w", err)
		}

		index, err := strconv.Atoi(selection)
		if err != nil || index < 1 || index > len(matches) {
			fmt.Fprintln(output, "Invalid selection.")
			continue
		}

		return matches[index-1], nil
	}
}

func filterServices(services []service.Service, query string) []service.Service {
	trimmedQuery := strings.ToLower(strings.TrimSpace(query))
	if trimmedQuery == "" {
		return services
	}

	filtered := make([]service.Service, 0, len(services))
	for _, svc := range services {
		if strings.Contains(strings.ToLower(svc.Name), trimmedQuery) || strings.Contains(strings.ToLower(svc.Description), trimmedQuery) {
			filtered = append(filtered, svc)
		}
	}

	return filtered
}

func pickTargetsInteractive(output ioWriter, reader *bufio.Reader) ([]targetpkg.Target, error) {
	targets := allTargets()
	if len(targets) == 0 {
		return nil, errors.New("no known targets found")
	}

	sortedTargets := make([]targetpkg.Target, len(targets))
	copy(sortedTargets, targets)
	sort.Slice(sortedTargets, func(i int, j int) bool {
		leftInstalled := sortedTargets[i].IsInstalled()
		rightInstalled := sortedTargets[j].IsInstalled()
		if leftInstalled != rightInstalled {
			return leftInstalled
		}

		return sortedTargets[i].Slug() < sortedTargets[j].Slug()
	})

	for {
		fmt.Fprintln(output, "Targets:")
		installedIndexes := make([]int, 0, len(sortedTargets))
		for i, targetDefinition := range sortedTargets {
			status := "not-installed"
			if targetDefinition.IsInstalled() {
				status = "installed"
				installedIndexes = append(installedIndexes, i+1)
			}

			fmt.Fprintf(output, "  %d) %s (%s) [%s]\n", i+1, targetDefinition.Name(), targetDefinition.Slug(), status)
		}

		if len(installedIndexes) == 0 {
			return nil, errors.New("no installed targets found")
		}

		selection, err := readTrimmedLine(reader, output, "Target numbers [e.g. 1,3] or \"all\": ")
		if err != nil {
			return nil, fmt.Errorf("read target selection: %w", err)
		}

		if strings.TrimSpace(selection) == "" {
			fmt.Fprintln(output, "Select at least one target.")
			continue
		}

		selectedTargets, parseErr := parseTargetSelection(selection, sortedTargets)
		if parseErr != nil {
			fmt.Fprintf(output, "Invalid target selection: %v\n", parseErr)
			continue
		}

		return selectedTargets, nil
	}
}

func parseTargetSelection(input string, targets []targetpkg.Target) ([]targetpkg.Target, error) {
	trimmedInput := strings.ToLower(strings.TrimSpace(input))
	if trimmedInput == "all" {
		selected := make([]targetpkg.Target, 0, len(targets))
		for _, targetDefinition := range targets {
			if targetDefinition.IsInstalled() {
				selected = append(selected, targetDefinition)
			}
		}

		if len(selected) == 0 {
			return nil, errors.New("no installed targets found")
		}

		return selected, nil
	}

	tokens := strings.Split(input, ",")
	selected := make([]targetpkg.Target, 0, len(tokens))
	seen := make(map[string]struct{})

	for _, token := range tokens {
		index, err := strconv.Atoi(strings.TrimSpace(token))
		if err != nil {
			return nil, fmt.Errorf("invalid target selection %q", token)
		}

		if index < 1 || index > len(targets) {
			return nil, fmt.Errorf("target index %d is out of range", index)
		}

		targetDefinition := targets[index-1]
		if !targetDefinition.IsInstalled() {
			return nil, fmt.Errorf("target %q is not installed", targetDefinition.Slug())
		}

		if _, exists := seen[targetDefinition.Slug()]; exists {
			continue
		}

		seen[targetDefinition.Slug()] = struct{}{}
		selected = append(selected, targetDefinition)
	}

	if len(selected) == 0 {
		return nil, errors.New("no targets selected")
	}

	return selected, nil
}

func confirmInstallSelection(
	output ioWriter,
	reader *bufio.Reader,
	svc service.Service,
	targetDefinitions []targetpkg.Target,
	noPrompt bool,
	scope targetpkg.ConfigScope,
) (bool, error) {
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Step 3/4: Review")
	fmt.Fprintf(output, "Service: %s\n", svc.Name)
	fmt.Fprintf(output, "Targets: %s\n", targetDisplayNames(targetDefinitions))
	credentialMode := "prompt as needed"
	if noPrompt {
		credentialMode = "existing values only"
	}
	fmt.Fprintf(output, "Credentials: %s\n", credentialMode)
	if anyTargetSupportsProjectScope(targetDefinitions) {
		fmt.Fprintf(output, "Scope (supported targets): %s\n", scopeDescription(scope))
	}

	confirmed, err := askYesNo(reader, output, "Apply changes? [Y/n]: ", true)
	if err != nil {
		return false, fmt.Errorf("read install confirmation: %w", err)
	}

	return confirmed, nil
}

func confirmUninstallSelection(
	output ioWriter,
	reader *bufio.Reader,
	svc service.Service,
	targetDefinitions []targetpkg.Target,
	scope targetpkg.ConfigScope,
) (bool, error) {
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Step 3/4: Review")
	fmt.Fprintf(output, "Service: %s\n", svc.Name)
	fmt.Fprintf(output, "Targets: %s\n", targetDisplayNames(targetDefinitions))
	if anyTargetSupportsProjectScope(targetDefinitions) {
		fmt.Fprintf(output, "Scope (supported targets): %s\n", scopeDescription(scope))
	}

	confirmed, err := askYesNo(reader, output, "Apply changes? [Y/n]: ", true)
	if err != nil {
		return false, fmt.Errorf("read uninstall confirmation: %w", err)
	}

	return confirmed, nil
}

func targetDisplayNames(targetDefinitions []targetpkg.Target) string {
	names := make([]string, 0, len(targetDefinitions))
	for _, targetDefinition := range targetDefinitions {
		names = append(names, targetDefinition.Name())
	}

	return strings.Join(names, ", ")
}

func buildEquivalentInstallCommand(serviceName string, targetDefinitions []targetpkg.Target, scope targetpkg.ConfigScope) string {
	command := "mcp-wire install " + serviceName
	for _, targetDefinition := range targetDefinitions {
		command += " --target " + targetDefinition.Slug()
	}
	if scope == targetpkg.ConfigScopeProject {
		command += " --scope project"
	}

	return command
}

func buildEquivalentUninstallCommand(serviceName string, targetDefinitions []targetpkg.Target, scope targetpkg.ConfigScope) string {
	command := "mcp-wire uninstall " + serviceName
	for _, targetDefinition := range targetDefinitions {
		command += " --target " + targetDefinition.Slug()
	}
	if scope == targetpkg.ConfigScopeProject {
		command += " --scope project"
	}

	return command
}

func resolveScopeForPlainWizard(
	output ioWriter,
	reader *bufio.Reader,
	targetDefinitions []targetpkg.Target,
	requestedScope targetpkg.ConfigScope,
	scopeSet bool,
	action string,
) (targetpkg.ConfigScope, error) {
	if !anyTargetSupportsProjectScope(targetDefinitions) {
		return targetpkg.ConfigScopeUser, nil
	}

	if scopeSet {
		return requestedScope, nil
	}

	for {
		prompt := fmt.Sprintf("%s scope for supported targets [1=user, 2=project, Enter=user]: ", action)
		selection, err := readTrimmedLine(reader, output, prompt)
		if err != nil {
			return "", fmt.Errorf("read scope selection: %w", err)
		}

		switch strings.ToLower(strings.TrimSpace(selection)) {
		case "", "1", "user":
			return targetpkg.ConfigScopeUser, nil
		case "2", "project":
			return targetpkg.ConfigScopeProject, nil
		default:
			fmt.Fprintln(output, "Invalid selection. Choose 1 (user) or 2 (project).")
		}
	}
}
