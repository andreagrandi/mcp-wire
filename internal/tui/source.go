package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// sourceOption describes one entry in the source selection screen.
type sourceOption struct {
	Label       string
	Description string
	Value       string // "curated", "registry", "all"
}

var sourceOptions = []sourceOption{
	{Label: "Curated services", Description: "recommended, maintained by mcp-wire", Value: "curated"},
	{Label: "Registry services", Description: "community-published MCP servers", Value: "registry"},
	{Label: "Both", Description: "curated + registry combined", Value: "all"},
}

// SourceScreen lets the user choose the service source.
type SourceScreen struct {
	theme  Theme
	cursor int
	width  int
}

// NewSourceScreen creates a new source selection screen.
func NewSourceScreen(theme Theme) *SourceScreen {
	return &SourceScreen{theme: theme}
}

func (s *SourceScreen) Init() tea.Cmd { return nil }

func (s *SourceScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
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
			if s.cursor < len(sourceOptions)-1 {
				s.cursor++
			}
		case "enter":
			opt := sourceOptions[s.cursor]
			return s, func() tea.Msg {
				return sourceSelectMsg{source: opt.Value}
			}
		case "esc":
			return s, func() tea.Msg { return BackMsg{} }
		}
	}

	return s, nil
}

func (s *SourceScreen) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  Where should mcp-wire look for services?\n\n")

	for i, opt := range sourceOptions {
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

	return b.String()
}

func (s *SourceScreen) StatusHints() []KeyHint {
	return []KeyHint{
		{Key: "\u2191\u2193", Desc: "move"},
		{Key: "Enter", Desc: "select"},
		{Key: "Esc", Desc: "back"},
	}
}

// Cursor returns the current cursor position (for testing).
func (s *SourceScreen) Cursor() int {
	return s.cursor
}
