package tui

import (
	"errors"
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andreagrandi/mcp-wire/internal/catalog"
	"github.com/andreagrandi/mcp-wire/internal/registry"
	"github.com/andreagrandi/mcp-wire/internal/service"
)

func testCallbacks() Callbacks {
	return Callbacks{
		RenderStatus: func(w io.Writer) error {
			_, err := w.Write([]byte("Status output"))
			return err
		},
		RenderServicesList: func(w io.Writer) error {
			_, err := w.Write([]byte("Services output"))
			return err
		},
		RenderTargetsList: func(w io.Writer) error {
			_, err := w.Write([]byte("Targets output"))
			return err
		},
	}
}

func TestNewWizardModel(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")

	assert.NotNil(t, model.screen)
	assert.Empty(t, model.steps)
	assert.Equal(t, 0, model.width)
	assert.Equal(t, 0, model.height)
	assert.Equal(t, "1.0.0", model.version)
}

func TestWizardModel_Init(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")
	assert.Nil(t, model.Init())
}

func TestWizardModel_WindowSizeMsg(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	wm := updated.(WizardModel)

	assert.Equal(t, 120, wm.width)
	assert.Equal(t, 40, wm.height)
}

func TestWizardModel_WindowSizeMsgForwardedToScreen(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")
	model.height = 30

	// Navigate to an output screen so we can verify resize forwarding.
	updated, _ := model.Update(menuSelectMsg{item: "Status"})
	wm := updated.(WizardModel)

	screen, ok := wm.screen.(*OutputScreen)
	require.True(t, ok)
	originalHeight := screen.viewHeight

	// Resize terminal.
	updated, _ = wm.Update(tea.WindowSizeMsg{Width: 80, Height: 50})
	wm = updated.(WizardModel)

	screen = wm.screen.(*OutputScreen)
	assert.NotEqual(t, originalHeight, screen.viewHeight)
	assert.Equal(t, 50-ChromeLines, screen.viewHeight)
}

func TestWizardModel_CtrlCQuits(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestWizardModel_ExitMenuQuits(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")

	_, cmd := model.Update(menuSelectMsg{item: "Exit"})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestWizardModel_StatusShowsOutput(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Status"})
	wm := updated.(WizardModel)

	_, isOutput := wm.screen.(*OutputScreen)
	assert.True(t, isOutput)
	assert.Contains(t, wm.screen.View(), "Status output")
}

func TestWizardModel_ListServicesShowsOutput(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "List services"})
	wm := updated.(WizardModel)

	_, isOutput := wm.screen.(*OutputScreen)
	assert.True(t, isOutput)
	assert.Contains(t, wm.screen.View(), "Services output")
}

func TestWizardModel_ListTargetsShowsOutput(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "List targets"})
	wm := updated.(WizardModel)

	_, isOutput := wm.screen.(*OutputScreen)
	assert.True(t, isOutput)
	assert.Contains(t, wm.screen.View(), "Targets output")
}

func TestWizardModel_CallbackError(t *testing.T) {
	cb := Callbacks{
		RenderStatus: func(w io.Writer) error {
			return errors.New("something went wrong")
		},
	}
	model := NewWizardModel(cb, "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Status"})
	wm := updated.(WizardModel)

	assert.Contains(t, wm.screen.View(), "Error: something went wrong")
}

func TestWizardModel_NilCallback(t *testing.T) {
	model := NewWizardModel(Callbacks{}, "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Status"})
	wm := updated.(WizardModel)

	assert.Contains(t, wm.screen.View(), "not available")
}

func TestWizardModel_InstallNoRegistry_SkipsToService(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	// Without registry, source screen is skipped and service screen shown.
	_, isService := wm.screen.(*ServiceScreen)
	assert.True(t, isService)
	assert.Equal(t, "install", wm.state.Action)
	assert.Equal(t, "curated", wm.state.Source)
}

func TestWizardModel_UninstallNoRegistry_SkipsToService(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Uninstall service"})
	wm := updated.(WizardModel)

	_, isService := wm.screen.(*ServiceScreen)
	assert.True(t, isService)
	assert.Equal(t, "uninstall", wm.state.Action)
	assert.Equal(t, "curated", wm.state.Source)
}

func testCallbacksWithRegistry() Callbacks {
	cb := testCallbacks()
	cb.RegistryEnabled = true
	return cb
}

