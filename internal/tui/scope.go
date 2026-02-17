package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
)

// scopeOption describes one entry in the scope selection screen.
type scopeOption struct {
	Label       string
	Description string
	Value       targetpkg.ConfigScope
}

var scopeOptions = []scopeOption{
	{Label: "User", Description: "Apply to user/global configuration", Value: targetpkg.ConfigScopeUser},
	{Label: "Project", Description: "Apply to current project only", Value: targetpkg.ConfigScopeProject},
}

// ScopeScreen lets the user choose between user and project scope.
type ScopeScreen struct {
	theme  Theme
	cursor int
	width  int
}

// NewScopeScreen creates a new scope selection screen.
func NewScopeScreen(theme Theme) *ScopeScreen {
	return &ScopeScreen{theme: theme}
}

func (s *ScopeScreen) Init() tea.Cmd { return nil }

func (s *ScopeScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		return s, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if s.cursor > 0 {
				s.cursor--
			}
		case "down", "j":
			if s.cursor < len(scopeOptions)-1 {
				s.cursor++
			}
		case "enter":
			opt := scopeOptions[s.cursor]
			return s, func() tea.Msg {
				return scopeSelectMsg{scope: opt.Value}
			}
		case "esc":
			return s, func() tea.Msg { return BackMsg{} }
		}
	}

	return s, nil
}

func (s *ScopeScreen) View() string {
	var b strings.Builder

	b.WriteString("\n")

	for i, opt := range scopeOptions {
		if i == s.cursor {
			label := "  \u276f " + opt.Label
			if s.width > 0 {
				b.WriteString(s.theme.Highlight.Width(s.width).Render(label))
			} else {
				b.WriteString(s.theme.Cursor.Render(label))
			}
		} else {
			b.WriteString("    " + opt.Label)
		}
		b.WriteString("\n")

		// Description line (indented, dimmed).
		b.WriteString(s.theme.Dim.Render("      " + opt.Description))
		b.WriteString("\n")
	}

	return b.String()
}

func (s *ScopeScreen) StatusHints() []KeyHint {
	return []KeyHint{
		{Key: "\u2191\u2193", Desc: "move"},
		{Key: "Enter", Desc: "select"},
		{Key: "Esc", Desc: "back"},
	}
}

// Cursor returns the current cursor position (for testing).
func (s *ScopeScreen) Cursor() int {
	return s.cursor
}
