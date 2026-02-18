package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andreagrandi/mcp-wire/internal/catalog"
	"github.com/andreagrandi/mcp-wire/internal/service"
	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
)

func testApplyState() WizardState {
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

func testApplyService() service.Service {
	return service.Service{
		Name:        "sentry",
		Description: "Error tracking",
		Transport:   "sse",
		URL:         "https://mcp.sentry.dev/sse",
	}
}

func testApplyCallbacks() ApplyCallbacks {
	return ApplyCallbacks{
		InstallTarget: func(_ service.Service, _ map[string]string, _ targetpkg.Target, _ targetpkg.ConfigScope) error {
			return nil
		},
		UninstallTarget: func(_ string, _ targetpkg.Target, _ targetpkg.ConfigScope) error {
			return nil
		},
	}
}

func TestNewApplyScreen(t *testing.T) {
	theme := NewTheme()
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())

	results := screen.Results()
	require.Len(t, results, 2)
	assert.Equal(t, "Claude Code", results[0].name)
	assert.Equal(t, "pending", results[0].status)
	assert.Equal(t, "Codex", results[1].name)
	assert.Equal(t, "pending", results[1].status)
}

func TestApplyScreen_Init_StartsFirstTarget(t *testing.T) {
	theme := NewTheme()
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())

	cmd := screen.Init()
	require.NotNil(t, cmd)

	results := screen.Results()
	assert.Equal(t, "running", results[0].status)
	assert.Equal(t, "pending", results[1].status)
}

func TestApplyScreen_Init_EmptyTargets(t *testing.T) {
	theme := NewTheme()
	state := testApplyState()
	state.Targets = nil
	screen := NewApplyScreen(theme, state, testApplyService(), nil, testApplyCallbacks())

	cmd := screen.Init()
	assert.Nil(t, cmd)
	assert.Equal(t, applySubStateDone, screen.ApplySubState())
}

func TestApplyScreen_ResultSuccess(t *testing.T) {
	theme := NewTheme()
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())
	screen.Init()

	// First target succeeds.
	s, cmd := screen.Update(applyResultMsg{index: 0, err: nil})
	updated := s.(*ApplyScreen)

	results := updated.Results()
	assert.Equal(t, "done", results[0].status)
	assert.Equal(t, "running", results[1].status)
	assert.NotNil(t, cmd) // dispatches next target
}

func TestApplyScreen_ResultFailure(t *testing.T) {
	theme := NewTheme()
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())
	screen.Init()

	// First target fails.
	s, cmd := screen.Update(applyResultMsg{index: 0, err: errors.New("file not found")})
	updated := s.(*ApplyScreen)

	results := updated.Results()
	assert.Equal(t, "failed", results[0].status)
	assert.Equal(t, "file not found", results[0].err.Error())
	assert.Equal(t, "running", results[1].status)
	assert.NotNil(t, cmd) // still dispatches next target
}

func TestApplyScreen_AllDone(t *testing.T) {
	theme := NewTheme()
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())
	screen.Init()

	// First target.
	s, _ := screen.Update(applyResultMsg{index: 0, err: nil})
	updated := s.(*ApplyScreen)

	// Second target.
	s, _ = updated.Update(applyResultMsg{index: 1, err: nil})
	updated = s.(*ApplyScreen)

	assert.Equal(t, applySubStateDone, updated.ApplySubState())
	assert.False(t, updated.hasFailures)
}

func TestApplyScreen_PartialFailure(t *testing.T) {
	theme := NewTheme()
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())
	screen.Init()

	s, _ := screen.Update(applyResultMsg{index: 0, err: nil})
	updated := s.(*ApplyScreen)

	s, _ = updated.Update(applyResultMsg{index: 1, err: errors.New("broken")})
	updated = s.(*ApplyScreen)

	assert.Equal(t, applySubStateDone, updated.ApplySubState())
	assert.True(t, updated.hasFailures)
}

