package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andreagrandi/mcp-wire/internal/service"
)

func testEnvVars() []service.EnvVar {
	return []service.EnvVar{
		{
			Name:        "API_TOKEN",
			Description: "API authentication token",
			Required:    true,
			SetupURL:    "https://example.com/tokens",
			SetupHint:   "Create a read-only token",
		},
		{
			Name:        "SECRET_KEY",
			Description: "Secret key for signing",
			Required:    true,
		},
	}
}

func testSingleEnvVar() []service.EnvVar {
	return []service.EnvVar{
		{
			Name:        "TOKEN",
			Description: "Auth token",
			Required:    true,
			SetupURL:    "https://example.com/setup",
		},
	}
}

func TestNewCredentialScreen(t *testing.T) {
	theme := NewTheme()
	screen := NewCredentialScreen(theme, testEnvVars(), nil, nil, nil)

	assert.Equal(t, 0, screen.Current())
	assert.Equal(t, credSubStateInput, screen.SubState())
	assert.Empty(t, screen.Resolved())
}

func TestNewCredentialScreen_PreResolved(t *testing.T) {
	theme := NewTheme()
	pre := map[string]string{"EXISTING": "value"}
	screen := NewCredentialScreen(theme, testEnvVars(), pre, nil, nil)

	resolved := screen.Resolved()
	assert.Equal(t, "value", resolved["EXISTING"])
}

func TestCredentialScreen_Init(t *testing.T) {
	theme := NewTheme()
	screen := NewCredentialScreen(theme, testEnvVars(), nil, nil, nil)

	cmd := screen.Init()
	assert.NotNil(t, cmd)
}

func TestCredentialScreen_ViewShowsFirstCredential(t *testing.T) {
	theme := NewTheme()
	screen := NewCredentialScreen(theme, testEnvVars(), nil, nil, nil)

	view := screen.View()
	assert.Contains(t, view, "API_TOKEN")
	assert.Contains(t, view, "API authentication token")
	assert.Contains(t, view, "[1/2]")
	assert.Contains(t, view, "https://example.com/tokens")
	assert.Contains(t, view, "Create a read-only token")
}

func TestCredentialScreen_ViewShowsProgress(t *testing.T) {
	theme := NewTheme()
	screen := NewCredentialScreen(theme, testEnvVars(), nil, nil, nil)

	view := screen.View()
	assert.Contains(t, view, "[1/2]")
}

func TestCredentialScreen_EmptyEnterDoesNothing(t *testing.T) {
	theme := NewTheme()
	screen := NewCredentialScreen(theme, testEnvVars(), nil, nil, nil)

	s, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := s.(*CredentialScreen)

	assert.Equal(t, 0, updated.Current())
	assert.Equal(t, credSubStateInput, updated.SubState())
	assert.Nil(t, cmd)
}

func TestCredentialScreen_EscSendsBack(t *testing.T) {
	theme := NewTheme()
	screen := NewCredentialScreen(theme, testEnvVars(), nil, nil, nil)

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(BackMsg)
	assert.True(t, ok)
}

func TestCredentialScreen_EnterValueNoStore_Advances(t *testing.T) {
	theme := NewTheme()
	screen := NewCredentialScreen(theme, testEnvVars(), nil, nil, nil)

	// Type a value.
	typeText(screen, "my-token-123")

	// Press enter to submit.
	s, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := s.(*CredentialScreen)

	// Without storeCredential callback, should advance directly.
	assert.Equal(t, 1, updated.Current())
	assert.Equal(t, credSubStateInput, updated.SubState())
	assert.Equal(t, "my-token-123", updated.Resolved()["API_TOKEN"])
	assert.NotNil(t, cmd) // focus cmd for next input
}

func TestCredentialScreen_EnterValueWithStore_ShowsSavePrompt(t *testing.T) {
	theme := NewTheme()
	stored := make(map[string]string)
	storeFn := func(name, value string) error {
		stored[name] = value
		return nil
	}
	screen := NewCredentialScreen(theme, testEnvVars(), nil, storeFn, nil)

	typeText(screen, "my-token")
	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := s.(*CredentialScreen)

	assert.Equal(t, credSubStateSave, updated.SubState())
	assert.Equal(t, 1, updated.SaveCursor()) // default to Yes
}

