package tui

import (
	"bytes"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Callbacks provides functions that generate output for display in the TUI
// and configuration flags that control wizard behavior.
type Callbacks struct {
	RenderStatus       func(w io.Writer) error
	RenderServicesList func(w io.Writer) error
	RenderTargetsList  func(w io.Writer) error
	RegistryEnabled    bool
}

// WizardState holds the accumulated selections across wizard screens.
type WizardState struct {
	Action string // "install" or "uninstall"
	Source string // "curated", "registry", "all"
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

	case BackMsg:
		m.screen = NewMenuScreen(m.theme)
		m.state = WizardState{}
		m.steps = nil
		return m, m.screen.Init()
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
	return m.showServicePlaceholder()
}

func (m WizardModel) handleSourceSelect(msg sourceSelectMsg) (tea.Model, tea.Cmd) {
	m.state.Source = msg.source
	return m.showServicePlaceholder()
}

// showServicePlaceholder shows a placeholder for the service selection screen
// (to be replaced in step 8.3).
func (m WizardModel) showServicePlaceholder() (tea.Model, tea.Cmd) {
	m.steps = sourceCompletedSteps(m.state.Source)
	content := "Service selection is not yet available in the TUI.\n" +
		"Use the command directly:\n\n" +
		"  mcp-wire " + m.state.Action + " <service>\n"
	m.screen = NewOutputScreen(m.theme, content, m.contentHeight())
	return m, m.screen.Init()
}

// sourceCompletedSteps returns breadcrumb steps with Source completed.
func sourceCompletedSteps(source string) []BreadcrumbStep {
	labels := map[string]string{
		"curated":  "Curated",
		"registry": "Registry",
		"all":      "Both",
	}

	return []BreadcrumbStep{
		{Label: "Source", Value: labels[source], Completed: true, Visible: true},
	}
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
