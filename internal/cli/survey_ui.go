package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	surveycore "github.com/AlecAivazis/survey/v2/core"
	surveyterminal "github.com/AlecAivazis/survey/v2/terminal"
	"github.com/andreagrandi/mcp-wire/internal/catalog"
	"github.com/andreagrandi/mcp-wire/internal/service"
	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var askSurveyOne = func(prompt survey.Prompt, response interface{}, opts ...survey.AskOpt) error {
	return survey.AskOne(prompt, response, opts...)
}

var errWizardBack = errors.New("wizard back")

func canUseInteractiveUI(input io.Reader, output io.Writer) bool {
	inputFile, inputOK := input.(*os.File)
	outputFile, outputOK := output.(*os.File)
	if !inputOK || !outputOK {
		return false
	}

	return term.IsTerminal(int(inputFile.Fd())) && term.IsTerminal(int(outputFile.Fd()))
}

func runGuidedMainMenuSurvey(cmd *cobra.Command) error {
	showMenuSpacing := false

	for {
		if showMenuSpacing {
			fmt.Fprintln(cmd.OutOrStdout())
		}

		printSurveyHint(cmd.OutOrStdout(), "Use Up/Down arrows, Enter to select. Esc keeps you in the main menu.")

		choice := ""
		prompt := &survey.Select{
			Message:  "Main Menu",
			Options:  []string{"Install service", "Uninstall service", "Status", "List services", "List targets", "Exit"},
			PageSize: 6,
		}

		if err := askSurveyPrompt(cmd, prompt, &choice); err != nil {
			if errors.Is(err, errWizardBack) {
				fmt.Fprintln(cmd.OutOrStdout())
				continue
			}

			return fmt.Errorf("read menu option: %w", err)
		}

		switch choice {
		case "Install service":
			if err := runInstallWizardSurvey(cmd, nil, false, targetpkg.ConfigScopeUser, false); err != nil {
				return err
			}
			showMenuSpacing = true
		case "Uninstall service":
			if err := runUninstallWizardSurvey(cmd, nil, targetpkg.ConfigScopeUser, false); err != nil {
				return err
			}
			showMenuSpacing = true
		case "Status":
			fmt.Fprintln(cmd.OutOrStdout())
			if err := runStatusFlow(cmd.OutOrStdout(), targetpkg.ConfigScopeEffective); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout())
			showMenuSpacing = true
		case "List services":
			services, err := loadServices()
			if err != nil {
				return fmt.Errorf("load services: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout())
			printServicesList(cmd.OutOrStdout(), services)
			fmt.Fprintln(cmd.OutOrStdout())
			showMenuSpacing = true
		case "List targets":
			fmt.Fprintln(cmd.OutOrStdout())
			printTargetsList(cmd.OutOrStdout(), allTargets())
			fmt.Fprintln(cmd.OutOrStdout())
			showMenuSpacing = true
		case "Exit":
			fmt.Fprintln(cmd.OutOrStdout(), "Goodbye.")
			return nil
		}
	}
}

func pickSourceSurvey(cmd *cobra.Command) (string, error) {
	sourceOptions := []string{
		"Curated services (recommended)",
		"Registry services (community)",
		"Both",
	}

	selectedLabel := ""
	printSurveyHint(cmd.OutOrStdout(), "Use Up/Down arrows, Enter to select. Esc goes back.")

	prompt := &survey.Select{
		Message:  "Source",
		Options:  sourceOptions,
		Default:  sourceOptions[0],
		PageSize: 3,
	}

	if err := askSurveyPrompt(cmd, prompt, &selectedLabel); err != nil {
		return "", err
	}

	switch selectedLabel {
	case sourceOptions[1]:
		return "registry", nil
	case sourceOptions[2]:
		return "all", nil
	default:
		return "curated", nil
	}
}

func runInstallWizardSurvey(
	cmd *cobra.Command,
	targetSlugs []string,
	noPrompt bool,
	requestedScope targetpkg.ConfigScope,
	scopeSet bool,
) error {
	output := cmd.OutOrStdout()

	fmt.Fprintln(output, "Install Wizard")
	fmt.Fprintln(output)

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	registryEnabled := cfg.IsFeatureEnabled("registry")

	services, err := loadServices()
	if err != nil {
		return fmt.Errorf("load services: %w", err)
	}

sourceStep:
	for {
		source := "curated"
		if registryEnabled {
			var sourceErr error
			source, sourceErr = pickSourceSurvey(cmd)
			if sourceErr != nil {
				if errors.Is(sourceErr, errWizardBack) {
					fmt.Fprintln(output, "Install cancelled.")
					return nil
				}

				return sourceErr
			}

			fmt.Fprintln(output)
		}

	serviceStep:
		for {
			fmt.Fprintln(output, "Step 1/4: Service")

			svc, err := pickServiceSurvey(cmd, services, registryEnabled, source)
			if err != nil {
				if errors.Is(err, errRegistryOnly) {
					fmt.Fprintln(output)
					continue sourceStep
				}

				if errors.Is(err, errWizardBack) {
					if registryEnabled {
						continue sourceStep
					}

					fmt.Fprintln(output, "Install cancelled.")
					return nil
				}

				return err
			}

		targetStep:
			for {
				fmt.Fprintln(output)
				fmt.Fprintln(output, "Step 2/4: Targets")

				targetDefinitions, err := resolveTargetsForSurveyWizard(cmd, targetSlugs)
				if err != nil {
					if errors.Is(err, errWizardBack) {
						continue serviceStep
					}

					return err
				}

				selectedScope, err := resolveScopeForSurveyWizard(cmd, targetDefinitions, requestedScope, scopeSet, "Install")
				if err != nil {
					if errors.Is(err, errWizardBack) {
						if len(targetSlugs) > 0 {
							continue serviceStep
						}

						continue targetStep
					}

					return err
				}

				for {
					fmt.Fprintln(output)
					fmt.Fprintln(output, "Step 3/4: Review")
					fmt.Fprintf(output, "Service: %s\n", svc.Name)
					fmt.Fprintf(output, "Targets: %s\n", targetDisplayNames(targetDefinitions))
					if anyTargetSupportsProjectScope(targetDefinitions) {
						fmt.Fprintf(output, "Scope (supported targets): %s\n", scopeDescription(selectedScope))
					}
					credentialMode := "prompt as needed"
					if noPrompt {
						credentialMode = "existing values only"
					}
					fmt.Fprintf(output, "Credentials: %s\n", credentialMode)

					confirmChoice := ""
					printSurveyHint(output, "Use Up/Down arrows, Enter to select. Esc goes back.")

					confirmPrompt := &survey.Select{
						Message:  "Apply changes?",
						Options:  []string{"Yes", "No"},
						Default:  "Yes",
						PageSize: 2,
					}
					if err := askSurveyPrompt(cmd, confirmPrompt, &confirmChoice); err != nil {
						if errors.Is(err, errWizardBack) {
							if len(targetSlugs) > 0 {
								continue serviceStep
							}

							continue targetStep
						}

						return fmt.Errorf("read install confirmation: %w", err)
					}

					if confirmChoice != "Yes" {
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
			}
		}
	}
}

func runUninstallWizardSurvey(
	cmd *cobra.Command,
	targetSlugs []string,
	requestedScope targetpkg.ConfigScope,
	scopeSet bool,
) error {
	output := cmd.OutOrStdout()

	fmt.Fprintln(output, "Uninstall Wizard")
	fmt.Fprintln(output)

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	registryEnabled := cfg.IsFeatureEnabled("registry")

	services, err := loadServices()
	if err != nil {
		return fmt.Errorf("load services: %w", err)
	}

sourceStep:
	for {
		source := "curated"
		if registryEnabled {
			var sourceErr error
			source, sourceErr = pickSourceSurvey(cmd)
			if sourceErr != nil {
				if errors.Is(sourceErr, errWizardBack) {
					fmt.Fprintln(output, "Uninstall cancelled.")
					return nil
				}

				return sourceErr
			}

			fmt.Fprintln(output)
		}

	serviceStep:
		for {
			fmt.Fprintln(output, "Step 1/4: Service")

			svc, err := pickServiceSurvey(cmd, services, registryEnabled, source)
			if err != nil {
				if errors.Is(err, errRegistryOnly) {
					fmt.Fprintln(output)
					continue sourceStep
				}

				if errors.Is(err, errWizardBack) {
					if registryEnabled {
						continue sourceStep
					}

					fmt.Fprintln(output, "Uninstall cancelled.")
					return nil
				}

				return err
			}

		targetStep:
			for {
				fmt.Fprintln(output)
				fmt.Fprintln(output, "Step 2/4: Targets")

				targetDefinitions, err := resolveTargetsForSurveyWizard(cmd, targetSlugs)
				if err != nil {
					if errors.Is(err, errWizardBack) {
						continue serviceStep
					}

					return err
				}

				selectedScope, err := resolveScopeForSurveyWizard(cmd, targetDefinitions, requestedScope, scopeSet, "Uninstall")
				if err != nil {
					if errors.Is(err, errWizardBack) {
						if len(targetSlugs) > 0 {
							continue serviceStep
						}

						continue targetStep
					}

					return err
				}

				for {
					fmt.Fprintln(output)
					fmt.Fprintln(output, "Step 3/4: Review")
					fmt.Fprintf(output, "Service: %s\n", svc.Name)
					fmt.Fprintf(output, "Targets: %s\n", targetDisplayNames(targetDefinitions))
					if anyTargetSupportsProjectScope(targetDefinitions) {
						fmt.Fprintf(output, "Scope (supported targets): %s\n", scopeDescription(selectedScope))
					}

					confirmChoice := ""
					printSurveyHint(output, "Use Up/Down arrows, Enter to select. Esc goes back.")

					confirmPrompt := &survey.Select{
						Message:  "Apply changes?",
						Options:  []string{"Yes", "No"},
						Default:  "Yes",
						PageSize: 2,
					}
					if err := askSurveyPrompt(cmd, confirmPrompt, &confirmChoice); err != nil {
						if errors.Is(err, errWizardBack) {
							if len(targetSlugs) > 0 {
								continue serviceStep
							}

							continue targetStep
						}

						return fmt.Errorf("read uninstall confirmation: %w", err)
					}

					if confirmChoice != "Yes" {
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
			}
		}
	}
}

func resolveScopeForSurveyWizard(
	cmd *cobra.Command,
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

	printSurveyHint(cmd.OutOrStdout(), "Use Up/Down arrows, Enter to select. Esc goes back.")
	selection := ""
	prompt := &survey.Select{
		Message:  fmt.Sprintf("%s scope (supported targets)", action),
		Options:  []string{"User (global)", "Project (current directory)"},
		Default:  "User (global)",
		PageSize: 2,
	}

	if err := askSurveyPrompt(cmd, prompt, &selection); err != nil {
		return "", fmt.Errorf("read scope selection: %w", err)
	}

	if selection == "Project (current directory)" {
		return targetpkg.ConfigScopeProject, nil
	}

	return targetpkg.ConfigScopeUser, nil
}

func pickServiceSurvey(cmd *cobra.Command, services map[string]service.Service, registryEnabled bool, source string) (service.Service, error) {
	if registryEnabled && source != "curated" {
		return pickServiceSurveyCatalog(cmd, source)
	}

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

	labels := make([]string, 0, len(rows))
	serviceByLabel := make(map[string]service.Service, len(rows))
	for _, svc := range rows {
		description := strings.TrimSpace(svc.Description)
		if description == "" {
			description = svc.Name
		}

		label := fmt.Sprintf("%s - %s", svc.Name, description)
		labels = append(labels, label)
		serviceByLabel[label] = svc
	}

	selectedLabel := ""
	printSurveyHint(cmd.OutOrStdout(), "Use Up/Down arrows, Enter to select. Type to filter. Esc goes back.")

	prompt := &survey.Select{
		Message:  "Select service",
		Options:  labels,
		PageSize: 10,
		Filter: func(filter string, value string, _ int) bool {
			if strings.TrimSpace(filter) == "" {
				return true
			}

			return strings.Contains(strings.ToLower(value), strings.ToLower(filter))
		},
		FilterMessage: "Filter:",
	}

	if err := askSurveyPrompt(cmd, prompt, &selectedLabel); err != nil {
		return service.Service{}, fmt.Errorf("read service selection: %w", err)
	}

	svc, found := serviceByLabel[selectedLabel]
	if !found {
		return service.Service{}, fmt.Errorf("selected service %q not found", selectedLabel)
	}

	return svc, nil
}

func pickServiceSurveyCatalog(cmd *cobra.Command, source string) (service.Service, error) {
	cat, err := loadCatalog(source, true)
	if err != nil {
		return service.Service{}, err
	}

	entries := cat.All()
	if len(entries) == 0 {
		return service.Service{}, errors.New("no service definitions available")
	}

	showMarkers := source == "all"

	for {
		labels := make([]string, 0, len(entries))
		entryByLabel := make(map[string]catalog.Entry, len(entries))

		for _, entry := range entries {
			description := strings.TrimSpace(entry.Description())
			if description == "" {
				description = entry.Name
			}

			prefix := ""
			if showMarkers && entry.Source == catalog.SourceCurated {
				prefix = "* "
			} else if showMarkers {
				prefix = "  "
			}

			label := fmt.Sprintf("%s%s - %s", prefix, entry.Name, description)
			labels = append(labels, label)
			entryByLabel[label] = entry
		}

		if showMarkers {
			fmt.Fprintln(cmd.OutOrStdout(), "(* = curated by mcp-wire)")
		}

		selectedLabel := ""
		printSurveyHint(cmd.OutOrStdout(), "Use Up/Down arrows, Enter to select. Type to filter. Esc goes back.")

		prompt := &survey.Select{
			Message:  "Select service",
			Options:  labels,
			PageSize: 10,
			Filter: func(filter string, value string, _ int) bool {
				if strings.TrimSpace(filter) == "" {
					return true
				}

				return strings.Contains(strings.ToLower(value), strings.ToLower(filter))
			},
			FilterMessage: "Filter:",
		}

		if err := askSurveyPrompt(cmd, prompt, &selectedLabel); err != nil {
			return service.Service{}, fmt.Errorf("read service selection: %w", err)
		}

		selected, found := entryByLabel[selectedLabel]
		if !found {
			return service.Service{}, fmt.Errorf("selected service %q not found", selectedLabel)
		}

		svc, ok := catalogEntryToService(selected)
		if !ok {
			if source == "registry" {
				fmt.Fprintln(cmd.OutOrStdout(), "Registry services cannot be installed yet.")
				return service.Service{}, errRegistryOnly
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Registry services cannot be installed yet. Choose a curated service.")
			continue
		}

		return svc, nil
	}
}

func resolveTargetsForSurveyWizard(cmd *cobra.Command, targetSlugs []string) ([]targetpkg.Target, error) {
	if len(targetSlugs) > 0 {
		return resolveInstallTargets(targetSlugs)
	}

	return pickTargetsSurvey(cmd)
}

func pickTargetsSurvey(cmd *cobra.Command) ([]targetpkg.Target, error) {
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

	fmt.Fprintln(cmd.OutOrStdout(), "Detected targets:")

	installedTargets := make([]targetpkg.Target, 0, len(sortedTargets))
	installedLabels := make([]string, 0, len(sortedTargets))
	labelToTarget := make(map[string]targetpkg.Target, len(sortedTargets))

	for _, targetDefinition := range sortedTargets {
		status := "not-installed"
		if targetDefinition.IsInstalled() {
			status = "installed"
			installedTargets = append(installedTargets, targetDefinition)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "  %s (%s) [%s]\n", targetDefinition.Name(), targetDefinition.Slug(), status)
	}

	if len(installedTargets) == 0 {
		return nil, errors.New("no installed targets found")
	}

	for _, targetDefinition := range installedTargets {
		label := fmt.Sprintf("%s (%s)", targetDefinition.Name(), targetDefinition.Slug())
		installedLabels = append(installedLabels, label)
		labelToTarget[label] = targetDefinition
	}

	for {
		var selectedLabels []string
		printSurveyHint(cmd.OutOrStdout(), "Use Up/Down arrows, Space to toggle, Right to select all, Left to clear all, Enter to confirm. Type to filter. Esc goes back.")

		prompt := &survey.MultiSelect{
			Message:  "Select targets",
			Options:  installedLabels,
			PageSize: 8,
			Filter: func(filter string, value string, _ int) bool {
				if strings.TrimSpace(filter) == "" {
					return true
				}

				return strings.Contains(strings.ToLower(value), strings.ToLower(filter))
			},
			FilterMessage: "Filter:",
		}

		if err := askSurveyPrompt(cmd, prompt, &selectedLabels); err != nil {
			return nil, fmt.Errorf("read target selection: %w", err)
		}

		if len(selectedLabels) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "Select at least one target.")
			continue
		}

		selectedTargets := make([]targetpkg.Target, 0, len(selectedLabels))
		for _, label := range selectedLabels {
			targetDefinition, found := labelToTarget[label]
			if !found {
				return nil, fmt.Errorf("selected target %q not found", label)
			}

			selectedTargets = append(selectedTargets, targetDefinition)
		}

		return selectedTargets, nil
	}
}

func askSurveyPrompt(cmd *cobra.Command, prompt survey.Prompt, response interface{}) error {
	colorEnabled := surveyColorsEnabled()
	previousDisableColor := surveycore.DisableColor
	surveycore.DisableColor = !colorEnabled
	defer func() {
		surveycore.DisableColor = previousDisableColor
	}()

	questionFormat := "default"
	selectFocusFormat := "default"
	markedFormat := "default"
	if colorEnabled {
		questionFormat = "cyan"
		selectFocusFormat = "cyan"
		markedFormat = "green"
	}

	options := []survey.AskOpt{survey.WithIcons(func(icons *survey.IconSet) {
		icons.Question.Text = ">"
		icons.Question.Format = questionFormat
		icons.SelectFocus.Text = ">"
		icons.SelectFocus.Format = selectFocusFormat
		icons.MarkedOption.Text = "[x]"
		icons.MarkedOption.Format = markedFormat
		icons.UnmarkedOption.Text = "[ ]"
		icons.UnmarkedOption.Format = "default"
	})}

	var escBackInput *surveyEscBackInput

	inputFile, inputOK := cmd.InOrStdin().(*os.File)
	outputFile, outputOK := cmd.OutOrStdout().(*os.File)
	if inputOK && outputOK {
		escBackInput = newSurveyEscBackInput(inputFile)
		options = append(options, survey.WithStdio(escBackInput, outputFile, outputFile))
	}

	err := askSurveyOne(prompt, response, options...)
	if err == nil {
		return nil
	}

	if escBackInput != nil && escBackInput.ConsumeBackPressed() && errors.Is(err, surveyterminal.InterruptErr) {
		return errWizardBack
	}

	return err
}

func surveyColorsEnabled() bool {
	if strings.TrimSpace(os.Getenv("NO_COLOR")) != "" {
		return false
	}

	termValue := strings.TrimSpace(strings.ToLower(os.Getenv("TERM")))
	return termValue != "dumb"
}

func printSurveyHint(output io.Writer, message string) {
	fmt.Fprintln(output, message)
}