func TestApplyScreen_ViewRunning(t *testing.T) {
	theme := NewTheme()
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())
	screen.Init()

	view := screen.View()
	assert.Contains(t, view, "Installing to targets...")
	assert.Contains(t, view, "Claude Code")
	assert.Contains(t, view, "configuring...")
}

func TestApplyScreen_ViewRunningUninstall(t *testing.T) {
	theme := NewTheme()
	state := testApplyState()
	state.Action = "uninstall"
	screen := NewApplyScreen(theme, state, testApplyService(), nil, testApplyCallbacks())
	screen.Init()

	view := screen.View()
	assert.Contains(t, view, "Removing from targets...")
	assert.Contains(t, view, "removing...")
}

func TestApplyScreen_ViewDoneSuccess(t *testing.T) {
	theme := NewTheme()
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())
	screen.Init()

	s, _ := screen.Update(applyResultMsg{index: 0, err: nil})
	updated := s.(*ApplyScreen)
	s, _ = updated.Update(applyResultMsg{index: 1, err: nil})
	updated = s.(*ApplyScreen)

	view := updated.View()
	assert.Contains(t, view, "Install complete!")
	assert.Contains(t, view, "configured")
	assert.Contains(t, view, "mcp-wire install sentry")
	assert.Contains(t, view, "Install another")
	assert.Contains(t, view, "Back to menu")
	assert.Contains(t, view, "Exit")
}

func TestApplyScreen_ViewDoneUninstall(t *testing.T) {
	theme := NewTheme()
	state := testApplyState()
	state.Action = "uninstall"
	screen := NewApplyScreen(theme, state, testApplyService(), nil, testApplyCallbacks())
	screen.Init()

	s, _ := screen.Update(applyResultMsg{index: 0, err: nil})
	updated := s.(*ApplyScreen)
	s, _ = updated.Update(applyResultMsg{index: 1, err: nil})
	updated = s.(*ApplyScreen)

	view := updated.View()
	assert.Contains(t, view, "Uninstall complete!")
	assert.Contains(t, view, "removed")
	assert.Contains(t, view, "Uninstall another")
}

func TestApplyScreen_ViewDoneWithErrors(t *testing.T) {
	theme := NewTheme()
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())
	screen.Init()

	s, _ := screen.Update(applyResultMsg{index: 0, err: nil})
	updated := s.(*ApplyScreen)
	s, _ = updated.Update(applyResultMsg{index: 1, err: errors.New("disk full")})
	updated = s.(*ApplyScreen)

	view := updated.View()
	assert.Contains(t, view, "Completed with errors")
	assert.Contains(t, view, "disk full")
}

func TestApplyScreen_ViewAllFailed(t *testing.T) {
	theme := NewTheme()
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())
	screen.Init()

	s, _ := screen.Update(applyResultMsg{index: 0, err: errors.New("err1")})
	updated := s.(*ApplyScreen)
	s, _ = updated.Update(applyResultMsg{index: 1, err: errors.New("err2")})
	updated = s.(*ApplyScreen)

	view := updated.View()
	assert.Contains(t, view, "Operation failed")
}

func TestApplyScreen_PostActionNavigation(t *testing.T) {
	theme := NewTheme()
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())
	screen.Init()

	s, _ := screen.Update(applyResultMsg{index: 0, err: nil})
	updated := s.(*ApplyScreen)
	s, _ = updated.Update(applyResultMsg{index: 1, err: nil})
	updated = s.(*ApplyScreen)

	assert.Equal(t, 0, updated.PostCursor())

	// Move right.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated = s.(*ApplyScreen)
	assert.Equal(t, 1, updated.PostCursor())

	// Move right again.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated = s.(*ApplyScreen)
	assert.Equal(t, 2, updated.PostCursor())

	// Can't go further right.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated = s.(*ApplyScreen)
	assert.Equal(t, 2, updated.PostCursor())

	// Move left.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyLeft})
	updated = s.(*ApplyScreen)
	assert.Equal(t, 1, updated.PostCursor())
}

