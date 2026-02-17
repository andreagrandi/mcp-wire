package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
)

func TestNewScopeScreen(t *testing.T) {
	theme := NewTheme()
	screen := NewScopeScreen(theme)

	assert.Equal(t, 0, screen.Cursor())
}

func TestScopeScreen_Init(t *testing.T) {
	theme := NewTheme()
	screen := NewScopeScreen(theme)
	assert.Nil(t, screen.Init())
}

func TestScopeScreen_NavigateDown(t *testing.T) {
	theme := NewTheme()
	screen := NewScopeScreen(theme)

	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := s.(*ScopeScreen)
	assert.Equal(t, 1, updated.Cursor())
}

func TestScopeScreen_NavigateUp(t *testing.T) {
	theme := NewTheme()
	screen := NewScopeScreen(theme)

	var s Screen = screen
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated := s.(*ScopeScreen)
	assert.Equal(t, 0, updated.Cursor())
}

func TestScopeScreen_NavigateUpAtTop(t *testing.T) {
	theme := NewTheme()
	screen := NewScopeScreen(theme)

	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated := s.(*ScopeScreen)
	assert.Equal(t, 0, updated.Cursor())
}

func TestScopeScreen_NavigateDownAtBottom(t *testing.T) {
	theme := NewTheme()
	screen := NewScopeScreen(theme)

	var s Screen = screen
	for i := 0; i < 10; i++ {
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	updated := s.(*ScopeScreen)
	assert.Equal(t, len(scopeOptions)-1, updated.Cursor())
}

func TestScopeScreen_VimKeys(t *testing.T) {
	theme := NewTheme()
	screen := NewScopeScreen(theme)

	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated := s.(*ScopeScreen)
	assert.Equal(t, 1, updated.Cursor())

	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	updated = s.(*ScopeScreen)
	assert.Equal(t, 0, updated.Cursor())
}

func TestScopeScreen_EnterSelectsUser(t *testing.T) {
	theme := NewTheme()
	screen := NewScopeScreen(theme)

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	sel, ok := msg.(scopeSelectMsg)
	require.True(t, ok)
	assert.Equal(t, targetpkg.ConfigScopeUser, sel.scope)
}

func TestScopeScreen_EnterSelectsProject(t *testing.T) {
	theme := NewTheme()
	screen := NewScopeScreen(theme)

	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyDown})
	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	sel, ok := msg.(scopeSelectMsg)
	require.True(t, ok)
	assert.Equal(t, targetpkg.ConfigScopeProject, sel.scope)
}

func TestScopeScreen_EscSendsBack(t *testing.T) {
	theme := NewTheme()
	screen := NewScopeScreen(theme)

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(BackMsg)
	assert.True(t, ok)
}

func TestScopeScreen_ViewContainsOptions(t *testing.T) {
	theme := NewTheme()
	screen := NewScopeScreen(theme)

	view := screen.View()
	assert.Contains(t, view, "User")
	assert.Contains(t, view, "Project")
}

func TestScopeScreen_ViewContainsDescriptions(t *testing.T) {
	theme := NewTheme()
	screen := NewScopeScreen(theme)

	view := screen.View()
	assert.Contains(t, view, "user/global configuration")
	assert.Contains(t, view, "current project only")
}

func TestScopeScreen_StatusHints(t *testing.T) {
	theme := NewTheme()
	screen := NewScopeScreen(theme)

	hints := screen.StatusHints()
	assert.Len(t, hints, 3)

	descs := make([]string, len(hints))
	for i, h := range hints {
		descs[i] = h.Desc
	}
	assert.Contains(t, descs, "move")
	assert.Contains(t, descs, "select")
	assert.Contains(t, descs, "back")
}

func TestScopeScreen_WindowSizeMsg(t *testing.T) {
	theme := NewTheme()
	screen := NewScopeScreen(theme)

	s, _ := screen.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	updated := s.(*ScopeScreen)
	assert.Equal(t, 80, updated.width)
}
