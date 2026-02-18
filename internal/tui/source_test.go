package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSourceScreen(t *testing.T) {
	theme := NewTheme()
	screen := NewSourceScreen(theme)

	assert.Equal(t, 0, screen.Cursor())
}

func TestSourceScreen_Init(t *testing.T) {
	theme := NewTheme()
	screen := NewSourceScreen(theme)

	assert.Nil(t, screen.Init())
}

func TestSourceScreen_NavigateDown(t *testing.T) {
	theme := NewTheme()
	screen := NewSourceScreen(theme)

	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := s.(*SourceScreen)

	assert.Equal(t, 1, updated.Cursor())
}

func TestSourceScreen_NavigateUp(t *testing.T) {
	theme := NewTheme()
	screen := NewSourceScreen(theme)

	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyDown})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated := s.(*SourceScreen)

	assert.Equal(t, 1, updated.Cursor())
}

func TestSourceScreen_NavigateUpAtTop(t *testing.T) {
	theme := NewTheme()
	screen := NewSourceScreen(theme)

	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated := s.(*SourceScreen)

	assert.Equal(t, 0, updated.Cursor())
}

func TestSourceScreen_NavigateDownAtBottom(t *testing.T) {
	theme := NewTheme()
	screen := NewSourceScreen(theme)

	var s Screen = screen
	for i := 0; i < 10; i++ {
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	updated := s.(*SourceScreen)
	assert.Equal(t, len(sourceOptions)-1, updated.Cursor())
}

func TestSourceScreen_VimKeys(t *testing.T) {
	theme := NewTheme()
	screen := NewSourceScreen(theme)

	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated := s.(*SourceScreen)
	assert.Equal(t, 1, updated.Cursor())

	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	updated = s.(*SourceScreen)
	assert.Equal(t, 0, updated.Cursor())
}

func TestSourceScreen_EnterSelectsCurated(t *testing.T) {
	theme := NewTheme()
	screen := NewSourceScreen(theme)

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	sel, ok := msg.(sourceSelectMsg)
	require.True(t, ok)
	assert.Equal(t, "curated", sel.source)
}

func TestSourceScreen_EnterSelectsRegistry(t *testing.T) {
	theme := NewTheme()
	screen := NewSourceScreen(theme)

	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyDown})
	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	sel, ok := msg.(sourceSelectMsg)
	require.True(t, ok)
	assert.Equal(t, "registry", sel.source)
}

func TestSourceScreen_EnterSelectsBoth(t *testing.T) {
	theme := NewTheme()
	screen := NewSourceScreen(theme)

	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyDown})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	sel, ok := msg.(sourceSelectMsg)
	require.True(t, ok)
	assert.Equal(t, "all", sel.source)
}

func TestSourceScreen_EscSendsBack(t *testing.T) {
	theme := NewTheme()
	screen := NewSourceScreen(theme)

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(BackMsg)
	assert.True(t, ok)
}

func TestSourceScreen_ViewContainsOptions(t *testing.T) {
	theme := NewTheme()
	screen := NewSourceScreen(theme)

	view := screen.View()
	assert.Contains(t, view, "Curated services")
	assert.Contains(t, view, "Registry services")
	assert.Contains(t, view, "Both")
	assert.Contains(t, view, "recommended")
}

func TestSourceScreen_ViewContainsDescriptions(t *testing.T) {
	theme := NewTheme()
	screen := NewSourceScreen(theme)

	view := screen.View()
	assert.Contains(t, view, "recommended, maintained by mcp-wire")
	assert.Contains(t, view, "community-published MCP servers")
	assert.Contains(t, view, "curated + registry combined")
}

func TestSourceScreen_ViewContainsQuestionHeader(t *testing.T) {
	theme := NewTheme()
	screen := NewSourceScreen(theme)

	view := screen.View()
	assert.Contains(t, view, "Where should mcp-wire look for services?")
}

func TestSourceScreen_StatusHints(t *testing.T) {
	theme := NewTheme()
	screen := NewSourceScreen(theme)

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

func TestSourceScreen_WindowSizeMsg(t *testing.T) {
	theme := NewTheme()
	screen := NewSourceScreen(theme)

	s, _ := screen.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	updated := s.(*SourceScreen)

	assert.Equal(t, 80, updated.width)
}
