package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var menuItems = []string{
	"Install service",
	"Uninstall service",
	"Status",
	"List services",
	"List targets",
	"Exit",
}

// MenuScreen is the main menu of the TUI wizard.
type MenuScreen struct {
	theme  Theme
	cursor int
}

// NewMenuScreen creates a new main menu screen.
func NewMenuScreen(theme Theme) *MenuScreen {
	return &MenuScreen{theme: theme}
}

func (m *MenuScreen) Init() tea.Cmd { return nil }

func (m *MenuScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(menuItems)-1 {
				m.cursor++
			}
		case "enter":
			item := menuItems[m.cursor]
			return m, func() tea.Msg {
				return menuSelectMsg{item: item}
			}
		case "q":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *MenuScreen) View() string {
	var b strings.Builder

	b.WriteString("\n")

	for i, item := range menuItems {
		if i == m.cursor {
			b.WriteString("  " + m.theme.Cursor.Render("\u25b8 "+item))
		} else {
			b.WriteString("    " + item)
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m *MenuScreen) StatusHints() []KeyHint {
	return []KeyHint{
		{Key: "\u2191\u2193", Desc: "navigate"},
		{Key: "enter", Desc: "select"},
		{Key: "q", Desc: "quit"},
	}
}

// Cursor returns the current cursor position (for testing).
func (m *MenuScreen) Cursor() int {
	return m.cursor
}
