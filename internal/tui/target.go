package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
)

// targetItem holds display data for one target in the multi-select list.
type targetItem struct {
	target    targetpkg.Target
	installed bool
	checked   bool
}

// TargetScreen shows a multi-select checkbox list of targets.
type TargetScreen struct {
	theme  Theme
	items  []targetItem
	cursor int
	width  int
}

// NewTargetScreen creates a target multi-select screen.
// allTargets is the full list of known targets. If preSelected is non-empty,
// those targets are pre-checked; otherwise none are pre-checked.
func NewTargetScreen(theme Theme, allTargets []targetpkg.Target, preSelected []targetpkg.Target) *TargetScreen {
	// Sort: installed first, then by slug.
	sorted := make([]targetpkg.Target, len(allTargets))
	copy(sorted, allTargets)
	sort.Slice(sorted, func(i, j int) bool {
		li := sorted[i].IsInstalled()
		lj := sorted[j].IsInstalled()
		if li != lj {
			return li
		}
		return sorted[i].Slug() < sorted[j].Slug()
	})

	preSelectedSet := make(map[string]struct{}, len(preSelected))
	for _, t := range preSelected {
		preSelectedSet[t.Slug()] = struct{}{}
	}

	items := make([]targetItem, len(sorted))
	for i, t := range sorted {
		installed := t.IsInstalled()
		var checked bool
		if len(preSelected) > 0 {
			_, checked = preSelectedSet[t.Slug()]
		}
		items[i] = targetItem{
			target:    t,
			installed: installed,
			checked:   checked,
		}
	}

	return &TargetScreen{
		theme: theme,
		items: items,
	}
}

func (t *TargetScreen) Init() tea.Cmd { return nil }

func (t *TargetScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		return t, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			t.moveCursorUp()
		case "down", "j":
			t.moveCursorDown()
		case " ":
			t.toggleCurrent()
		case "a":
			t.selectAllInstalled()
		case "n":
			t.selectNone()
		case "enter":
			return t.confirm()
		case "esc":
			return t, func() tea.Msg { return BackMsg{} }
		}
	}

	return t, nil
}

func (t *TargetScreen) moveCursorUp() {
	if t.cursor > 0 {
		t.cursor--
	}
}

func (t *TargetScreen) moveCursorDown() {
	if t.cursor < len(t.items)-1 {
		t.cursor++
	}
}

func (t *TargetScreen) toggleCurrent() {
	if t.cursor >= 0 && t.cursor < len(t.items) && t.items[t.cursor].installed {
		t.items[t.cursor].checked = !t.items[t.cursor].checked
	}
}

func (t *TargetScreen) selectAllInstalled() {
	for i := range t.items {
		if t.items[i].installed {
			t.items[i].checked = true
		}
	}
}

func (t *TargetScreen) selectNone() {
	for i := range t.items {
		t.items[i].checked = false
	}
}

func (t *TargetScreen) confirm() (Screen, tea.Cmd) {
	selected := t.selectedTargets()
	if len(selected) == 0 {
		return t, nil
	}

	targets := selected
	return t, func() tea.Msg {
		return targetSelectMsg{targets: targets}
	}
}

func (t *TargetScreen) selectedTargets() []targetpkg.Target {
	var selected []targetpkg.Target
	for _, item := range t.items {
		if item.checked {
			selected = append(selected, item.target)
		}
	}
	return selected
}

func (t *TargetScreen) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  Select targets:\n\n")

	for i, item := range t.items {
		check := "[ ]"
		if item.checked {
			check = "[x]"
		}

		label := item.target.Name() + " (" + item.target.Slug() + ")"

		if !item.installed {
			line := "    " + check + " " + label + " \u2014 not installed"
			b.WriteString(t.theme.Dim.Render(line))
		} else if i == t.cursor {
			line := "  \u276f " + check + " " + label
			if t.width > 0 {
				b.WriteString(t.theme.Highlight.Width(t.width).Render(line))
			} else {
				b.WriteString(t.theme.Cursor.Render(line))
			}
		} else {
			if item.checked {
				b.WriteString("    " + t.theme.Selected.Render(check) + " " + label)
			} else {
				b.WriteString("    " + check + " " + label)
			}
		}

		b.WriteString("\n")
	}

	count := len(t.selectedTargets())
	b.WriteString("\n")
	if count == 0 {
		b.WriteString(t.theme.Warning.Render("  Select at least one target"))
	} else {
		b.WriteString(t.theme.Dim.Render(fmt.Sprintf("  %d target(s) selected", count)))
	}

	return b.String()
}

func (t *TargetScreen) StatusHints() []KeyHint {
	return []KeyHint{
		{Key: "\u2191\u2193", Desc: "move"},
		{Key: "Space", Desc: "toggle"},
		{Key: "a", Desc: "all"},
		{Key: "n", Desc: "none"},
		{Key: "Enter", Desc: "confirm"},
		{Key: "Esc", Desc: "back"},
	}
}

// Cursor returns the current cursor position (for testing).
func (t *TargetScreen) Cursor() int { return t.cursor }

// Items returns the target items (for testing).
func (t *TargetScreen) Items() []targetItem { return t.items }
