package tui

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/andreagrandi/mcp-wire/internal/catalog"
	"github.com/andreagrandi/mcp-wire/internal/service"
	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
)

// Callbacks provides functions that generate output for display in the TUI
// and configuration flags that control wizard behavior.
type Callbacks struct {
	RenderStatus          func(w io.Writer) error
	RenderServicesList    func(w io.Writer) error
	RenderTargetsList     func(w io.Writer) error
	LoadCatalog           func(source string) (*catalog.Catalog, error)
	RegistrySyncStatus    func() string
	RefreshRegistryEntry  func(catalog.Entry) catalog.Entry
	CatalogEntryToService func(catalog.Entry) (service.Service, bool)
	AllTargets            func() []targetpkg.Target
	RegistryEnabled       bool
}

// WizardState holds the accumulated selections across wizard screens.
type WizardState struct {
	Action  string                // "install" or "uninstall"
	Source  string                // "curated", "registry", "all"
	Entry   catalog.Entry         // selected service
	Targets []targetpkg.Target    // selected targets
	Scope   targetpkg.ConfigScope // "user" or "project"
}

// WizardModel is the root Bubble Tea model for the full-screen TUI.
type WizardModel struct {
	theme     Theme
	screen    Screen
	callbacks Callbacks
	version   string
	state     WizardState
	steps     []BreadcrumbStep
	width     int
	height    int
}

// NewWizardModel creates a new root model starting at the main menu.
func NewWizardModel(cb Callbacks, version string) WizardModel {
	theme := NewTheme()
	return WizardModel{
		theme:     theme,
		screen:    NewMenuScreen(theme),
		callbacks: cb,
		version:   version,
	}
}

func (m WizardModel) Init() tea.Cmd {
	return m.screen.Init()
}

func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Forward to screen so it can adjust (e.g. scroll bounds).

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case menuSelectMsg:
		return m.handleMenuSelect(msg)

	case sourceSelectMsg:
		return m.handleSourceSelect(msg)

	case serviceSelectMsg:
		return m.handleServiceSelect(msg)

	case trustConfirmMsg:
		return m.handleTrustConfirm(msg)

	case targetSelectMsg:
		return m.handleTargetSelect(msg)

	case scopeSelectMsg:
		return m.handleScopeSelect(msg)

	case reviewConfirmMsg:
		return m.handleReviewConfirm(msg)

	case BackMsg:
		return m.handleBack()
	}

	var cmd tea.Cmd
	m.screen, cmd = m.screen.Update(msg)
	return m, cmd
}

func (m WizardModel) View() string {
	// Title bar.
	titleLabel := "mcp-wire"
	if m.version != "" {
		titleLabel += " v" + m.version
	}

	titleText := m.theme.Title.Render(titleLabel)
	breadcrumb := RenderBreadcrumb(m.theme, m.steps)

	var titleBar string
	if breadcrumb != "" {
		titleBar = titleText + "  " + breadcrumb
	} else {
		titleBar = titleText
	}

	// Separator line.
	sepWidth := m.width
	if sepWidth <= 0 {
		sepWidth = 40
	}

	separator := m.theme.Separator.Render(strings.Repeat("\u2500", sepWidth))

	// Content area.
	content := m.screen.View()
	contentHeight := m.contentHeight()
	content = padToHeight(content, contentHeight)

	// Status bar.
	statusBar := RenderStatusBar(m.theme, m.screen.StatusHints(), m.width)

	return titleBar + "\n" + separator + "\n" + content + "\n" + statusBar
}

func (m WizardModel) contentHeight() int {
	return contentHeightFromTerminal(m.height)
}