func TestCredentialScreen_SavePromptYes(t *testing.T) {
	theme := NewTheme()
	stored := make(map[string]string)
	storeFn := func(name, value string) error {
		stored[name] = value
		return nil
	}
	screen := NewCredentialScreen(theme, testEnvVars(), nil, storeFn, nil)

	typeText(screen, "my-token")
	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := s.(*CredentialScreen)

	// Cursor defaults to Yes (1). Press enter to confirm save.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = s.(*CredentialScreen)

	assert.Equal(t, "my-token", stored["API_TOKEN"])
	assert.Equal(t, 1, updated.Current()) // advanced to next
	assert.Equal(t, credSubStateInput, updated.SubState())
}

func TestCredentialScreen_SavePromptNo(t *testing.T) {
	theme := NewTheme()
	stored := make(map[string]string)
	storeFn := func(name, value string) error {
		stored[name] = value
		return nil
	}
	screen := NewCredentialScreen(theme, testEnvVars(), nil, storeFn, nil)

	typeText(screen, "my-token")
	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := s.(*CredentialScreen)

	// Move cursor to No (0) then confirm.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyLeft})
	updated = s.(*CredentialScreen)
	assert.Equal(t, 0, updated.SaveCursor())

	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = s.(*CredentialScreen)

	assert.Empty(t, stored) // nothing stored
	assert.Equal(t, 1, updated.Current())
	assert.Equal(t, "my-token", updated.Resolved()["API_TOKEN"])
}

func TestCredentialScreen_SavePromptEscSkips(t *testing.T) {
	theme := NewTheme()
	stored := make(map[string]string)
	storeFn := func(name, value string) error {
		stored[name] = value
		return nil
	}
	screen := NewCredentialScreen(theme, testEnvVars(), nil, storeFn, nil)

	typeText(screen, "my-token")
	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := s.(*CredentialScreen)

	// Press Esc to skip saving.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = s.(*CredentialScreen)

	assert.Empty(t, stored) // nothing stored
	assert.Equal(t, 1, updated.Current())
}

func TestCredentialScreen_CompletionSendsDoneMsg(t *testing.T) {
	theme := NewTheme()
	screen := NewCredentialScreen(theme, testSingleEnvVar(), nil, nil, nil)

	typeText(screen, "the-token")
	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	done, ok := msg.(credentialDoneMsg)
	require.True(t, ok)
	assert.Equal(t, "the-token", done.resolvedEnv["TOKEN"])
}

func TestCredentialScreen_AllCredentialsDone(t *testing.T) {
	theme := NewTheme()
	screen := NewCredentialScreen(theme, testEnvVars(), nil, nil, nil)

	// First credential.
	typeText(screen, "token1")
	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := s.(*CredentialScreen)

	// Second credential.
	typeText(updated, "secret1")
	_, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	done, ok := msg.(credentialDoneMsg)
	require.True(t, ok)
	assert.Equal(t, "token1", done.resolvedEnv["API_TOKEN"])
	assert.Equal(t, "secret1", done.resolvedEnv["SECRET_KEY"])
}

func TestCredentialScreen_PreResolvedMerged(t *testing.T) {
	theme := NewTheme()
	pre := map[string]string{"EXISTING": "existing-val"}
	screen := NewCredentialScreen(theme, testSingleEnvVar(), pre, nil, nil)

	typeText(screen, "new-token")
	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	done, ok := msg.(credentialDoneMsg)
	require.True(t, ok)
	assert.Equal(t, "new-token", done.resolvedEnv["TOKEN"])
	assert.Equal(t, "existing-val", done.resolvedEnv["EXISTING"])
}

func TestCredentialScreen_OpenURLKeyOnlyWhenAvailable(t *testing.T) {
	theme := NewTheme()
	screen := NewCredentialScreen(theme, testEnvVars(), nil, nil, nil)

	hints := screen.StatusHints()
	hasOpen := false
	for _, h := range hints {
		if h.Key == "Ctrl+O" {
			hasOpen = true
		}
	}
	assert.True(t, hasOpen, "first env var has setup URL, should show Ctrl+O hint")

	// Second env var has no setup URL.
	typeText(screen, "token1")
	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := s.(*CredentialScreen)

	hints = updated.StatusHints()
	hasOpen = false
	for _, h := range hints {
		if h.Key == "Ctrl+O" {
			hasOpen = true
		}
	}
	assert.False(t, hasOpen, "second env var has no setup URL, should not show Ctrl+O hint")
}

