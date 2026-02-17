package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// WizardState holds the accumulated selections across wizard screens.
type WizardState struct {
	Action string // "install" or "uninstall"
	Source string // "curated", "registry", "all"
}

// WizardModel is the root Bubble Tea model for the full-screen TUI.
type WizardModel struct {
	theme  Theme
	screen Screen
	state  WizardState
	steps  []BreadcrumbStep
	width  int
	height int
}

// NewWizardModel creates a new root model starting at the main menu.
func NewWizardModel() WizardModel {
	theme := NewTheme()
	return WizardModel{
		theme:  theme,
		screen: NewMenuScreen(theme),
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
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case menuSelectMsg:
		return m.handleMenuSelect(msg)

	case BackMsg:
		m.screen = NewMenuScreen(m.theme)
		m.steps = nil
		return m, m.screen.Init()
	}

	var cmd tea.Cmd
	m.screen, cmd = m.screen.Update(msg)
	return m, cmd
}

func (m WizardModel) View() string {
	titleText := m.theme.Title.Render("mcp-wire")
	breadcrumb := RenderBreadcrumb(m.theme, m.steps)

	var titleBar string
	if breadcrumb != "" {
		titleBar = titleText + "  " + breadcrumb
	} else {
		titleBar = titleText
	}

	content := m.screen.View()
	contentHeight := m.contentHeight()
	content = padToHeight(content, contentHeight)

	statusBar := RenderStatusBar(m.theme, m.screen.StatusHints(), m.width)

	return titleBar + "\n" + content + "\n" + statusBar
}

func (m WizardModel) contentHeight() int {
	if m.height <= 0 {
		return ContentHeight
	}

	h := m.height - 2 // title bar + status bar
	if h < 1 {
		h = 1
	}

	return h
}

func (m WizardModel) handleMenuSelect(msg menuSelectMsg) (tea.Model, tea.Cmd) {
	switch msg.item {
	case "Exit":
		return m, tea.Quit
	}

	// Other menu items will be handled in later steps.
	return m, nil
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
func Run() error {
	p := tea.NewProgram(NewWizardModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