func (m WizardModel) handleMenuSelect(msg menuSelectMsg) (tea.Model, tea.Cmd) {
	switch msg.item {
	case "Exit":
		return m, tea.Quit

	case "Status":
		m.screen = m.renderToOutput(m.callbacks.RenderStatus)
		return m, m.screen.Init()

	case "List services":
		m.screen = m.renderToOutput(m.callbacks.RenderServicesList)
		return m, m.screen.Init()

	case "List targets":
		m.screen = m.renderToOutput(m.callbacks.RenderTargetsList)
		return m, m.screen.Init()

	case "Install service":
		return m.startWizard("install")

	case "Uninstall service":
		return m.startWizard("uninstall")
	}

	return m, nil
}

func (m WizardModel) startWizard(action string) (tea.Model, tea.Cmd) {
	m.state = WizardState{Action: action}

	if m.callbacks.RegistryEnabled {
		m.screen = NewSourceScreen(m.theme)
		m.steps = []BreadcrumbStep{
			{Label: "Source", Active: true, Visible: true},
		}
		return m, m.screen.Init()
	}

	// Registry disabled — default to curated, skip source screen.
	m.state.Source = "curated"
	return m.showServiceScreen()
}

func (m WizardModel) handleSourceSelect(msg sourceSelectMsg) (tea.Model, tea.Cmd) {
	m.state.Source = msg.source
	return m.showServiceScreen()
}

func (m WizardModel) showServiceScreen() (tea.Model, tea.Cmd) {
	var steps []BreadcrumbStep
	if m.callbacks.RegistryEnabled {
		steps = append(steps, BreadcrumbStep{
			Label: "Source", Value: sourceValueLabel(m.state.Source),
			Completed: true, Visible: true,
		})
	}
	steps = append(steps, BreadcrumbStep{
		Label: "Service", Active: true, Visible: true,
	})
	m.steps = steps
	m.screen = NewServiceScreen(
		m.theme, m.state.Source, m.contentHeight(),
		m.callbacks.LoadCatalog, m.callbacks.RegistrySyncStatus,
	)
	return m, m.screen.Init()
}

func (m WizardModel) handleServiceSelect(msg serviceSelectMsg) (tea.Model, tea.Cmd) {
	m.state.Entry = msg.entry

	if registryEntryNeedsConfirmation(msg.entry) {
		return m.showTrustScreen()
	}

	return m.showTargetScreen()
}

func (m WizardModel) handleTrustConfirm(msg trustConfirmMsg) (tea.Model, tea.Cmd) {
	if !msg.confirmed {
		// Back to service selection.
		return m.showServiceScreen()
	}

	// Refresh the entry with latest details.
	if m.callbacks.RefreshRegistryEntry != nil {
		m.state.Entry = m.callbacks.RefreshRegistryEntry(m.state.Entry)
	}

	return m.showTargetScreen()
}

func (m WizardModel) showTrustScreen() (tea.Model, tea.Cmd) {
	var steps []BreadcrumbStep
	if m.callbacks.RegistryEnabled {
		steps = append(steps, BreadcrumbStep{
			Label: "Source", Value: sourceValueLabel(m.state.Source),
			Completed: true, Visible: true,
		})
	}
	steps = append(steps, BreadcrumbStep{
		Label: "Service", Value: m.state.Entry.Name,
		Completed: true, Visible: true,
	})
	steps = append(steps, BreadcrumbStep{
		Label: "Trust", Active: true, Visible: true,
	})
	m.steps = steps
	m.screen = NewTrustScreen(m.theme, m.state.Entry)
	return m, m.screen.Init()
}

func (m WizardModel) showTargetScreen() (tea.Model, tea.Cmd) {
	var steps []BreadcrumbStep
	if m.callbacks.RegistryEnabled {
		steps = append(steps, BreadcrumbStep{
			Label: "Source", Value: sourceValueLabel(m.state.Source),
			Completed: true, Visible: true,
		})
	}
	steps = append(steps, BreadcrumbStep{
		Label: "Service", Value: m.state.Entry.Name,
		Completed: true, Visible: true,
	})
	steps = append(steps, BreadcrumbStep{
		Label: "Targets", Active: true, Visible: true,
	})
	m.steps = steps

	var allTargets []targetpkg.Target
	if m.callbacks.AllTargets != nil {
		allTargets = m.callbacks.AllTargets()
	}

	m.screen = NewTargetScreen(m.theme, allTargets, m.state.Targets)
	return m, m.screen.Init()
}