func TestWizardModel_InstallWithRegistry_ShowsSource(t *testing.T) {
	model := NewWizardModel(testCallbacksWithRegistry(), "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	_, isSource := wm.screen.(*SourceScreen)
	assert.True(t, isSource)
	assert.Equal(t, "install", wm.state.Action)
	assert.Empty(t, wm.state.Source)
}

func TestWizardModel_UninstallWithRegistry_ShowsSource(t *testing.T) {
	model := NewWizardModel(testCallbacksWithRegistry(), "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Uninstall service"})
	wm := updated.(WizardModel)

	_, isSource := wm.screen.(*SourceScreen)
	assert.True(t, isSource)
	assert.Equal(t, "uninstall", wm.state.Action)
}

func TestWizardModel_SourceBreadcrumb(t *testing.T) {
	model := NewWizardModel(testCallbacksWithRegistry(), "1.0.0")
	model.width = 80
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	require.Len(t, wm.steps, 1)
	assert.Equal(t, "Source", wm.steps[0].Label)
	assert.True(t, wm.steps[0].Active)
}

func TestWizardModel_SourceSelectShowsServiceScreen(t *testing.T) {
	model := NewWizardModel(testCallbacksWithRegistry(), "1.0.0")
	model.height = 20

	// Navigate to source screen.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	// Select "registry" source → service screen.
	updated, _ = wm.Update(sourceSelectMsg{source: "registry"})
	wm = updated.(WizardModel)

	assert.Equal(t, "registry", wm.state.Source)
	_, isService := wm.screen.(*ServiceScreen)
	assert.True(t, isService)
}

func TestWizardModel_SourceSelectBreadcrumb(t *testing.T) {
	model := NewWizardModel(testCallbacksWithRegistry(), "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	updated, _ = wm.Update(sourceSelectMsg{source: "all"})
	wm = updated.(WizardModel)

	require.Len(t, wm.steps, 2)
	assert.True(t, wm.steps[0].Completed)
	assert.Equal(t, "Both", wm.steps[0].Value)
	assert.True(t, wm.steps[1].Active)
	assert.Equal(t, "Service", wm.steps[1].Label)
}

func TestWizardModel_ServiceBreadcrumbNoRegistry(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	// Without registry, no Source breadcrumb, just Service.
	require.Len(t, wm.steps, 1)
	assert.True(t, wm.steps[0].Active)
	assert.Equal(t, "Service", wm.steps[0].Label)
}

func TestWizardModel_ServiceSelectShowsPlaceholder(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")
	model.height = 20

	// Navigate to service screen.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	// Simulate service selection.
	entry := catalog.FromCurated(service.Service{Name: "sentry", Description: "Error tracking"})
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	assert.Equal(t, "sentry", wm.state.Entry.Name)
	_, isOutput := wm.screen.(*OutputScreen)
	assert.True(t, isOutput)
	assert.Contains(t, wm.screen.View(), "mcp-wire install sentry")
}

func TestWizardModel_ServiceSelectBreadcrumb(t *testing.T) {
	model := NewWizardModel(testCallbacksWithRegistry(), "1.0.0")
	model.height = 20

	// Source → Service.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)
	updated, _ = wm.Update(sourceSelectMsg{source: "curated"})
	wm = updated.(WizardModel)

	// Select service.
	entry := catalog.FromCurated(service.Service{Name: "sentry"})
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	require.Len(t, wm.steps, 2)
	assert.True(t, wm.steps[0].Completed)
	assert.Equal(t, "Curated", wm.steps[0].Value)
	assert.True(t, wm.steps[1].Completed)
	assert.Equal(t, "sentry", wm.steps[1].Value)
}

func TestWizardModel_BackFromSourceResetsState(t *testing.T) {
	model := NewWizardModel(testCallbacksWithRegistry(), "1.0.0")
	model.height = 20

	// Navigate to source screen.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)
	assert.Equal(t, "install", wm.state.Action)

	// Back resets state and returns to menu.
	updated, _ = wm.Update(BackMsg{})
	wm = updated.(WizardModel)

	_, isMenu := wm.screen.(*MenuScreen)
	assert.True(t, isMenu)
	assert.Empty(t, wm.state.Action)
	assert.Nil(t, wm.steps)
}

func TestWizardModel_BackFromServiceToSource(t *testing.T) {
	model := NewWizardModel(testCallbacksWithRegistry(), "1.0.0")
	model.height = 20

	// Navigate to source → select → service.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)
	updated, _ = wm.Update(sourceSelectMsg{source: "registry"})
	wm = updated.(WizardModel)

	_, isService := wm.screen.(*ServiceScreen)
	require.True(t, isService)

	// Back from service goes to source.
	updated, _ = wm.Update(BackMsg{})
	wm = updated.(WizardModel)

	_, isSource := wm.screen.(*SourceScreen)
	assert.True(t, isSource)
	assert.Equal(t, "install", wm.state.Action) // action preserved
	assert.Empty(t, wm.state.Source)            // source cleared
}

func TestWizardModel_BackFromServiceNoRegistryGoesToMenu(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")
	model.height = 20

	// Navigate to service screen (no registry).
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	_, isService := wm.screen.(*ServiceScreen)
	require.True(t, isService)

	// Back goes to menu (no source screen to return to).
	updated, _ = wm.Update(BackMsg{})
	wm = updated.(WizardModel)

	_, isMenu := wm.screen.(*MenuScreen)
	assert.True(t, isMenu)
}

func TestWizardModel_BackFromOutputReturnsToMenu(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")
	model.height = 20

	// Go to Status output.
	updated, _ := model.Update(menuSelectMsg{item: "Status"})
	wm := updated.(WizardModel)

	_, isOutput := wm.screen.(*OutputScreen)
	require.True(t, isOutput)

	// Back returns to menu.
	updated, _ = wm.Update(BackMsg{})
	wm = updated.(WizardModel)

	_, isMenu := wm.screen.(*MenuScreen)
	assert.True(t, isMenu)
}

func TestWizardModel_BackMsg(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")
	model.steps = []BreadcrumbStep{
		{Label: "Source", Active: true, Visible: true},
	}

	updated, _ := model.Update(BackMsg{})
	wm := updated.(WizardModel)

	assert.Nil(t, wm.steps)
	_, isMenu := wm.screen.(*MenuScreen)
	assert.True(t, isMenu)
}

func TestWizardModel_ViewLayout(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "0.1.3")
	model.width = 80
	model.height = 20

	view := model.View()

	assert.Contains(t, view, "mcp-wire v0.1.3")
	assert.Contains(t, view, "\u2500") // separator
	assert.Contains(t, view, "Install service")
	assert.Contains(t, view, "move")
}

