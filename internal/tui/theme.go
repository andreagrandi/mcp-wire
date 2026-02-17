package tui

import "github.com/charmbracelet/lipgloss"

// ContentHeight is the default number of lines for the content area
// when the terminal height is unknown.
const ContentHeight = 13

// ChromeLines is the number of fixed lines used by the layout frame
// (title bar + separator + status bar).
const ChromeLines = 3

// Theme holds Lip Gloss styles for the TUI.
type Theme struct {
	Title     lipgloss.Style
	Active    lipgloss.Style
	Completed lipgloss.Style
	Dim       lipgloss.Style
	Warning   lipgloss.Style
	Error     lipgloss.Style
	Normal    lipgloss.Style
	StatusBar lipgloss.Style
	StatusKey lipgloss.Style
	Cursor    lipgloss.Style
	Selected  lipgloss.Style
	BreadSep  lipgloss.Style
	Highlight lipgloss.Style
	Separator lipgloss.Style
}

// NewTheme creates a Theme with the default color palette.
func NewTheme() Theme {
	cyan := lipgloss.Color("6")
	green := lipgloss.Color("2")
	yellow := lipgloss.Color("3")
	red := lipgloss.Color("1")
	dim := lipgloss.Color("8")
	blue := lipgloss.Color("4")

	return Theme{
		Title:     lipgloss.NewStyle().Bold(true),
		Active:    lipgloss.NewStyle().Bold(true).Foreground(cyan),
		Completed: lipgloss.NewStyle().Foreground(green),
		Dim:       lipgloss.NewStyle().Foreground(dim),
		Warning:   lipgloss.NewStyle().Foreground(yellow),
		Error:     lipgloss.NewStyle().Foreground(red),
		Normal:    lipgloss.NewStyle(),
		StatusBar: lipgloss.NewStyle().Foreground(dim),
		StatusKey: lipgloss.NewStyle().Bold(true).Foreground(dim),
		Cursor:    lipgloss.NewStyle().Bold(true).Foreground(cyan),
		Selected:  lipgloss.NewStyle().Foreground(green),
		BreadSep:  lipgloss.NewStyle().Foreground(dim),
		Highlight: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(blue),
		Separator: lipgloss.NewStyle().Foreground(blue),
	}
}
