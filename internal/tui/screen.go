package tui

import tea "github.com/charmbracelet/bubbletea"

// ScreenID identifies a wizard screen.
type ScreenID int

const (
	ScreenMenu ScreenID = iota
	ScreenSource
	ScreenService
	ScreenTrust
	ScreenTarget
	ScreenScope
	ScreenReview
	ScreenApply
)

// KeyHint describes a keybinding shown in the status bar.
type KeyHint struct {
	Key  string
	Desc string
}

// Screen defines the interface each wizard screen must implement.
type Screen interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (Screen, tea.Cmd)
	View() string
	StatusHints() []KeyHint
}

// NavigateMsg requests navigation to a new screen.
type NavigateMsg struct {
	Screen ScreenID
}

// BackMsg requests navigation to the previous screen.
type BackMsg struct{}

// menuSelectMsg is sent when a main menu item is selected.
type menuSelectMsg struct {
	item string
}

// sourceSelectMsg is sent when a source is selected.
type sourceSelectMsg struct {
	source string // "curated", "registry", "all"
}
