package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andreagrandi/mcp-wire/internal/service"
	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
)

// mockTarget implements target.Target for testing.
type mockTarget struct {
	name      string
	slug      string
	installed bool
	scopes    []targetpkg.ConfigScope
}

func (m *mockTarget) Name() string      { return m.name }
func (m *mockTarget) Slug() string      { return m.slug }
func (m *mockTarget) IsInstalled() bool { return m.installed }
func (m *mockTarget) Install(_ service.Service, _ map[string]string) error {
	return nil
}
func (m *mockTarget) Uninstall(_ string) error                 { return nil }
func (m *mockTarget) List() ([]string, error)                  { return nil, nil }
func (m *mockTarget) SupportedScopes() []targetpkg.ConfigScope { return m.scopes }
func (m *mockTarget) InstallWithScope(_ service.Service, _ map[string]string, _ targetpkg.ConfigScope) error {
	return nil
}
func (m *mockTarget) UninstallWithScope(_ string, _ targetpkg.ConfigScope) error { return nil }
func (m *mockTarget) ListWithScope(_ targetpkg.ConfigScope) ([]string, error)    { return nil, nil }

func testTargets() []targetpkg.Target {
	return []targetpkg.Target{
		&mockTarget{name: "Claude Code", slug: "claude", installed: true},
		&mockTarget{name: "Codex", slug: "codex", installed: true},
		&mockTarget{name: "Gemini CLI", slug: "geminicli", installed: false},
	}
}

func testTargetsWithScopes() []targetpkg.Target {
	return []targetpkg.Target{
		&mockTarget{
			name: "Claude Code", slug: "claude", installed: true,
			scopes: []targetpkg.ConfigScope{targetpkg.ConfigScopeUser, targetpkg.ConfigScopeProject},
		},
		&mockTarget{name: "Codex", slug: "codex", installed: true},
	}
}

func TestNewTargetScreen(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	assert.Equal(t, 0, screen.Cursor())
	assert.Len(t, screen.Items(), 3)
}

func TestNewTargetScreen_InstalledFirst(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	items := screen.Items()
	// Installed targets should come first.
	assert.True(t, items[0].installed)
	assert.True(t, items[1].installed)
	assert.False(t, items[2].installed)
}

func TestNewTargetScreen_InstalledPreChecked(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	items := screen.Items()
	assert.True(t, items[0].checked)  // installed → checked
	assert.True(t, items[1].checked)  // installed → checked
	assert.False(t, items[2].checked) // not installed → not checked
}

func TestNewTargetScreen_PreSelected(t *testing.T) {
	theme := NewTheme()
	targets := testTargets()
	// Pre-select only claude.
	preSelected := []targetpkg.Target{targets[0]}
	screen := NewTargetScreen(theme, targets, preSelected)

	items := screen.Items()
	var claudeChecked, codexChecked bool
	for _, item := range items {
		if item.target.Slug() == "claude" {
			claudeChecked = item.checked
		}
		if item.target.Slug() == "codex" {
			codexChecked = item.checked
		}
	}
	assert.True(t, claudeChecked)
	assert.False(t, codexChecked)
}

func TestTargetScreen_Init(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)
	assert.Nil(t, screen.Init())
}

func TestTargetScreen_NavigateDown(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := s.(*TargetScreen)
	assert.Equal(t, 1, updated.Cursor())
}

func TestTargetScreen_NavigateUp(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	var s Screen = screen
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated := s.(*TargetScreen)
	assert.Equal(t, 0, updated.Cursor())
}

func TestTargetScreen_NavigateUpAtTop(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated := s.(*TargetScreen)
	assert.Equal(t, 0, updated.Cursor())
}