func (m WizardModel) handleTargetSelect(msg targetSelectMsg) (tea.Model, tea.Cmd) {
	m.state.Targets = msg.targets

	if anyTargetSupportsProjectScope(msg.targets) {
		return m.showScopeScreen()
	}

	// No scope selection needed — default to user scope.
	m.state.Scope = targetpkg.ConfigScopeUser
	return m.showReviewScreen()
}

func (m WizardModel) showScopeScreen() (tea.Model, tea.Cmd) {
	var steps []BreadcrumbStep
	if m.callbacks.RegistryEnabled {
		steps = append(steps, BreadcrumbStep{
			Label: "Source", Value: sourceValueLabel(m.state.Source),
			Completed: true, Visible: true,
		})
	}
	steps = append(steps, BreadcrumbStep{
		Label: "Service", Value: m.state.Entry.Name,
		Completed: true, Visible: true,
	})
	steps = append(steps, BreadcrumbStep{
		Label: "Targets", Value: targetSummary(m.state.Targets),
		Completed: true, Visible: true,
	})
	steps = append(steps, BreadcrumbStep{
		Label: "Scope", Active: true, Visible: true,
	})
	m.steps = steps
	m.screen = NewScopeScreen(m.theme)
	return m, m.screen.Init()
}

func (m WizardModel) handleScopeSelect(msg scopeSelectMsg) (tea.Model, tea.Cmd) {
	m.state.Scope = msg.scope
	return m.showReviewScreen()
}

func (m WizardModel) showReviewScreen() (tea.Model, tea.Cmd) {
	m.steps = m.reviewBreadcrumbs()
	m.screen = NewReviewScreen(m.theme, m.state, m.callbacks.RegistryEnabled)
	return m, m.screen.Init()
}

func (m WizardModel) handleReviewConfirm(msg reviewConfirmMsg) (tea.Model, tea.Cmd) {
	if !msg.confirmed {
		return m.reviewGoBack()
	}

	return m.showApplyPlaceholder()
}

// reviewGoBack navigates back from the review screen to the previous step.
func (m WizardModel) reviewGoBack() (tea.Model, tea.Cmd) {
	if anyTargetSupportsProjectScope(m.state.Targets) {
		m.state.Scope = ""
		return m.showScopeScreen()
	}

	return m.showTargetScreen()
}

// showApplyPlaceholder shows a placeholder for the apply screen
// (to be replaced in step 8.7).
func (m WizardModel) showApplyPlaceholder() (tea.Model, tea.Cmd) {
	m.steps = m.reviewBreadcrumbs()

	content := "Apply is not yet available in the TUI.\n" +
		"Use the command directly:\n\n" +
		"  mcp-wire " + m.state.Action + " " + m.state.Entry.Name + "\n"
	m.screen = NewOutputScreen(m.theme, content, m.contentHeight())
	return m, m.screen.Init()
}

// reviewBreadcrumbs builds the breadcrumb steps for the review screen.
func (m WizardModel) reviewBreadcrumbs() []BreadcrumbStep {
	var steps []BreadcrumbStep
	if m.callbacks.RegistryEnabled {
		steps = append(steps, BreadcrumbStep{
			Label: "Source", Value: sourceValueLabel(m.state.Source),
			Completed: true, Visible: true,
		})
	}
	steps = append(steps, BreadcrumbStep{
		Label: "Service", Value: m.state.Entry.Name,
		Completed: true, Visible: true,
	})
	steps = append(steps, BreadcrumbStep{
		Label: "Targets", Value: targetSummary(m.state.Targets),
		Completed: true, Visible: true,
	})
	if m.state.Scope == targetpkg.ConfigScopeProject {
		steps = append(steps, BreadcrumbStep{
			Label: "Scope", Value: "project",
			Completed: true, Visible: true,
		})
	}
	return steps
}

