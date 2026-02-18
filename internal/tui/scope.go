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
	{Label: "User", Description: "available across all projects (default)", Value: targetpkg.ConfigScopeUser},
	{Label: "Project", Description: "only for the current directory", Value: targetpkg.ConfigScopeProject},
}

// ScopeScreen lets the user choose between user and project scope.
type ScopeScreen struct {
	theme       Theme
	targetNames string
	cursor      int
	width       int
}

// NewScopeScreen creates a new scope selection screen.
// targetNames is a human-readable list of targets that support scopes (e.g. "Claude Code").
func NewScopeScreen(theme Theme, targetNames string) *ScopeScreen {
	return &ScopeScreen{theme: theme, targetNames: targetNames}
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
	heading := "  Install scope for targets that support it"
	if s.targetNames != "" {
		heading += " (" + s.targetNames + ")"
	}
	b.WriteString(heading + ":\n\n")

	for i, opt := range scopeOptions {
		desc := s.theme.Dim.Render(opt.Description)
		if i == s.cursor {
			label := "  \u276f " + opt.Label
			if s.width > 0 {
				b.WriteString(s.theme.Highlight.Width(s.width).Render(label + "    " + opt.Description))
			} else {
				b.WriteString(s.theme.Cursor.Render(label) + "    " + desc)
			}
		} else {
			b.WriteString("    " + opt.Label + "    " + desc)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n\n")
	b.WriteString(s.theme.Dim.Render("  Targets without scope support will use their default behavior."))

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
