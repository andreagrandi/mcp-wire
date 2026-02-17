package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTheme(t *testing.T) {
	theme := NewTheme()

	// Verify all styles are initialized (render without panic).
	assert.NotEmpty(t, theme.Title.Render("test"))
	assert.NotEmpty(t, theme.Active.Render("test"))
	assert.NotEmpty(t, theme.Completed.Render("test"))
	assert.NotEmpty(t, theme.Dim.Render("test"))
	assert.NotEmpty(t, theme.Warning.Render("test"))
	assert.NotEmpty(t, theme.Error.Render("test"))
	assert.NotEmpty(t, theme.Normal.Render("test"))
	assert.NotEmpty(t, theme.StatusBar.Render("test"))
	assert.NotEmpty(t, theme.StatusKey.Render("test"))
	assert.NotEmpty(t, theme.Cursor.Render("test"))
	assert.NotEmpty(t, theme.Selected.Render("test"))
	assert.NotEmpty(t, theme.BreadSep.Render("test"))
}
