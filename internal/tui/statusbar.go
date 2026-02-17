package tui

import "strings"

// RenderStatusBar renders keybinding hints for the bottom status bar.
func RenderStatusBar(theme Theme, hints []KeyHint, width int) string {
	var parts []string

	for _, h := range hints {
		key := theme.StatusKey.Render(h.Key)
		parts = append(parts, key+" "+h.Desc)
	}

	content := strings.Join(parts, "  ")
	return theme.StatusBar.Render(content)
}