func TestWizardModel_ViewNoVersion(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "")
	model.width = 80
	model.height = 20

	view := model.View()
	assert.Contains(t, view, "mcp-wire")
	assert.NotContains(t, view, "mcp-wire v")
}

func TestWizardModel_ContentHeight(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")

	// Unknown height returns default.
	assert.Equal(t, ContentHeight, model.contentHeight())

	// Known height subtracts chrome.
	model.height = 30
	assert.Equal(t, 30-ChromeLines, model.contentHeight())

	// Very small terminal.
	model.height = ChromeLines
	assert.Equal(t, 1, model.contentHeight())
}

func TestWizardModel_RegistryServiceShowsTrustScreen(t *testing.T) {
	model := NewWizardModel(testCallbacksWithRegistry(), "1.0.0")
	model.height = 20

	// Navigate to service screen.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)
	updated, _ = wm.Update(sourceSelectMsg{source: "registry"})
	wm = updated.(WizardModel)

	// Select a registry entry.
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "community-svc",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{Name: "community-svc", Description: "A service"},
		},
	}
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	_, isTrust := wm.screen.(*TrustScreen)
	assert.True(t, isTrust)
}

func TestWizardModel_CuratedServiceSkipsTrustScreen(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	// Select a curated entry — should skip trust and go to placeholder.
	entry := catalog.FromCurated(service.Service{Name: "sentry"})
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	_, isOutput := wm.screen.(*OutputScreen)
	assert.True(t, isOutput)
}

func TestWizardModel_TrustConfirmYesShowsPlaceholder(t *testing.T) {
	model := NewWizardModel(testCallbacksWithRegistry(), "1.0.0")
	model.height = 20

	// Navigate to trust screen.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)
	updated, _ = wm.Update(sourceSelectMsg{source: "registry"})
	wm = updated.(WizardModel)

	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "community-svc",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{Name: "community-svc"},
		},
	}
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	// Confirm trust.
	updated, _ = wm.Update(trustConfirmMsg{confirmed: true})
	wm = updated.(WizardModel)

	_, isOutput := wm.screen.(*OutputScreen)
	assert.True(t, isOutput)
	assert.Contains(t, wm.screen.View(), "mcp-wire install community-svc")
}

