package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andreagrandi/mcp-wire/internal/catalog"
	"github.com/andreagrandi/mcp-wire/internal/service"
)

func testServiceCatalog() *catalog.Catalog {
	curated := []catalog.Entry{
		catalog.FromCurated(service.Service{Name: "alpha", Description: "Alpha service"}),
		catalog.FromCurated(service.Service{Name: "beta", Description: "Beta service"}),
		catalog.FromCurated(service.Service{Name: "gamma", Description: "Gamma service"}),
		catalog.FromCurated(service.Service{Name: "delta", Description: "Delta service"}),
		catalog.FromCurated(service.Service{Name: "epsilon", Description: "Epsilon service"}),
	}
	return catalog.Merge(curated, nil)
}

// loadedServiceScreen returns a ServiceScreen with the test catalog loaded.
func loadedServiceScreen(t *testing.T, viewHeight int) *ServiceScreen {
	t.Helper()
	theme := NewTheme()
	screen := NewServiceScreen(theme, "curated", viewHeight, nil, nil)
	s, _ := screen.Update(catalogLoadedMsg{catalog: testServiceCatalog()})
	return s.(*ServiceScreen)
}

func TestNewServiceScreen(t *testing.T) {
	theme := NewTheme()
	screen := NewServiceScreen(theme, "curated", 20, nil, nil)

	assert.True(t, screen.IsLoading())
	assert.Equal(t, 0, screen.CursorPos())
	assert.Equal(t, 0, screen.OffsetPos())
	assert.Empty(t, screen.Filtered())
	assert.False(t, screen.showMarkers)
}

func TestNewServiceScreen_AllSource(t *testing.T) {
	theme := NewTheme()
	screen := NewServiceScreen(theme, "all", 20, nil, nil)
	assert.True(t, screen.showMarkers)
}

func TestServiceScreen_Init(t *testing.T) {
	theme := NewTheme()
	screen := NewServiceScreen(theme, "curated", 20, nil, nil)
	cmd := screen.Init()
	assert.NotNil(t, cmd)
}

func TestServiceScreen_CatalogLoaded(t *testing.T) {
	theme := NewTheme()
	screen := NewServiceScreen(theme, "curated", 20, nil, nil)

	cat := testServiceCatalog()
	s, _ := screen.Update(catalogLoadedMsg{catalog: cat})
	updated := s.(*ServiceScreen)

	assert.False(t, updated.IsLoading())
	assert.Equal(t, cat.Count(), len(updated.Filtered()))
}

func TestServiceScreen_CatalogLoadError(t *testing.T) {
	theme := NewTheme()
	screen := NewServiceScreen(theme, "curated", 20, nil, nil)

	s, _ := screen.Update(catalogLoadedMsg{err: errors.New("load failed")})
	updated := s.(*ServiceScreen)

	assert.False(t, updated.IsLoading())
	require.NotNil(t, updated.LoadError())
	assert.Contains(t, updated.View(), "load failed")
}

func TestServiceScreen_FilterByTyping(t *testing.T) {
	screen := loadedServiceScreen(t, 20)

	// Type "alp" to filter to just "alpha".
	var s Screen = screen
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	updated := s.(*ServiceScreen)

	assert.Equal(t, 1, len(updated.Filtered()))
	assert.Equal(t, "alpha", updated.Filtered()[0].Name)
}

func TestServiceScreen_FilterResetsOnClear(t *testing.T) {
	screen := loadedServiceScreen(t, 20)
	total := len(screen.Filtered())

	// Type then clear.
	var s Screen = screen
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	updated := s.(*ServiceScreen)
	assert.Equal(t, 0, len(updated.Filtered()))

	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	updated = s.(*ServiceScreen)
	assert.Equal(t, total, len(updated.Filtered()))
}

func TestServiceScreen_FilterResetsCursor(t *testing.T) {
	screen := loadedServiceScreen(t, 20)

	// Move cursor down.
	var s Screen = screen
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := s.(*ServiceScreen)
	assert.Equal(t, 2, updated.CursorPos())

	// Typing resets cursor to 0.
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	updated = s.(*ServiceScreen)
	assert.Equal(t, 0, updated.CursorPos())
}

