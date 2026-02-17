package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWizardModel(t *testing.T) {
	model := NewWizardModel()

	assert.NotNil(t, model.screen)
	assert.Empty(t, model.steps)
	assert.Equal(t, 0, model.width)
	assert.Equal(t, 0, model.height)
}

func TestWizardModel_Init(t *testing.T) {
	model := NewWizardModel()
	assert.Nil(t, model.Init())
}

func TestWizardModel_WindowSizeMsg(t *testing.T) {
	model := NewWizardModel()

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	wm := updated.(WizardModel)

	assert.Equal(t, 120, wm.width)
	assert.Equal(t, 40, wm.height)
}

func TestWizardModel_CtrlCQuits(t *testing.T) {
	model := NewWizardModel()

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestWizardModel_ExitMenuQuits(t *testing.T) {
	model := NewWizardModel()

	_, cmd := model.Update(menuSelectMsg{item: "Exit"})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestWizardModel_UnhandledMenuActionNoOp(t *testing.T) {
	model := NewWizardModel()

	updated, cmd := model.Update(menuSelectMsg{item: "Status"})
	wm := updated.(WizardModel)

	assert.Nil(t, cmd)
	assert.NotNil(t, wm.screen)
}

func TestWizardModel_BackMsg(t *testing.T) {
	model := NewWizardModel()
	model.steps = []BreadcrumbStep{
		{Label: "Source", Active: true, Visible: true},
	}

	updated, _ := model.Update(BackMsg{})
	wm := updated.(WizardModel)

	assert.Nil(t, wm.steps)
	_, isMenu := wm.screen.(*MenuScreen)
	assert.True(t, isMenu)
}

func TestWizardModel_ViewLayout(t *testing.T) {
	model := NewWizardModel()
	model.width = 80
	model.height = 20

	view := model.View()

	assert.Contains(t, view, "mcp-wire")
	assert.Contains(t, view, "Install service")
	assert.Contains(t, view, "navigate")
}

func TestWizardModel_ContentHeight(t *testing.T) {
	model := NewWizardModel()

	// Unknown height returns default.
	assert.Equal(t, ContentHeight, model.contentHeight())

	// Known height subtracts title + status bar.
	model.height = 30
	assert.Equal(t, 28, model.contentHeight())

	// Very small terminal.
	model.height = 2
	assert.Equal(t, 1, model.contentHeight())
}

func TestWizardModel_ViewNoBreadcrumbOnMenu(t *testing.T) {
	model := NewWizardModel()
	model.width = 80
	model.height = 20

	view := model.View()

	// No breadcrumb separator on main menu.
	assert.NotContains(t, view, "\u203a")
}

func TestPadToHeight(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		target   int
		wantLines int
	}{
		{"empty to 5", "", 5, 5},
		{"one line to 5", "hello", 5, 5},
		{"trailing newline", "hello\n", 5, 5},
		{"exact match", "a\nb\nc", 3, 3},
		{"truncate", "a\nb\nc\nd\ne", 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padToHeight(tt.content, tt.target)
			lines := len(splitLines(result))
			assert.Equal(t, tt.wantLines, lines)
		})
	}
}

func splitLines(s string) []string {
	if s == "" {
		return []string{""}
	}

	lines := []string{}
	start := 0

	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}

	lines = append(lines, s[start:])
	return lines
}