func TestWizardModel_TrustConfirmNoGoesBackToService(t *testing.T) {
	model := NewWizardModel(testCallbacksWithRegistry(), "1.0.0")
	model.height = 20

	// Navigate to trust screen.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)
	updated, _ = wm.Update(sourceSelectMsg{source: "registry"})
	wm = updated.(WizardModel)

	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "community-svc",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{Name: "community-svc"},
		},
	}
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	// Reject trust.
	updated, _ = wm.Update(trustConfirmMsg{confirmed: false})
	wm = updated.(WizardModel)

	_, isService := wm.screen.(*ServiceScreen)
	assert.True(t, isService)
}

func TestWizardModel_TrustBreadcrumb(t *testing.T) {
	model := NewWizardModel(testCallbacksWithRegistry(), "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)
	updated, _ = wm.Update(sourceSelectMsg{source: "registry"})
	wm = updated.(WizardModel)

	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "community-svc",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{Name: "community-svc"},
		},
	}
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	require.Len(t, wm.steps, 3)
	assert.True(t, wm.steps[0].Completed)
	assert.Equal(t, "Registry", wm.steps[0].Value)
	assert.True(t, wm.steps[1].Completed)
	assert.Equal(t, "community-svc", wm.steps[1].Value)
	assert.True(t, wm.steps[2].Active)
	assert.Equal(t, "Trust", wm.steps[2].Label)
}

func TestWizardModel_BackFromTrustGoesToService(t *testing.T) {
	model := NewWizardModel(testCallbacksWithRegistry(), "1.0.0")
	model.height = 20

	// Navigate to trust screen.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)
	updated, _ = wm.Update(sourceSelectMsg{source: "registry"})
	wm = updated.(WizardModel)

	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "community-svc",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{Name: "community-svc"},
		},
	}
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	_, isTrust := wm.screen.(*TrustScreen)
	require.True(t, isTrust)

	// Back from trust goes to service screen.
	updated, _ = wm.Update(BackMsg{})
	wm = updated.(WizardModel)

	_, isService := wm.screen.(*ServiceScreen)
	assert.True(t, isService)
	assert.Empty(t, wm.state.Entry.Name)
}

func TestWizardModel_TrustRefreshesEntry(t *testing.T) {
	cb := testCallbacksWithRegistry()
	refreshed := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "community-svc",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:        "community-svc",
				Description: "Refreshed description",
			},
		},
	}
	cb.RefreshRegistryEntry = func(_ catalog.Entry) catalog.Entry {
		return refreshed
	}

	model := NewWizardModel(cb, "1.0.0")
	model.height = 20

	// Navigate to trust screen.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)
	updated, _ = wm.Update(sourceSelectMsg{source: "registry"})
	wm = updated.(WizardModel)

	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "community-svc",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{Name: "community-svc"},
		},
	}
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	// Confirm trust — entry should be refreshed.
	updated, _ = wm.Update(trustConfirmMsg{confirmed: true})
	wm = updated.(WizardModel)

	assert.Equal(t, "Refreshed description", wm.state.Entry.Description())
}

func TestWizardModel_ViewNoBreadcrumbOnMenu(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")
	model.width = 80
	model.height = 20

	view := model.View()

	// No breadcrumb separator on main menu.
	assert.NotContains(t, view, "\u203a")
}

func TestContentHeightFromTerminal(t *testing.T) {
	assert.Equal(t, ContentHeight, contentHeightFromTerminal(0))
	assert.Equal(t, ContentHeight, contentHeightFromTerminal(-1))
	assert.Equal(t, 1, contentHeightFromTerminal(1))
	assert.Equal(t, 1, contentHeightFromTerminal(ChromeLines))
	assert.Equal(t, 20-ChromeLines, contentHeightFromTerminal(20))
	assert.Equal(t, 50-ChromeLines, contentHeightFromTerminal(50))
}

func TestPadToHeight(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		target    int
		wantLines int
	}{
		{"empty to 5", "", 5, 5},
		{"one line to 5", "hello", 5, 5},
		{"trailing newline", "hello\n", 5, 5},
		{"exact match", "a\nb\nc", 3, 3},
		{"truncate", "a\nb\nc\nd\ne", 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padToHeight(tt.content, tt.target)
			lines := len(splitLines(result))
			assert.Equal(t, tt.wantLines, lines)
		})
	}
}

func splitLines(s string) []string {
	if s == "" {
		return []string{""}
	}

	lines := []string{}
	start := 0

	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}

	lines = append(lines, s[start:])
	return lines
}