func TestTargetScreen_NavigateDownAtBottom(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	var s Screen = screen
	for i := 0; i < 10; i++ {
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	updated := s.(*TargetScreen)
	assert.Equal(t, len(screen.Items())-1, updated.Cursor())
}

func TestTargetScreen_VimKeys(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	// j moves down.
	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated := s.(*TargetScreen)
	assert.Equal(t, 1, updated.Cursor())

	// k moves up.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	updated = s.(*TargetScreen)
	assert.Equal(t, 0, updated.Cursor())
}

func TestTargetScreen_SpaceTogglesOn(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	// First uncheck with 'n', then toggle back on.
	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	assert.False(t, screen.Items()[0].checked)

	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	assert.True(t, screen.Items()[0].checked)
}

func TestTargetScreen_SpaceTogglesOff(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	// First item is installed and pre-checked. Toggle it off.
	_, _ = screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	assert.False(t, screen.Items()[0].checked)
}

func TestTargetScreen_SpaceIgnoresNotInstalled(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	// Navigate to the not-installed target (last item).
	var s Screen = screen
	for i := 0; i < len(screen.Items())-1; i++ {
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	// Try to toggle — should remain unchecked.
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	updated := s.(*TargetScreen)
	lastItem := updated.Items()[len(updated.Items())-1]
	assert.False(t, lastItem.checked)
}

func TestTargetScreen_SelectAllInstalled(t *testing.T) {
	theme := NewTheme()
	// Start with nothing selected.
	screen := NewTargetScreen(theme, testTargets(), []targetpkg.Target{})

	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	for _, item := range screen.Items() {
		if item.installed {
			assert.True(t, item.checked, "installed target %s should be checked", item.target.Slug())
		} else {
			assert.False(t, item.checked, "not-installed target %s should not be checked", item.target.Slug())
		}
	}
}

func TestTargetScreen_SelectNone(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil) // all installed pre-checked

	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	for _, item := range screen.Items() {
		assert.False(t, item.checked, "target %s should not be checked", item.target.Slug())
	}
}

func TestTargetScreen_EnterConfirmsSelection(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	sel, ok := msg.(targetSelectMsg)
	require.True(t, ok)
	assert.Len(t, sel.targets, 2) // two installed targets
}

func TestTargetScreen_EnterDoesNothingWithNoSelection(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	// Deselect all.
	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd) // no confirmation with 0 selected
}

func TestTargetScreen_EscSendsBack(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(BackMsg)
	assert.True(t, ok)
}

func TestTargetScreen_ViewContainsTargetNames(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	view := screen.View()
	assert.Contains(t, view, "Claude Code")
	assert.Contains(t, view, "Codex")
	assert.Contains(t, view, "Gemini CLI")
}

func TestTargetScreen_ViewShowsCheckboxes(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	view := screen.View()
	assert.Contains(t, view, "[x]")
}

func TestTargetScreen_ViewShowsNotInstalled(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	view := screen.View()
	assert.Contains(t, view, "not installed")
}

func TestTargetScreen_ViewShowsSelectedCount(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	view := screen.View()
	assert.Contains(t, view, "2 target(s) selected")
}

func TestTargetScreen_ViewShowsWarningWhenNoneSelected(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	view := screen.View()
	assert.Contains(t, view, "Select at least one target")
}

func TestTargetScreen_ViewShowsSelectTargetsHeader(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	view := screen.View()
	assert.Contains(t, view, "Select targets:")
}

func TestTargetScreen_StatusHints(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	hints := screen.StatusHints()
	assert.Len(t, hints, 6)

	descs := make([]string, len(hints))
	for i, h := range hints {
		descs[i] = h.Desc
	}
	assert.Contains(t, descs, "move")
	assert.Contains(t, descs, "toggle")
	assert.Contains(t, descs, "all")
	assert.Contains(t, descs, "none")
	assert.Contains(t, descs, "confirm")
	assert.Contains(t, descs, "back")
}

func TestTargetScreen_WindowSizeMsg(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	s, _ := screen.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	updated := s.(*TargetScreen)
	assert.Equal(t, 100, updated.width)
}

func TestTargetScreen_EmptyTargetList(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, nil, nil)

	assert.Len(t, screen.Items(), 0)
	view := screen.View()
	assert.Contains(t, view, "Select at least one target")
}

func TestTargetScreen_SlugsInView(t *testing.T) {
	theme := NewTheme()
	screen := NewTargetScreen(theme, testTargets(), nil)

	view := screen.View()
	assert.Contains(t, view, "(claude)")
	assert.Contains(t, view, "(codex)")
	assert.Contains(t, view, "(geminicli)")
}