func TestApplyScreen_PostActionVimKeys(t *testing.T) {
	theme := NewTheme()
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())
	screen.Init()

	s, _ := screen.Update(applyResultMsg{index: 0, err: nil})
	updated := s.(*ApplyScreen)
	s, _ = updated.Update(applyResultMsg{index: 1, err: nil})
	updated = s.(*ApplyScreen)

	// 'l' moves right.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	updated = s.(*ApplyScreen)
	assert.Equal(t, 1, updated.PostCursor())

	// 'h' moves left.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	updated = s.(*ApplyScreen)
	assert.Equal(t, 0, updated.PostCursor())
}

func TestApplyScreen_PostActionAnother(t *testing.T) {
	theme := NewTheme()
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())
	screen.Init()

	s, _ := screen.Update(applyResultMsg{index: 0, err: nil})
	updated := s.(*ApplyScreen)
	s, _ = updated.Update(applyResultMsg{index: 1, err: nil})
	updated = s.(*ApplyScreen)

	// Cursor at 0 = "Install another".
	_, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	action, ok := msg.(applyPostActionMsg)
	require.True(t, ok)
	assert.Equal(t, "another", action.action)
}

func TestApplyScreen_PostActionMenu(t *testing.T) {
	theme := NewTheme()
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())
	screen.Init()

	s, _ := screen.Update(applyResultMsg{index: 0, err: nil})
	updated := s.(*ApplyScreen)
	s, _ = updated.Update(applyResultMsg{index: 1, err: nil})
	updated = s.(*ApplyScreen)

	// Move to "Back to menu" (index 1).
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated = s.(*ApplyScreen)

	_, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	action, ok := msg.(applyPostActionMsg)
	require.True(t, ok)
	assert.Equal(t, "menu", action.action)
}

func TestApplyScreen_PostActionExit(t *testing.T) {
	theme := NewTheme()
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())
	screen.Init()

	s, _ := screen.Update(applyResultMsg{index: 0, err: nil})
	updated := s.(*ApplyScreen)
	s, _ = updated.Update(applyResultMsg{index: 1, err: nil})
	updated = s.(*ApplyScreen)

	// Move to "Exit" (index 2).
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated = s.(*ApplyScreen)
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated = s.(*ApplyScreen)

	_, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	action, ok := msg.(applyPostActionMsg)
	require.True(t, ok)
	assert.Equal(t, "exit", action.action)
}

func TestApplyScreen_EscSendsMenuPostAction(t *testing.T) {
	theme := NewTheme()
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())
	screen.Init()

	s, _ := screen.Update(applyResultMsg{index: 0, err: nil})
	updated := s.(*ApplyScreen)
	s, _ = updated.Update(applyResultMsg{index: 1, err: nil})
	updated = s.(*ApplyScreen)

	_, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)

	msg := cmd()
	action, ok := msg.(applyPostActionMsg)
	require.True(t, ok)
	assert.Equal(t, "menu", action.action)
}

func TestApplyScreen_StatusHintsRunning(t *testing.T) {
	theme := NewTheme()
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())
	screen.Init()

	hints := screen.StatusHints()
	assert.Empty(t, hints)
}

func TestApplyScreen_StatusHintsDone(t *testing.T) {
	theme := NewTheme()
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())
	screen.Init()

	s, _ := screen.Update(applyResultMsg{index: 0, err: nil})
	updated := s.(*ApplyScreen)
	s, _ = updated.Update(applyResultMsg{index: 1, err: nil})
	updated = s.(*ApplyScreen)

	hints := updated.StatusHints()
	assert.Len(t, hints, 2)
	descs := hintDescs(hints)
	assert.Contains(t, descs, "choose")
	assert.Contains(t, descs, "confirm")
}

func TestApplyScreen_WindowSizeMsg(t *testing.T) {
	theme := NewTheme()
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())

	s, _ := screen.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	updated := s.(*ApplyScreen)
	assert.Equal(t, 100, updated.width)
}