// targetSummary returns a short label for the selected targets.
func targetSummary(targets []targetpkg.Target) string {
	if len(targets) == 0 {
		return ""
	}
	if len(targets) == 1 {
		return targets[0].Name()
	}
	return targets[0].Name() + " +" + fmt.Sprintf("%d", len(targets)-1)
}

// anyTargetSupportsProjectScope checks if any of the given targets support
// the project config scope.
func anyTargetSupportsProjectScope(targets []targetpkg.Target) bool {
	for _, t := range targets {
		st, ok := t.(targetpkg.ScopedTarget)
		if !ok {
			continue
		}
		for _, s := range st.SupportedScopes() {
			if s == targetpkg.ConfigScopeProject {
				return true
			}
		}
	}
	return false
}

func (m WizardModel) handleBack() (tea.Model, tea.Cmd) {
	switch m.screen.(type) {
	case *ReviewScreen:
		return m.reviewGoBack()

	case *ScopeScreen:
		// Back from scope goes to target selection, preserving selections.
		return m.showTargetScreen()

	case *TargetScreen:
		// Back from target goes to service selection.
		m.state.Targets = nil
		m.state.Scope = ""
		m.state.Entry = catalog.Entry{}
		return m.showServiceScreen()

	case *TrustScreen:
		// Back from trust goes to service selection.
		m.state.Entry = catalog.Entry{}
		return m.showServiceScreen()

	case *ServiceScreen:
		if m.callbacks.RegistryEnabled {
			// Back to source selection.
			m.screen = NewSourceScreen(m.theme)
			m.state.Source = ""
			m.state.Entry = catalog.Entry{}
			m.steps = []BreadcrumbStep{
				{Label: "Source", Active: true, Visible: true},
			}
			return m, m.screen.Init()
		}
	}

	// Default: return to menu.
	m.screen = NewMenuScreen(m.theme)
	m.state = WizardState{}
	m.steps = nil
	return m, m.screen.Init()
}

// sourceValueLabel returns a display label for a source value.
func sourceValueLabel(source string) string {
	labels := map[string]string{
		"curated":  "Curated",
		"registry": "Registry",
		"all":      "Both",
	}
	if l, ok := labels[source]; ok {
		return l
	}
	return source
}

// renderToOutput runs a callback, captures its output, and returns an
// OutputScreen displaying the result.
func (m WizardModel) renderToOutput(fn func(io.Writer) error) *OutputScreen {
	if fn == nil {
		return NewOutputScreen(m.theme, "(not available)", m.contentHeight())
	}

	var buf bytes.Buffer
	if err := fn(&buf); err != nil {
		return NewOutputScreen(m.theme, "Error: "+err.Error(), m.contentHeight())
	}

	return NewOutputScreen(m.theme, buf.String(), m.contentHeight())
}

// contentHeightFromTerminal calculates the content area height from the
// terminal height, subtracting the chrome lines (title + separator + status bar).
func contentHeightFromTerminal(termHeight int) int {
	if termHeight <= 0 {
		return ContentHeight
	}

	h := termHeight - ChromeLines
	if h < 1 {
		h = 1
	}

	return h
}

// padToHeight pads or truncates content to exactly targetHeight lines.
func padToHeight(content string, targetHeight int) string {
	content = strings.TrimRight(content, "\n")

	lines := strings.Split(content, "\n")
	for len(lines) < targetHeight {
		lines = append(lines, "")
	}

	if len(lines) > targetHeight {
		lines = lines[:targetHeight]
	}

	return strings.Join(lines, "\n")
}

// Run starts the full-screen TUI.
func Run(cb Callbacks, version string) error {
	p := tea.NewProgram(NewWizardModel(cb, version), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
