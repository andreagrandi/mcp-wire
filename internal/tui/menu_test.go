package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMenuScreen(t *testing.T) {
	theme := NewTheme()
	menu := NewMenuScreen(theme)

	assert.Equal(t, 0, menu.Cursor())
}

func TestMenuScreen_Init(t *testing.T) {
	theme := NewTheme()
	menu := NewMenuScreen(theme)

	assert.Nil(t, menu.Init())
}

func TestMenuScreen_NavigateDown(t *testing.T) {
	theme := NewTheme()
	menu := NewMenuScreen(theme)

	screen, _ := menu.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := screen.(*MenuScreen)

	assert.Equal(t, 1, updated.Cursor())
}

func TestMenuScreen_NavigateUp(t *testing.T) {
	theme := NewTheme()
	menu := NewMenuScreen(theme)

	// Move down first, then up.
	screen, _ := menu.Update(tea.KeyMsg{Type: tea.KeyDown})
	screen, _ = screen.Update(tea.KeyMsg{Type: tea.KeyDown})
	screen, _ = screen.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated := screen.(*MenuScreen)

	assert.Equal(t, 1, updated.Cursor())
}

func TestMenuScreen_NavigateUpAtTop(t *testing.T) {
	theme := NewTheme()
	menu := NewMenuScreen(theme)

	screen, _ := menu.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated := screen.(*MenuScreen)

	assert.Equal(t, 0, updated.Cursor())
}

func TestMenuScreen_NavigateDownAtBottom(t *testing.T) {
	theme := NewTheme()
	menu := NewMenuScreen(theme)

	// Move to last item.
	var screen Screen = menu
	for i := 0; i < len(menuItems); i++ {
		screen, _ = screen.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	updated := screen.(*MenuScreen)
	assert.Equal(t, len(menuItems)-1, updated.Cursor())
}

func TestMenuScreen_VimKeys(t *testing.T) {
	theme := NewTheme()
	menu := NewMenuScreen(theme)

	screen, _ := menu.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated := screen.(*MenuScreen)
	assert.Equal(t, 1, updated.Cursor())

	screen, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	updated = screen.(*MenuScreen)
	assert.Equal(t, 0, updated.Cursor())
}

func TestMenuScreen_EnterSendsSelection(t *testing.T) {
	theme := NewTheme()
	menu := NewMenuScreen(theme)

	_, cmd := menu.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	sel, ok := msg.(menuSelectMsg)
	require.True(t, ok)
	assert.Equal(t, "Install service", sel.item)
}

func TestMenuScreen_QuitSendsQuit(t *testing.T) {
	theme := NewTheme()
	menu := NewMenuScreen(theme)

	_, cmd := menu.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd)

	// tea.Quit returns a special QuitMsg.
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestMenuScreen_ViewContainsCursor(t *testing.T) {
	theme := NewTheme()
	menu := NewMenuScreen(theme)

	view := menu.View()
	assert.Contains(t, view, "Install service")
	assert.Contains(t, view, "Exit")
}

func TestMenuScreen_StatusHints(t *testing.T) {
	theme := NewTheme()
	menu := NewMenuScreen(theme)

	hints := menu.StatusHints()
	assert.NotEmpty(t, hints)

	hintKeys := make([]string, len(hints))
	for i, h := range hints {
		hintKeys[i] = h.Desc
	}

	assert.Contains(t, hintKeys, "move")
	assert.Contains(t, hintKeys, "select")
	assert.Contains(t, hintKeys, "quit")
}