func TestApplyScreen_EquivalentCommandShowsScope(t *testing.T) {
	theme := NewTheme()
	state := testApplyState()
	state.Scope = targetpkg.ConfigScopeProject
	screen := NewApplyScreen(theme, state, testApplyService(), nil, testApplyCallbacks())
	screen.Init()

	s, _ := screen.Update(applyResultMsg{index: 0, err: nil})
	updated := s.(*ApplyScreen)
	s, _ = updated.Update(applyResultMsg{index: 1, err: nil})
	updated = s.(*ApplyScreen)

	view := updated.View()
	assert.Contains(t, view, "--scope project")
}

func TestApplyScreen_AuthHintShown(t *testing.T) {
	theme := NewTheme()
	callbacks := testApplyCallbacks()
	callbacks.ServiceUsesOAuth = func(_ service.Service) bool { return true }
	callbacks.OAuthManualHint = func(t targetpkg.Target) string {
		if t.Slug() == "claude" {
			return "In Claude Code, run /mcp to complete OAuth"
		}
		return ""
	}
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, callbacks)
	screen.Init()

	s, _ := screen.Update(applyResultMsg{index: 0, err: nil, authHint: "In Claude Code, run /mcp to complete OAuth"})
	updated := s.(*ApplyScreen)
	s, _ = updated.Update(applyResultMsg{index: 1, err: nil})
	updated = s.(*ApplyScreen)

	view := updated.View()
	assert.Contains(t, view, "In Claude Code, run /mcp to complete OAuth")
}

func TestApplyScreen_DispatchCallsInstall(t *testing.T) {
	theme := NewTheme()
	installed := make([]string, 0)
	callbacks := ApplyCallbacks{
		InstallTarget: func(_ service.Service, _ map[string]string, tgt targetpkg.Target, _ targetpkg.ConfigScope) error {
			installed = append(installed, tgt.Slug())
			return nil
		},
	}
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, callbacks)
	cmd := screen.Init()
	require.NotNil(t, cmd)

	// Execute the command (simulates the runtime executing the tea.Cmd).
	msg := cmd()
	result, ok := msg.(applyResultMsg)
	require.True(t, ok)
	assert.Equal(t, 0, result.index)
	assert.Nil(t, result.err)
	assert.Contains(t, installed, "claude")
}

func TestApplyScreen_DispatchCallsUninstall(t *testing.T) {
	theme := NewTheme()
	uninstalled := make([]string, 0)
	callbacks := ApplyCallbacks{
		UninstallTarget: func(_ string, tgt targetpkg.Target, _ targetpkg.ConfigScope) error {
			uninstalled = append(uninstalled, tgt.Slug())
			return nil
		},
	}
	state := testApplyState()
	state.Action = "uninstall"
	screen := NewApplyScreen(theme, state, testApplyService(), nil, callbacks)
	cmd := screen.Init()
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(applyResultMsg)
	require.True(t, ok)
	assert.Equal(t, 0, result.index)
	assert.Nil(t, result.err)
	assert.Contains(t, uninstalled, "claude")
}

func TestApplyScreen_InvalidResultIndex(t *testing.T) {
	theme := NewTheme()
	screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())
	screen.Init()

	// Out-of-range index should be handled gracefully.
	s, cmd := screen.Update(applyResultMsg{index: 99, err: nil})
	updated := s.(*ApplyScreen)
	assert.Nil(t, cmd)
	assert.Equal(t, applySubStateRunning, updated.ApplySubState())
}

// --- Credential cleanup tests ---

