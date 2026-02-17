package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderStatusBar_Empty(t *testing.T) {
	theme := NewTheme()
	result := RenderStatusBar(theme, nil, 80)
	assert.Empty(t, result)
}

func TestRenderStatusBar_SingleHint(t *testing.T) {
	theme := NewTheme()

	result := RenderStatusBar(theme, []KeyHint{
		{Key: "q", Desc: "quit"},
	}, 80)

	assert.Contains(t, result, "q")
	assert.Contains(t, result, "quit")
}

func TestRenderStatusBar_MultipleHints(t *testing.T) {
	theme := NewTheme()

	result := RenderStatusBar(theme, []KeyHint{
		{Key: "\u2191\u2193", Desc: "navigate"},
		{Key: "enter", Desc: "select"},
		{Key: "q", Desc: "quit"},
	}, 80)

	assert.Contains(t, result, "navigate")
	assert.Contains(t, result, "select")
	assert.Contains(t, result, "quit")
}
