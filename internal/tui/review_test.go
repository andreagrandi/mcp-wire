package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andreagrandi/mcp-wire/internal/catalog"
	"github.com/andreagrandi/mcp-wire/internal/service"
	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
)

func testReviewState() WizardState {
	return WizardState{
		Action: "install",
		Source: "curated",
		Entry:  catalog.FromCurated(service.Service{Name: "sentry", Description: "Error tracking"}),
		Targets: []targetpkg.Target{
			&mockTarget{name: "Claude Code", slug: "claude", installed: true},
			&mockTarget{name: "Codex", slug: "codex", installed: true},
		},
		Scope: targetpkg.ConfigScopeUser,
	}
}

func TestNewReviewScreen(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)

	assert.Equal(t, 1, screen.Cursor()) // default to Apply
}

func TestReviewScreen_Init(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)
	assert.Nil(t, screen.Init())
}

func TestReviewScreen_NavigateLeft(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)

	// Default cursor=1 (Apply), move left to Cancel.
	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyLeft})
	updated := s.(*ReviewScreen)
	assert.Equal(t, 0, updated.Cursor())
}

func TestReviewScreen_NavigateRight(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)

	// Move left then right.
	var s Screen = screen
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyLeft})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated := s.(*ReviewScreen)
	assert.Equal(t, 1, updated.Cursor())
}

func TestReviewScreen_NavigateLeftAtStart(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)

	var s Screen = screen
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyLeft})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyLeft})
	updated := s.(*ReviewScreen)
	assert.Equal(t, 0, updated.Cursor())
}

func TestReviewScreen_NavigateRightAtEnd(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)

	var s Screen = screen
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRight})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated := s.(*ReviewScreen)
	assert.Equal(t, 1, updated.Cursor())
}

func TestReviewScreen_VimKeys(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)

	// 'h' moves left.
	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	updated := s.(*ReviewScreen)
	assert.Equal(t, 0, updated.Cursor())

	// 'l' moves right.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	updated = s.(*ReviewScreen)
	assert.Equal(t, 1, updated.Cursor())
}

func TestReviewScreen_EnterConfirmsApply(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)

	// Default cursor=1 (Apply).
	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	confirm, ok := msg.(reviewConfirmMsg)
	require.True(t, ok)
	assert.True(t, confirm.confirmed)
}

func TestReviewScreen_EnterConfirmsCancel(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)

	// Move to Cancel.
	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyLeft})

	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	confirm, ok := msg.(reviewConfirmMsg)
	require.True(t, ok)
	assert.False(t, confirm.confirmed)
}

func TestReviewScreen_EscSendsBack(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(BackMsg)
	assert.True(t, ok)
}

func TestReviewScreen_ViewShowsAction(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)

	view := screen.View()
	assert.Contains(t, view, "Install")
}

func TestReviewScreen_ViewShowsUninstallAction(t *testing.T) {
	theme := NewTheme()
	state := testReviewState()
	state.Action = "uninstall"
	screen := NewReviewScreen(theme, state, false)

	view := screen.View()
	assert.Contains(t, view, "Uninstall")
}

func TestReviewScreen_ViewShowsServiceName(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)

	view := screen.View()
	assert.Contains(t, view, "sentry")
	assert.Contains(t, view, "Error tracking")
}

func TestReviewScreen_ViewShowsTargets(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)

	view := screen.View()
	assert.Contains(t, view, "Claude Code")
	assert.Contains(t, view, "Codex")
}

func TestReviewScreen_ViewShowsSourceWhenRegistryEnabled(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), true)

	view := screen.View()
	assert.Contains(t, view, "Curated")
}

func TestReviewScreen_ViewHidesSourceWhenRegistryDisabled(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)

	view := screen.View()
	assert.NotContains(t, view, "Curated")
}

func TestReviewScreen_ViewShowsScopeForSupportedTargets(t *testing.T) {
	theme := NewTheme()
	state := testReviewState()
	state.Targets = []targetpkg.Target{
		&mockTarget{
			name: "Claude Code", slug: "claude", installed: true,
			scopes: []targetpkg.ConfigScope{targetpkg.ConfigScopeUser, targetpkg.ConfigScopeProject},
		},
	}
	state.Scope = targetpkg.ConfigScopeProject
	screen := NewReviewScreen(theme, state, false)

	view := screen.View()
	assert.Contains(t, view, "project")
}

func TestReviewScreen_ViewHidesScopeForUnsupportedTargets(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)

	view := screen.View()
	// Targets don't support scopes, so Scope line should not appear.
	assert.NotContains(t, view, "Scope:")
}

func TestReviewScreen_ViewShowsCredentialsForInstall(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)

	view := screen.View()
	assert.Contains(t, view, "prompt as needed")
}

func TestReviewScreen_ViewHidesCredentialsForUninstall(t *testing.T) {
	theme := NewTheme()
	state := testReviewState()
	state.Action = "uninstall"
	screen := NewReviewScreen(theme, state, false)

	view := screen.View()
	assert.NotContains(t, view, "Credentials")
}

func TestReviewScreen_ViewShowsEquivalentCommand(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)

	view := screen.View()
	assert.Contains(t, view, "mcp-wire install sentry --target claude --target codex")
}

func TestReviewScreen_ViewShowsEquivalentUninstallCommand(t *testing.T) {
	theme := NewTheme()
	state := testReviewState()
	state.Action = "uninstall"
	screen := NewReviewScreen(theme, state, false)

	view := screen.View()
	assert.Contains(t, view, "mcp-wire uninstall sentry --target claude --target codex")
}

func TestReviewScreen_ViewShowsScopeInCommand(t *testing.T) {
	theme := NewTheme()
	state := testReviewState()
	state.Scope = targetpkg.ConfigScopeProject
	screen := NewReviewScreen(theme, state, false)

	view := screen.View()
	assert.Contains(t, view, "--scope project")
}

func TestReviewScreen_ViewNoScopeInCommandForUserScope(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)

	view := screen.View()
	assert.NotContains(t, view, "--scope")
}

func TestReviewScreen_ViewShowsChoices(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)

	view := screen.View()
	assert.Contains(t, view, "Cancel")
	assert.Contains(t, view, "Apply")
	assert.Contains(t, view, "Apply changes?")
}

func TestReviewScreen_StatusHints(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)

	hints := screen.StatusHints()
	assert.Len(t, hints, 3)

	descs := make([]string, len(hints))
	for i, h := range hints {
		descs[i] = h.Desc
	}
	assert.Contains(t, descs, "choose")
	assert.Contains(t, descs, "confirm")
	assert.Contains(t, descs, "back")
}

func TestReviewScreen_WindowSizeMsg(t *testing.T) {
	theme := NewTheme()
	screen := NewReviewScreen(theme, testReviewState(), false)

	s, _ := screen.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	updated := s.(*ReviewScreen)
	assert.Equal(t, 100, updated.width)
}

func TestScopeLabel(t *testing.T) {
	assert.Equal(t, "user", scopeLabel(targetpkg.ConfigScopeUser))
	assert.Equal(t, "project", scopeLabel(targetpkg.ConfigScopeProject))
	assert.Equal(t, "effective", scopeLabel(targetpkg.ConfigScope("effective")))
}