func testUninstallStateWithEnvVars() (WizardState, service.Service) {
	svc := service.Service{
		Name:        "sentry",
		Description: "Error tracking",
		Transport:   "sse",
		URL:         "https://mcp.sentry.dev/sse",
		Env: []service.EnvVar{
			{Name: "SENTRY_AUTH_TOKEN", Description: "Auth token", Required: true},
			{Name: "SENTRY_ORG", Description: "Org slug", Required: false},
		},
	}
	state := WizardState{
		Action: "uninstall",
		Source: "curated",
		Entry:  catalog.FromCurated(svc),
		Targets: []targetpkg.Target{
			&mockTarget{name: "Claude Code", slug: "claude", installed: true},
		},
		Scope: targetpkg.ConfigScopeUser,
	}
	return state, svc
}

func testApplyCallbacksWithCredRemoval() (ApplyCallbacks, *[]string) {
	var removedNames []string
	cb := testApplyCallbacks()
	cb.RemoveStoredCredentials = func(envNames []string) (int, error) {
		removedNames = append(removedNames, envNames...)
		return len(envNames), nil
	}
	return cb, &removedNames
}

func finishAllTargets(screen *ApplyScreen, count int) *ApplyScreen {
	var s Screen
	for i := 0; i < count; i++ {
		s, _ = screen.Update(applyResultMsg{index: i, err: nil})
		screen = s.(*ApplyScreen)
	}
	return screen
}

func TestApplyScreen_UninstallWithEnvVars_ShowsCredCleanup(t *testing.T) {
	theme := NewTheme()
	state, svc := testUninstallStateWithEnvVars()
	callbacks, _ := testApplyCallbacksWithCredRemoval()
	screen := NewApplyScreen(theme, state, svc, nil, callbacks)
	screen.Init()

	updated := finishAllTargets(screen, 1)

	assert.Equal(t, applySubStateCredCleanup, updated.ApplySubState())
	assert.Equal(t, 0, updated.CredCleanupCursor()) // default to No
}

func TestApplyScreen_UninstallNoEnvVars_SkipsCred(t *testing.T) {
	theme := NewTheme()
	state := testApplyState()
	state.Action = "uninstall"
	callbacks, _ := testApplyCallbacksWithCredRemoval()
	// testApplyService() has no env vars.
	screen := NewApplyScreen(theme, state, testApplyService(), nil, callbacks)
	screen.Init()

	updated := finishAllTargets(screen, 2)

	assert.Equal(t, applySubStateDone, updated.ApplySubState())
}

func TestApplyScreen_UninstallWithFailures_SkipsCred(t *testing.T) {
	theme := NewTheme()
	state, svc := testUninstallStateWithEnvVars()
	callbacks, _ := testApplyCallbacksWithCredRemoval()
	screen := NewApplyScreen(theme, state, svc, nil, callbacks)
	screen.Init()

	// Target fails.
	s, _ := screen.Update(applyResultMsg{index: 0, err: errors.New("fail")})
	updated := s.(*ApplyScreen)

	assert.Equal(t, applySubStateDone, updated.ApplySubState())
}

func TestApplyScreen_UninstallNoCallback_SkipsCred(t *testing.T) {
	theme := NewTheme()
	state, svc := testUninstallStateWithEnvVars()
	callbacks := testApplyCallbacks() // no RemoveStoredCredentials
	screen := NewApplyScreen(theme, state, svc, nil, callbacks)
	screen.Init()

	updated := finishAllTargets(screen, 1)

	assert.Equal(t, applySubStateDone, updated.ApplySubState())
}

func TestApplyScreen_InstallNeverShowsCredCleanup(t *testing.T) {
	theme := NewTheme()
	state := testApplyState() // install action
	svc := service.Service{
		Name: "sentry",
		Env:  []service.EnvVar{{Name: "TOKEN", Required: true}},
	}
	callbacks, _ := testApplyCallbacksWithCredRemoval()
	screen := NewApplyScreen(theme, state, svc, nil, callbacks)
	screen.Init()

	updated := finishAllTargets(screen, 2)

	assert.Equal(t, applySubStateDone, updated.ApplySubState())
}

