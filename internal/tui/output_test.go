package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOutputScreen(t *testing.T) {
	theme := NewTheme()
	screen := NewOutputScreen(theme, "hello\nworld", 10)

	assert.Equal(t, 0, screen.Offset())
	assert.Nil(t, screen.Init())
}

func TestOutputScreen_ViewShowsContent(t *testing.T) {
	theme := NewTheme()
	screen := NewOutputScreen(theme, "line 1\nline 2\nline 3", 10)

	view := screen.View()
	assert.Contains(t, view, "line 1")
	assert.Contains(t, view, "line 2")
	assert.Contains(t, view, "line 3")
}

func TestOutputScreen_AnyKeyReturns(t *testing.T) {
	theme := NewTheme()
	screen := NewOutputScreen(theme, "short text", 10)

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(BackMsg)
	assert.True(t, ok)
}

func TestOutputScreen_AnyKeyReturnsWhenNotScrollable(t *testing.T) {
	theme := NewTheme()
	screen := NewOutputScreen(theme, "short", 10)

	// Even arrow keys return when content fits.
	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyUp})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(BackMsg)
	assert.True(t, ok)
}

func TestOutputScreen_ScrollDown(t *testing.T) {
	theme := NewTheme()
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "line"
	}

	screen := NewOutputScreen(theme, strings.Join(lines, "\n"), 5)

	newScreen, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Nil(t, cmd) // scroll keys don't produce commands
	updated := newScreen.(*OutputScreen)
	assert.Equal(t, 1, updated.Offset())
}

func TestOutputScreen_ScrollUp(t *testing.T) {
	theme := NewTheme()
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "line"
	}

	screen := NewOutputScreen(theme, strings.Join(lines, "\n"), 5)

	// Scroll down first.
	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyDown})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := s.(*OutputScreen)
	assert.Equal(t, 2, updated.Offset())

	// Scroll back up.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated = s.(*OutputScreen)
	assert.Equal(t, 1, updated.Offset())
}

func TestOutputScreen_ScrollAtTopStays(t *testing.T) {
	theme := NewTheme()
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "line"
	}

	screen := NewOutputScreen(theme, strings.Join(lines, "\n"), 5)

	// Already at top, scroll up does not go negative.
	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated := s.(*OutputScreen)
	assert.Equal(t, 0, updated.Offset())
}

func TestOutputScreen_ScrollAtBottomStops(t *testing.T) {
	theme := NewTheme()
	lines := make([]string, 7)
	for i := range lines {
		lines[i] = "line"
	}

	screen := NewOutputScreen(theme, strings.Join(lines, "\n"), 5)
	// maxOffset = 7 - 5 = 2

	var s Screen = screen
	for i := 0; i < 10; i++ {
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	updated := s.(*OutputScreen)
	assert.Equal(t, 2, updated.Offset())
}

func TestOutputScreen_VimScrollKeys(t *testing.T) {
	theme := NewTheme()
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "line"
	}

	screen := NewOutputScreen(theme, strings.Join(lines, "\n"), 5)

	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated := s.(*OutputScreen)
	assert.Equal(t, 1, updated.Offset())

	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	updated = s.(*OutputScreen)
	assert.Equal(t, 0, updated.Offset())
}

func TestOutputScreen_NonScrollKeyReturnsWhenScrollable(t *testing.T) {
	theme := NewTheme()
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "line"
	}

	screen := NewOutputScreen(theme, strings.Join(lines, "\n"), 5)

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(BackMsg)
	assert.True(t, ok)
}

func TestOutputScreen_WindowResize(t *testing.T) {
	theme := NewTheme()
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "line"
	}

	screen := NewOutputScreen(theme, strings.Join(lines, "\n"), 5)

	// Scroll down to offset 10.
	var s Screen = screen
	for i := 0; i < 15; i++ {
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	updated := s.(*OutputScreen)
	assert.Equal(t, 15, updated.Offset())

	// Resize to larger terminal → viewHeight = 20 - ChromeLines = 17, maxOffset = 3.
	s, _ = updated.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	updated = s.(*OutputScreen)
	assert.Equal(t, 3, updated.Offset()) // clamped: 20 lines - 17 viewHeight = 3
}

func TestOutputScreen_StatusHintsScrollable(t *testing.T) {
	theme := NewTheme()
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "line"
	}

	screen := NewOutputScreen(theme, strings.Join(lines, "\n"), 5)
	hints := screen.StatusHints()
	assert.Len(t, hints, 2)
	assert.Equal(t, "scroll", hints[0].Desc)
	assert.Equal(t, "return to menu", hints[1].Desc)
}

func TestOutputScreen_StatusHintsNotScrollable(t *testing.T) {
	theme := NewTheme()
	screen := NewOutputScreen(theme, "short", 10)
	hints := screen.StatusHints()
	assert.Len(t, hints, 1)
	assert.Equal(t, "return to menu", hints[0].Desc)
}

func TestOutputScreen_ScrollIndicator(t *testing.T) {
	theme := NewTheme()
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = "line"
	}

	screen := NewOutputScreen(theme, strings.Join(lines, "\n"), 5)
	view := screen.View()
	assert.Contains(t, view, "6 more")
}

func TestOutputScreen_EmptyContent(t *testing.T) {
	theme := NewTheme()
	screen := NewOutputScreen(theme, "", 10)

	view := screen.View()
	assert.NotEmpty(t, view)

	// Any key returns.
	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
}

func TestItoa(t *testing.T) {
	assert.Equal(t, "0", itoa(0))
	assert.Equal(t, "1", itoa(1))
	assert.Equal(t, "10", itoa(10))
	assert.Equal(t, "42", itoa(42))
	assert.Equal(t, "100", itoa(100))
	assert.Equal(t, "999", itoa(999))
}
