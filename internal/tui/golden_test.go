package tui

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andreagrandi/mcp-wire/internal/service"
	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
)

// errPermissionDenied is a fixed error used to render the failure golden
// deterministically.
var errPermissionDenied = errors.New("permission denied")

// updateGolden regenerates the golden files instead of comparing against them.
// Run: go test ./internal/tui/ -run Golden -update
var updateGolden = flag.Bool("update", false, "update golden files in testdata/golden")

// TestMain forces an uncolored profile so rendered TUI output is deterministic
// and golden files stay human-readable across environments (no ANSI escapes).
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.Ascii)
	os.Exit(m.Run())
}

// assertGolden compares actual against the golden file named name (without the
// .golden suffix), or rewrites it when -update is passed.
func assertGolden(t *testing.T, name, actual string) {
	t.Helper()

	path := filepath.Join("testdata", "golden", name+".golden")

	if *updateGolden {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(actual), 0o644))
		return
	}

	want, err := os.ReadFile(path)
	require.NoErrorf(t, err, "missing golden file %s; regenerate with: go test ./internal/tui/ -run Golden -update", path)
	assert.Equalf(t, string(want), actual,
		"rendered output differs from %s; if the change is intentional, regenerate with: go test ./internal/tui/ -run Golden -update", path)
}

func TestReviewScreenGolden(t *testing.T) {
	theme := NewTheme()

	t.Run("install", func(t *testing.T) {
		screen := NewReviewScreen(theme, testReviewState(), false)
		assertGolden(t, "review_install", screen.View())
	})

	t.Run("install_with_source_and_scope", func(t *testing.T) {
		state := testReviewState()
		state.Targets = []targetpkg.Target{
			&mockTarget{
				name: "Claude Code", slug: "claude", installed: true,
				scopes: []targetpkg.ConfigScope{targetpkg.ConfigScopeUser, targetpkg.ConfigScopeProject},
			},
		}
		state.Scope = targetpkg.ConfigScopeProject
		screen := NewReviewScreen(theme, state, true)
		assertGolden(t, "review_install_source_scope", screen.View())
	})

	t.Run("uninstall", func(t *testing.T) {
		state := testReviewState()
		state.Action = "uninstall"
		screen := NewReviewScreen(theme, state, false)
		assertGolden(t, "review_uninstall", screen.View())
	})
}

func TestTrustScreenGolden(t *testing.T) {
	theme := NewTheme()

	t.Run("remote", func(t *testing.T) {
		screen := NewTrustScreen(theme, testRegistryEntry())
		assertGolden(t, "trust_remote", screen.View())
	})

	t.Run("package", func(t *testing.T) {
		screen := NewTrustScreen(theme, testRegistryEntryWithPackage())
		assertGolden(t, "trust_package", screen.View())
	})

	t.Run("secrets", func(t *testing.T) {
		screen := NewTrustScreen(theme, testRegistryEntryWithSecrets())
		assertGolden(t, "trust_secrets", screen.View())
	})
}

func testCredentialEnvVars() []service.EnvVar {
	return []service.EnvVar{
		{
			Name:        "SENTRY_TOKEN",
			Description: "API token",
			Required:    true,
			SetupURL:    "https://sentry.io/settings/auth-tokens/",
			SetupHint:   "Create a token with read scope",
		},
		{
			Name:        "SENTRY_ORG",
			Description: "Organization slug",
			Required:    true,
		},
	}
}

func TestCredentialScreenGolden(t *testing.T) {
	theme := NewTheme()
	noopStore := func(_, _ string) error { return nil }
	noopOpen := func(_ string) error { return nil }

	t.Run("input", func(t *testing.T) {
		screen := NewCredentialScreen(theme, testCredentialEnvVars(), nil, noopStore, noopOpen)
		assertGolden(t, "credential_input", screen.View())
	})

	t.Run("save", func(t *testing.T) {
		screen := NewCredentialScreen(theme, testCredentialEnvVars(), nil, noopStore, noopOpen)
		s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("secret-token")})
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyEnter})
		assertGolden(t, "credential_save", s.View())
	})
}

func TestApplyScreenGolden(t *testing.T) {
	theme := NewTheme()

	t.Run("running", func(t *testing.T) {
		screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())
		screen.Init()
		assertGolden(t, "apply_running", screen.View())
	})

	t.Run("done_success", func(t *testing.T) {
		screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())
		screen.Init()
		var s Screen = screen
		s, _ = s.Update(applyResultMsg{index: 0, err: nil})
		s, _ = s.Update(applyResultMsg{index: 1, err: nil})
		assertGolden(t, "apply_done_success", s.View())
	})

	t.Run("done_partial_failure", func(t *testing.T) {
		screen := NewApplyScreen(theme, testApplyState(), testApplyService(), nil, testApplyCallbacks())
		screen.Init()
		var s Screen = screen
		s, _ = s.Update(applyResultMsg{index: 0, err: nil})
		s, _ = s.Update(applyResultMsg{index: 1, err: errPermissionDenied})
		assertGolden(t, "apply_done_partial_failure", s.View())
	})

	t.Run("uninstall_credential_cleanup", func(t *testing.T) {
		state := testApplyState()
		state.Action = "uninstall"
		svc := testApplyService()
		svc.Env = []service.EnvVar{{Name: "SENTRY_TOKEN", Required: true}}
		callbacks := testApplyCallbacks()
		callbacks.RemoveStoredCredentials = func(_ []string) (int, error) { return 0, nil }

		screen := NewApplyScreen(theme, state, svc, nil, callbacks)
		screen.Init()
		var s Screen = screen
		s, _ = s.Update(applyResultMsg{index: 0, err: nil})
		s, _ = s.Update(applyResultMsg{index: 1, err: nil})
		assertGolden(t, "apply_uninstall_credential_cleanup", s.View())
	})
}