func TestApplyScreen_CredCleanupCursorNavigation(t *testing.T) {
	theme := NewTheme()
	state, svc := testUninstallStateWithEnvVars()
	callbacks, _ := testApplyCallbacksWithCredRemoval()
	screen := NewApplyScreen(theme, state, svc, nil, callbacks)
	screen.Init()

	updated := finishAllTargets(screen, 1)
	require.Equal(t, applySubStateCredCleanup, updated.ApplySubState())
	assert.Equal(t, 0, updated.CredCleanupCursor()) // default No

	// Move right to Yes.
	s, _ := updated.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated = s.(*ApplyScreen)
	assert.Equal(t, 1, updated.CredCleanupCursor())

	// Can't go further right.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated = s.(*ApplyScreen)
	assert.Equal(t, 1, updated.CredCleanupCursor())

	// Move left to No.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyLeft})
	updated = s.(*ApplyScreen)
	assert.Equal(t, 0, updated.CredCleanupCursor())

	// Can't go further left.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyLeft})
	updated = s.(*ApplyScreen)
	assert.Equal(t, 0, updated.CredCleanupCursor())
}

func TestApplyScreen_CredCleanupVimKeys(t *testing.T) {
	theme := NewTheme()
	state, svc := testUninstallStateWithEnvVars()
	callbacks, _ := testApplyCallbacksWithCredRemoval()
	screen := NewApplyScreen(theme, state, svc, nil, callbacks)
	screen.Init()

	updated := finishAllTargets(screen, 1)

	// 'l' moves right.
	s, _ := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	updated = s.(*ApplyScreen)
	assert.Equal(t, 1, updated.CredCleanupCursor())

	// 'h' moves left.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	updated = s.(*ApplyScreen)
	assert.Equal(t, 0, updated.CredCleanupCursor())
}

func TestApplyScreen_CredCleanupYes_RemovesCredentials(t *testing.T) {
	theme := NewTheme()
	state, svc := testUninstallStateWithEnvVars()
	callbacks, removedNames := testApplyCallbacksWithCredRemoval()
	screen := NewApplyScreen(theme, state, svc, nil, callbacks)
	screen.Init()

	updated := finishAllTargets(screen, 1)

	// Move to Yes.
	s, _ := updated.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated = s.(*ApplyScreen)

	// Confirm.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = s.(*ApplyScreen)

	assert.Equal(t, applySubStateDone, updated.ApplySubState())
	assert.Equal(t, "Stored credentials removed.", updated.CredCleanupMsg())
	assert.Contains(t, *removedNames, "SENTRY_AUTH_TOKEN")
	assert.Contains(t, *removedNames, "SENTRY_ORG")
}

func TestApplyScreen_CredCleanupNo_SkipsRemoval(t *testing.T) {
	theme := NewTheme()
	state, svc := testUninstallStateWithEnvVars()
	callbacks, removedNames := testApplyCallbacksWithCredRemoval()
	screen := NewApplyScreen(theme, state, svc, nil, callbacks)
	screen.Init()

	updated := finishAllTargets(screen, 1)

	// Cursor defaults to No. Press enter.
	s, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = s.(*ApplyScreen)

	assert.Equal(t, applySubStateDone, updated.ApplySubState())
	assert.Empty(t, updated.CredCleanupMsg())
	assert.Empty(t, *removedNames)
}

func TestApplyScreen_CredCleanupEsc_SkipsRemoval(t *testing.T) {
	theme := NewTheme()
	state, svc := testUninstallStateWithEnvVars()
	callbacks, removedNames := testApplyCallbacksWithCredRemoval()
	screen := NewApplyScreen(theme, state, svc, nil, callbacks)
	screen.Init()

	updated := finishAllTargets(screen, 1)

	// Esc skips.
	s, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = s.(*ApplyScreen)

	assert.Equal(t, applySubStateDone, updated.ApplySubState())
	assert.Empty(t, updated.CredCleanupMsg())
	assert.Empty(t, *removedNames)
}