func TestCredentialScreen_OpenURLCallsCallback(t *testing.T) {
	theme := NewTheme()
	var openedURL string
	openFn := func(url string) error {
		openedURL = url
		return nil
	}
	screen := NewCredentialScreen(theme, testEnvVars(), nil, nil, openFn)

	screen.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	assert.Equal(t, "https://example.com/tokens", openedURL)
}

func TestCredentialScreen_SaveViewShowsChoices(t *testing.T) {
	theme := NewTheme()
	storeFn := func(name, value string) error { return nil }
	screen := NewCredentialScreen(theme, testEnvVars(), nil, storeFn, nil)

	typeText(screen, "tok")
	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := s.(*CredentialScreen)

	view := updated.View()
	assert.Contains(t, view, "Save to credential store?")
	assert.Contains(t, view, "No")
	assert.Contains(t, view, "Yes")
}

func TestCredentialScreen_SaveCursorNavigation(t *testing.T) {
	theme := NewTheme()
	storeFn := func(name, value string) error { return nil }
	screen := NewCredentialScreen(theme, testEnvVars(), nil, storeFn, nil)

	typeText(screen, "tok")
	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := s.(*CredentialScreen)
	assert.Equal(t, 1, updated.SaveCursor()) // default Yes

	// Move left to No.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyLeft})
	updated = s.(*CredentialScreen)
	assert.Equal(t, 0, updated.SaveCursor())

	// Can't go further left.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyLeft})
	updated = s.(*CredentialScreen)
	assert.Equal(t, 0, updated.SaveCursor())

	// Move right to Yes.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated = s.(*CredentialScreen)
	assert.Equal(t, 1, updated.SaveCursor())

	// Can't go further right.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated = s.(*CredentialScreen)
	assert.Equal(t, 1, updated.SaveCursor())
}

func TestCredentialScreen_VimKeysInSave(t *testing.T) {
	theme := NewTheme()
	storeFn := func(name, value string) error { return nil }
	screen := NewCredentialScreen(theme, testEnvVars(), nil, storeFn, nil)

	typeText(screen, "tok")
	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := s.(*CredentialScreen)

	// 'h' moves left.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	updated = s.(*CredentialScreen)
	assert.Equal(t, 0, updated.SaveCursor())

	// 'l' moves right.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	updated = s.(*CredentialScreen)
	assert.Equal(t, 1, updated.SaveCursor())
}

func TestCredentialScreen_WindowSizeMsg(t *testing.T) {
	theme := NewTheme()
	screen := NewCredentialScreen(theme, testEnvVars(), nil, nil, nil)

	s, _ := screen.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	updated := s.(*CredentialScreen)
	assert.Equal(t, 120, updated.width)
}

func TestCredentialScreen_StatusHintsInput(t *testing.T) {
	theme := NewTheme()
	screen := NewCredentialScreen(theme, testEnvVars(), nil, nil, nil)

	hints := screen.StatusHints()
	descs := hintDescs(hints)
	assert.Contains(t, descs, "submit")
	assert.Contains(t, descs, "back")
}

func TestCredentialScreen_StatusHintsSave(t *testing.T) {
	theme := NewTheme()
	storeFn := func(name, value string) error { return nil }
	screen := NewCredentialScreen(theme, testEnvVars(), nil, storeFn, nil)

	typeText(screen, "tok")
	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := s.(*CredentialScreen)

	hints := updated.StatusHints()
	descs := hintDescs(hints)
	assert.Contains(t, descs, "choose")
	assert.Contains(t, descs, "confirm")
	assert.Contains(t, descs, "skip")
}

func TestCredentialScreen_NoDescriptionFormat(t *testing.T) {
	theme := NewTheme()
	envVars := []service.EnvVar{
		{Name: "PLAIN_VAR", Required: true},
	}
	screen := NewCredentialScreen(theme, envVars, nil, nil, nil)

	view := screen.View()
	assert.Contains(t, view, "PLAIN_VAR required.")
	assert.NotContains(t, view, "()")
}

// typeText simulates typing a string into a credential screen's textinput.
func typeText(screen *CredentialScreen, text string) {
	for _, r := range text {
		screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func hintDescs(hints []KeyHint) []string {
	descs := make([]string, len(hints))
	for i, h := range hints {
		descs[i] = h.Desc
	}
	return descs
}
