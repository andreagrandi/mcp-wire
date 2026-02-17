package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// OutputScreen displays pre-rendered text with optional scrolling.
// Any key other than scroll keys returns to the previous screen.
type OutputScreen struct {
	theme      Theme
	lines      []string
	offset     int
	viewHeight int
}

// NewOutputScreen creates a screen that displays content text.
func NewOutputScreen(theme Theme, content string, viewHeight int) *OutputScreen {
	trimmed := strings.TrimRight(content, "\n")

	var lines []string
	if trimmed == "" {
		lines = []string{""}
	} else {
		lines = strings.Split(trimmed, "\n")
	}

	return &OutputScreen{
		theme:      theme,
		lines:      lines,
		viewHeight: viewHeight,
	}
}

func (o *OutputScreen) Init() tea.Cmd { return nil }

func (o *OutputScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		o.viewHeight = contentHeightFromTerminal(msg.Height)
		o.clampOffset()
		return o, nil

	case tea.KeyMsg:
		key := msg.String()

		if o.scrollable() {
			switch key {
			case "up", "k":
				if o.offset > 0 {
					o.offset--
				}
				return o, nil
			case "down", "j":
				if max := o.maxOffset(); o.offset < max {
					o.offset++
				}
				return o, nil
			}
		}

		return o, func() tea.Msg { return BackMsg{} }
	}

	return o, nil
}

func (o *OutputScreen) View() string {
	viewLines := o.viewHeight

	// Reserve a line for the scroll indicator when there is more content below.
	hasMore := o.offset+viewLines < len(o.lines)
	if hasMore {
		viewLines--
	}

	end := o.offset + viewLines
	if end > len(o.lines) {
		end = len(o.lines)
	}

	visible := o.lines[o.offset:end]

	var b strings.Builder
	for _, line := range visible {
		b.WriteString(line)
		b.WriteByte('\n')
	}

	if hasMore {
		remaining := len(o.lines) - end
		b.WriteString(o.theme.Dim.Render("  \u25bc " + strings.Repeat(".", 3) + " " + itoa(remaining) + " more"))
	}

	return b.String()
}

func (o *OutputScreen) StatusHints() []KeyHint {
	if o.scrollable() {
		return []KeyHint{
			{Key: "\u2191\u2193", Desc: "scroll"},
			{Key: "any key", Desc: "return to menu"},
		}
	}

	return []KeyHint{
		{Key: "any key", Desc: "return to menu"},
	}
}

func (o *OutputScreen) scrollable() bool {
	return len(o.lines) > o.viewHeight
}

func (o *OutputScreen) maxOffset() int {
	max := len(o.lines) - o.viewHeight
	if max < 0 {
		return 0
	}

	return max
}

func (o *OutputScreen) clampOffset() {
	if max := o.maxOffset(); o.offset > max {
		o.offset = max
	}
}

// Offset returns the current scroll offset (for testing).
func (o *OutputScreen) Offset() int {
	return o.offset
}

// itoa converts a small non-negative int to a string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	digits := make([]byte, 0, 4)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}

	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	return string(digits)
}