func TestApplyScreen_CredCleanupNoneFound(t *testing.T) {
	theme := NewTheme()
	state, svc := testUninstallStateWithEnvVars()
	callbacks := testApplyCallbacks()
	callbacks.RemoveStoredCredentials = func(_ []string) (int, error) {
		return 0, nil // no stored credentials found
	}
	screen := NewApplyScreen(theme, state, svc, nil, callbacks)
	screen.Init()

	updated := finishAllTargets(screen, 1)

	// Move to Yes and confirm.
	s, _ := updated.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated = s.(*ApplyScreen)
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = s.(*ApplyScreen)

	assert.Equal(t, applySubStateDone, updated.ApplySubState())
	assert.Equal(t, "No stored credentials found.", updated.CredCleanupMsg())
}

func TestApplyScreen_CredCleanupError(t *testing.T) {
	theme := NewTheme()
	state, svc := testUninstallStateWithEnvVars()
	callbacks := testApplyCallbacks()
	callbacks.RemoveStoredCredentials = func(_ []string) (int, error) {
		return 0, errors.New("permission denied")
	}
	screen := NewApplyScreen(theme, state, svc, nil, callbacks)
	screen.Init()

	updated := finishAllTargets(screen, 1)

	// Move to Yes and confirm.
	s, _ := updated.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated = s.(*ApplyScreen)
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = s.(*ApplyScreen)

	assert.Equal(t, applySubStateDone, updated.ApplySubState())
	assert.Equal(t, "Error removing credentials: permission denied", updated.CredCleanupMsg())
}

func TestApplyScreen_CredCleanupViewShowsPrompt(t *testing.T) {
	theme := NewTheme()
	state, svc := testUninstallStateWithEnvVars()
	callbacks, _ := testApplyCallbacksWithCredRemoval()
	screen := NewApplyScreen(theme, state, svc, nil, callbacks)
	screen.Init()

	updated := finishAllTargets(screen, 1)

	view := updated.View()
	assert.Contains(t, view, "Remove stored credentials?")
	assert.Contains(t, view, "No")
	assert.Contains(t, view, "Yes")
	assert.Contains(t, view, "Uninstall complete!")
}

func TestApplyScreen_CredCleanupMsgInDoneView(t *testing.T) {
	theme := NewTheme()
	state, svc := testUninstallStateWithEnvVars()
	callbacks, _ := testApplyCallbacksWithCredRemoval()
	screen := NewApplyScreen(theme, state, svc, nil, callbacks)
	screen.Init()

	updated := finishAllTargets(screen, 1)

	// Choose Yes.
	s, _ := updated.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated = s.(*ApplyScreen)
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = s.(*ApplyScreen)

	view := updated.View()
	assert.Contains(t, view, "Stored credentials removed.")
	assert.Contains(t, view, "Uninstall another")
}

func TestApplyScreen_CredCleanupStatusHints(t *testing.T) {
	theme := NewTheme()
	state, svc := testUninstallStateWithEnvVars()
	callbacks, _ := testApplyCallbacksWithCredRemoval()
	screen := NewApplyScreen(theme, state, svc, nil, callbacks)
	screen.Init()

	updated := finishAllTargets(screen, 1)

	hints := updated.StatusHints()
	descs := hintDescs(hints)
	assert.Contains(t, descs, "choose")
	assert.Contains(t, descs, "confirm")
	assert.Contains(t, descs, "skip")
}

func TestApplyScreen_EnvVarNames(t *testing.T) {
	theme := NewTheme()
	state, svc := testUninstallStateWithEnvVars()
	// Add a duplicate env var.
	svc.Env = append(svc.Env, service.EnvVar{Name: "SENTRY_AUTH_TOKEN"})
	screen := NewApplyScreen(theme, state, svc, nil, testApplyCallbacks())

	names := screen.envVarNames()
	assert.Equal(t, []string{"SENTRY_AUTH_TOKEN", "SENTRY_ORG"}, names)
}