func TestServiceScreen_NavigateDown(t *testing.T) {
	screen := loadedServiceScreen(t, 20)

	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := s.(*ServiceScreen)
	assert.Equal(t, 1, updated.CursorPos())
}

func TestServiceScreen_NavigateUp(t *testing.T) {
	screen := loadedServiceScreen(t, 20)

	var s Screen = screen
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated := s.(*ServiceScreen)
	assert.Equal(t, 1, updated.CursorPos())
}

func TestServiceScreen_NavigateUpAtTop(t *testing.T) {
	screen := loadedServiceScreen(t, 20)

	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyUp})
	updated := s.(*ServiceScreen)
	assert.Equal(t, 0, updated.CursorPos())
}

func TestServiceScreen_NavigateDownAtBottom(t *testing.T) {
	screen := loadedServiceScreen(t, 20)
	total := len(screen.Filtered())

	var s Screen = screen
	for i := 0; i < total+5; i++ {
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	updated := s.(*ServiceScreen)
	assert.Equal(t, total-1, updated.CursorPos())
}

func TestServiceScreen_EnterSelectsEntry(t *testing.T) {
	screen := loadedServiceScreen(t, 20)

	// Move to second entry.
	s, _ := screen.Update(tea.KeyMsg{Type: tea.KeyDown})
	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	sel, ok := msg.(serviceSelectMsg)
	require.True(t, ok)
	// Entries are sorted alphabetically; second is "beta".
	assert.Equal(t, "beta", sel.entry.Name)
}

func TestServiceScreen_EnterOnEmpty(t *testing.T) {
	screen := loadedServiceScreen(t, 20)

	// Filter to no results.
	var s Screen = screen
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// No command when nothing to select.
	assert.Nil(t, cmd)
}

func TestServiceScreen_EscGoesBack(t *testing.T) {
	screen := loadedServiceScreen(t, 20)

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(BackMsg)
	assert.True(t, ok)
}

func TestServiceScreen_EscDuringLoading(t *testing.T) {
	theme := NewTheme()
	screen := NewServiceScreen(theme, "curated", 20, nil, nil)

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(BackMsg)
	assert.True(t, ok)
}

func TestServiceScreen_KeysIgnoredDuringLoading(t *testing.T) {
	theme := NewTheme()
	screen := NewServiceScreen(theme, "curated", 20, nil, nil)

	// Non-esc keys should be ignored during loading.
	s, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Nil(t, cmd)
	updated := s.(*ServiceScreen)
	assert.True(t, updated.IsLoading())
}

func TestServiceScreen_ScrollDown(t *testing.T) {
	// Small viewport: viewHeight=7, headerLines=3, so 4 lines → 2 entries visible.
	screen := loadedServiceScreen(t, 7)
	assert.Equal(t, 0, screen.OffsetPos())

	// Move cursor past visible range.
	var s Screen = screen
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := s.(*ServiceScreen)

	assert.Equal(t, 2, updated.CursorPos())
	assert.True(t, updated.OffsetPos() > 0)
}

func TestServiceScreen_ScrollUp(t *testing.T) {
	screen := loadedServiceScreen(t, 7)

	// Scroll down first.
	var s Screen = screen
	for i := 0; i < 4; i++ {
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	updated := s.(*ServiceScreen)
	assert.True(t, updated.OffsetPos() > 0)

	// Scroll back up.
	for i := 0; i < 4; i++ {
		s, _ = s.Update(tea.KeyMsg{Type: tea.KeyUp})
	}
	updated = s.(*ServiceScreen)
	assert.Equal(t, 0, updated.CursorPos())
	assert.Equal(t, 0, updated.OffsetPos())
}

func TestServiceScreen_ViewShowsEntries(t *testing.T) {
	screen := loadedServiceScreen(t, 20)

	view := screen.View()
	assert.Contains(t, view, "alpha")
	assert.Contains(t, view, "beta")
}

func TestServiceScreen_ViewShowsDescriptions(t *testing.T) {
	screen := loadedServiceScreen(t, 20)

	view := screen.View()
	assert.Contains(t, view, "Alpha service")
	assert.Contains(t, view, "Beta service")
}

func TestServiceScreen_ViewShowsCount(t *testing.T) {
	screen := loadedServiceScreen(t, 20)

	view := screen.View()
	assert.Contains(t, view, "5 services")
}

func TestServiceScreen_ViewShowsMatchCount(t *testing.T) {
	screen := loadedServiceScreen(t, 20)

	var s Screen = screen
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	updated := s.(*ServiceScreen)

	view := updated.View()
	assert.Contains(t, view, "matches")
}

func TestServiceScreen_ViewShowsNoMatches(t *testing.T) {
	screen := loadedServiceScreen(t, 20)

	var s Screen = screen
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	updated := s.(*ServiceScreen)

	view := updated.View()
	assert.Contains(t, view, "No matching services")
}

func TestServiceScreen_ViewShowsLoadingState(t *testing.T) {
	theme := NewTheme()
	screen := NewServiceScreen(theme, "curated", 20, nil, nil)

	view := screen.View()
	assert.Contains(t, view, "Loading")
}

func TestServiceScreen_ViewShowsScrollIndicator(t *testing.T) {
	screen := loadedServiceScreen(t, 7)

	view := screen.View()
	assert.Contains(t, view, "more")
}

func TestServiceScreen_ViewShowsMarkers(t *testing.T) {
	theme := NewTheme()
	screen := NewServiceScreen(theme, "all", 20, nil, nil)

	curated := []catalog.Entry{
		catalog.FromCurated(service.Service{Name: "sentry", Description: "Error tracking"}),
	}
	reg := []catalog.Entry{
		{Source: catalog.SourceRegistry, Name: "community-svc"},
	}
	cat := catalog.Merge(curated, reg)

	s, _ := screen.Update(catalogLoadedMsg{catalog: cat})
	updated := s.(*ServiceScreen)

	view := updated.View()
	assert.Contains(t, view, "* sentry")
}

func TestServiceScreen_SyncStatusUpdates(t *testing.T) {
	theme := NewTheme()
	screen := NewServiceScreen(theme, "curated", 20, nil, nil)

	s, cmd := screen.Update(syncStatusMsg{status: "Registry sync in background"})
	updated := s.(*ServiceScreen)

	assert.Equal(t, "Registry sync in background", updated.SyncStatusText())
	assert.NotNil(t, cmd) // schedules next tick
}

func TestServiceScreen_SyncStatusStopsPolling(t *testing.T) {
	theme := NewTheme()
	screen := NewServiceScreen(theme, "curated", 20, nil, nil)

	s, cmd := screen.Update(syncStatusMsg{status: ""})
	updated := s.(*ServiceScreen)

	assert.Empty(t, updated.SyncStatusText())
	assert.Nil(t, cmd) // no more ticks
}

func TestServiceScreen_SyncStatusInView(t *testing.T) {
	screen := loadedServiceScreen(t, 20)
	screen.width = 80

	s, _ := screen.Update(syncStatusMsg{status: "syncing..."})
	updated := s.(*ServiceScreen)

	view := updated.View()
	assert.Contains(t, view, "syncing...")
	assert.Contains(t, view, "5 services")
}

func TestServiceScreen_StatusHints(t *testing.T) {
	screen := loadedServiceScreen(t, 20)

	hints := screen.StatusHints()
	assert.Len(t, hints, 3)

	descs := make([]string, len(hints))
	for i, h := range hints {
		descs[i] = h.Desc
	}
	assert.Contains(t, descs, "move")
	assert.Contains(t, descs, "select")
	assert.Contains(t, descs, "back")
}

func TestServiceScreen_StatusHintsDuringLoading(t *testing.T) {
	theme := NewTheme()
	screen := NewServiceScreen(theme, "curated", 20, nil, nil)

	hints := screen.StatusHints()
	assert.Len(t, hints, 1)
	assert.Equal(t, "back", hints[0].Desc)
}

func TestServiceScreen_WindowResize(t *testing.T) {
	screen := loadedServiceScreen(t, 20)

	s, _ := screen.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	updated := s.(*ServiceScreen)

	assert.Equal(t, 100, updated.width)
	assert.Equal(t, 30-ChromeLines, updated.viewHeight)
}
