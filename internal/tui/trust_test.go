package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andreagrandi/mcp-wire/internal/catalog"
	"github.com/andreagrandi/mcp-wire/internal/registry"
	"github.com/andreagrandi/mcp-wire/internal/service"
)

func testRegistryEntry() catalog.Entry {
	return catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "community-svc",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:        "community-svc",
				Description: "A community service",
				Remotes: []registry.Transport{
					{Type: "sse", URL: "https://example.com/sse"},
				},
				Repository: &registry.Repository{
					URL: "https://github.com/example/svc",
				},
			},
		},
	}
}

func testRegistryEntryWithSecrets() catalog.Entry {
	entry := testRegistryEntry()
	entry.Registry.Server.Remotes[0].Headers = []registry.KeyValueInput{
		{Name: "API_KEY", Description: "API key", IsRequired: true, IsSecret: true},
	}
	return entry
}

func testRegistryEntryWithPackage() catalog.Entry {
	return catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "pkg-svc",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:        "pkg-svc",
				Description: "A package service",
				Packages: []registry.Package{
					{
						RegistryType: "npm",
						Identifier:   "@example/mcp-server",
						Version:      "1.2.3",
						RuntimeHint:  "requires Node.js 18+",
					},
				},
			},
		},
	}
}

func TestNewTrustScreen(t *testing.T) {
	theme := NewTheme()
	entry := testRegistryEntry()
	screen := NewTrustScreen(theme, entry)

	assert.Equal(t, 0, screen.Cursor()) // default to "No"
	assert.Equal(t, "community-svc", screen.entry.Name)
}

func TestTrustScreen_Init(t *testing.T) {
	theme := NewTheme()
	screen := NewTrustScreen(theme, testRegistryEntry())
	assert.Nil(t, screen.Init())
}

func TestTrustScreen_NavigateRight(t *testing.T) {
	theme := NewTheme()
	screen := NewTrustScreen(theme, testRegistryEntry())

	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated := s.(*TrustScreen)
	assert.Equal(t, 1, updated.Cursor())
}

func TestTrustScreen_NavigateLeft(t *testing.T) {
	theme := NewTheme()
	screen := NewTrustScreen(theme, testRegistryEntry())

	// Move right then left.
	var s Screen = screen
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRight})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyLeft})
	updated := s.(*TrustScreen)
	assert.Equal(t, 0, updated.Cursor())
}

func TestTrustScreen_NavigateLeftAtStart(t *testing.T) {
	theme := NewTheme()
	screen := NewTrustScreen(theme, testRegistryEntry())

	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyLeft})
	updated := s.(*TrustScreen)
	assert.Equal(t, 0, updated.Cursor())
}

func TestTrustScreen_NavigateRightAtEnd(t *testing.T) {
	theme := NewTheme()
	screen := NewTrustScreen(theme, testRegistryEntry())

	var s Screen = screen
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRight})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated := s.(*TrustScreen)
	assert.Equal(t, 1, updated.Cursor())
}

func TestTrustScreen_VimKeys(t *testing.T) {
	theme := NewTheme()
	screen := NewTrustScreen(theme, testRegistryEntry())

	// 'l' moves right.
	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	updated := s.(*TrustScreen)
	assert.Equal(t, 1, updated.Cursor())

	// 'h' moves left.
	s, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	updated = s.(*TrustScreen)
	assert.Equal(t, 0, updated.Cursor())
}

func TestTrustScreen_EnterConfirmsNo(t *testing.T) {
	theme := NewTheme()
	screen := NewTrustScreen(theme, testRegistryEntry())

	// Default cursor = 0 = "No".
	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	confirm, ok := msg.(trustConfirmMsg)
	require.True(t, ok)
	assert.False(t, confirm.confirmed)
}

func TestTrustScreen_EnterConfirmsYes(t *testing.T) {
	theme := NewTheme()
	screen := NewTrustScreen(theme, testRegistryEntry())

	// Move to "Yes".
	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyRight})

	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	confirm, ok := msg.(trustConfirmMsg)
	require.True(t, ok)
	assert.True(t, confirm.confirmed)
}

func TestTrustScreen_EscGoesBack(t *testing.T) {
	theme := NewTheme()
	screen := NewTrustScreen(theme, testRegistryEntry())

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(BackMsg)
	assert.True(t, ok)
}

func TestTrustScreen_ViewShowsWarningHeader(t *testing.T) {
	theme := NewTheme()
	screen := NewTrustScreen(theme, testRegistryEntry())

	view := screen.View()
	assert.Contains(t, view, "Registry Service Information")
}

func TestTrustScreen_ViewShowsSource(t *testing.T) {
	theme := NewTheme()
	screen := NewTrustScreen(theme, testRegistryEntry())

	view := screen.View()
	assert.Contains(t, view, "registry")
	assert.Contains(t, view, "community, not vetted by mcp-wire")
}

func TestTrustScreen_ViewShowsTransport(t *testing.T) {
	theme := NewTheme()
	screen := NewTrustScreen(theme, testRegistryEntry())

	view := screen.View()
	assert.Contains(t, view, "sse")
}

func TestTrustScreen_ViewShowsRepo(t *testing.T) {
	theme := NewTheme()
	screen := NewTrustScreen(theme, testRegistryEntry())

	view := screen.View()
	assert.Contains(t, view, "https://github.com/example/svc")
}

func TestTrustScreen_ViewShowsSecrets(t *testing.T) {
	theme := NewTheme()
	screen := NewTrustScreen(theme, testRegistryEntryWithSecrets())

	view := screen.View()
	assert.Contains(t, view, "API_KEY")
}

func TestTrustScreen_ViewShowsPackageInfo(t *testing.T) {
	theme := NewTheme()
	screen := NewTrustScreen(theme, testRegistryEntryWithPackage())

	view := screen.View()
	assert.Contains(t, view, "npm")
	assert.Contains(t, view, "@example/mcp-server@1.2.3")
	assert.Contains(t, view, "Node.js 18+")
}

func TestTrustScreen_ViewShowsInstallType(t *testing.T) {
	theme := NewTheme()
	screen := NewTrustScreen(theme, testRegistryEntry())

	view := screen.View()
	assert.Contains(t, view, "remote")
}

func TestTrustScreen_ViewShowsChoices(t *testing.T) {
	theme := NewTheme()
	screen := NewTrustScreen(theme, testRegistryEntry())

	view := screen.View()
	assert.Contains(t, view, "No")
	assert.Contains(t, view, "Yes")
	assert.Contains(t, view, "Proceed with this registry service?")
}

func TestTrustScreen_StatusHints(t *testing.T) {
	theme := NewTheme()
	screen := NewTrustScreen(theme, testRegistryEntry())

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

func TestTrustScreen_WindowResize(t *testing.T) {
	theme := NewTheme()
	screen := NewTrustScreen(theme, testRegistryEntry())

	s, _ := screen.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	updated := s.(*TrustScreen)
	assert.Equal(t, 100, updated.width)
}

func TestRegistryEntryNeedsConfirmation(t *testing.T) {
	curated := catalog.FromCurated(service.Service{Name: "sentry"})
	assert.False(t, registryEntryNeedsConfirmation(curated))

	reg := testRegistryEntry()
	assert.True(t, registryEntryNeedsConfirmation(reg))
}
