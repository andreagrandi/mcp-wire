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
	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
)

func testMockTargets() []targetpkg.Target {
	return []targetpkg.Target{
		&mockTarget{name: "Claude Code", slug: "claude", installed: true},
		&mockTarget{name: "Codex", slug: "codex", installed: true},
		&mockTarget{name: "Gemini CLI", slug: "geminicli", installed: false},
	}
}

func testMockTargetsWithScopes() []targetpkg.Target {
	return []targetpkg.Target{
		&mockTarget{
			name: "Claude Code", slug: "claude", installed: true,
			scopes: []targetpkg.ConfigScope{targetpkg.ConfigScopeUser, targetpkg.ConfigScopeProject},
		},
		&mockTarget{name: "Codex", slug: "codex", installed: true},
	}
}

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
		AllTargets: testMockTargets,
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

func TestWizardModel_ServiceSelectShowsTargetScreen(t *testing.T) {
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
	_, isTarget := wm.screen.(*TargetScreen)
	assert.True(t, isTarget)
}

func TestWizardModel_ServiceSelectBreadcrumb(t *testing.T) {
	model := NewWizardModel(testCallbacksWithRegistry(), "1.0.0")
	model.height = 20

	// Source → Service.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)
	updated, _ = wm.Update(sourceSelectMsg{source: "curated"})
	wm = updated.(WizardModel)

	// Select service → goes to target screen.
	entry := catalog.FromCurated(service.Service{Name: "sentry"})
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	require.Len(t, wm.steps, 3)
	assert.True(t, wm.steps[0].Completed)
	assert.Equal(t, "Curated", wm.steps[0].Value)
	assert.True(t, wm.steps[1].Completed)
	assert.Equal(t, "sentry", wm.steps[1].Value)
	assert.True(t, wm.steps[2].Active)
	assert.Equal(t, "Targets", wm.steps[2].Label)
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

	// Select a curated entry — should skip trust and go to target screen.
	entry := catalog.FromCurated(service.Service{Name: "sentry"})
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	_, isTarget := wm.screen.(*TargetScreen)
	assert.True(t, isTarget)
}

func TestWizardModel_TrustConfirmYesShowsTargetScreen(t *testing.T) {
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

	_, isTarget := wm.screen.(*TargetScreen)
	assert.True(t, isTarget)
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

// --- Target and scope screen integration tests ---

func testCallbacksWithTargets(targets []targetpkg.Target) Callbacks {
	cb := testCallbacks()
	cb.AllTargets = func() []targetpkg.Target { return targets }
	return cb
}

func TestWizardModel_TargetScreenBreadcrumb(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")
	model.height = 20

	// Navigate to target screen.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	entry := catalog.FromCurated(service.Service{Name: "sentry"})
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	// Should be on target screen with breadcrumb.
	_, isTarget := wm.screen.(*TargetScreen)
	require.True(t, isTarget)

	// Breadcrumb: Service ✓ › Targets (active)
	require.Len(t, wm.steps, 2)
	assert.True(t, wm.steps[0].Completed)
	assert.Equal(t, "sentry", wm.steps[0].Value)
	assert.True(t, wm.steps[1].Active)
	assert.Equal(t, "Targets", wm.steps[1].Label)
}

func TestWizardModel_TargetSelectNoScopeGoesToReview(t *testing.T) {
	// Targets without scope support → skip scope screen.
	model := NewWizardModel(testCallbacks(), "1.0.0")
	model.height = 20

	// Navigate to target screen.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	entry := catalog.FromCurated(service.Service{Name: "sentry"})
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	// Select targets (no scope support).
	targets := testMockTargets()[:2] // claude and codex, no scope support
	updated, _ = wm.Update(targetSelectMsg{targets: targets})
	wm = updated.(WizardModel)

	// Should skip scope and go to review screen.
	_, isReview := wm.screen.(*ReviewScreen)
	assert.True(t, isReview)
	assert.Equal(t, targetpkg.ConfigScopeUser, wm.state.Scope)
	assert.Len(t, wm.state.Targets, 2)
}

func TestWizardModel_TargetSelectWithScopeShowsScopeScreen(t *testing.T) {
	cb := testCallbacksWithTargets(testMockTargetsWithScopes())
	model := NewWizardModel(cb, "1.0.0")
	model.height = 20

	// Navigate to target screen.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	entry := catalog.FromCurated(service.Service{Name: "sentry"})
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	// Select targets that support scopes.
	updated, _ = wm.Update(targetSelectMsg{targets: testMockTargetsWithScopes()})
	wm = updated.(WizardModel)

	_, isScope := wm.screen.(*ScopeScreen)
	assert.True(t, isScope)
}

func TestWizardModel_ScopeBreadcrumb(t *testing.T) {
	cb := testCallbacksWithTargets(testMockTargetsWithScopes())
	model := NewWizardModel(cb, "1.0.0")
	model.height = 20

	// Navigate through to scope screen.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	entry := catalog.FromCurated(service.Service{Name: "sentry"})
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	updated, _ = wm.Update(targetSelectMsg{targets: testMockTargetsWithScopes()})
	wm = updated.(WizardModel)

	// Breadcrumb: Service ✓ › Targets ✓ › Scope (active)
	require.Len(t, wm.steps, 3)
	assert.True(t, wm.steps[0].Completed)
	assert.Equal(t, "sentry", wm.steps[0].Value)
	assert.True(t, wm.steps[1].Completed)
	assert.Equal(t, "Targets", wm.steps[1].Label)
	assert.NotEmpty(t, wm.steps[1].Value)
	assert.True(t, wm.steps[2].Active)
	assert.Equal(t, "Scope", wm.steps[2].Label)
}

func TestWizardModel_ScopeSelectGoesToReview(t *testing.T) {
	cb := testCallbacksWithTargets(testMockTargetsWithScopes())
	model := NewWizardModel(cb, "1.0.0")
	model.height = 20

	// Navigate through to scope screen.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	entry := catalog.FromCurated(service.Service{Name: "sentry"})
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	updated, _ = wm.Update(targetSelectMsg{targets: testMockTargetsWithScopes()})
	wm = updated.(WizardModel)

	// Select project scope.
	updated, _ = wm.Update(scopeSelectMsg{scope: targetpkg.ConfigScopeProject})
	wm = updated.(WizardModel)

	_, isReview := wm.screen.(*ReviewScreen)
	assert.True(t, isReview)
	assert.Equal(t, targetpkg.ConfigScopeProject, wm.state.Scope)
}

func TestWizardModel_BackFromTargetGoesToService(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")
	model.height = 20

	// Navigate to target screen.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	entry := catalog.FromCurated(service.Service{Name: "sentry"})
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	_, isTarget := wm.screen.(*TargetScreen)
	require.True(t, isTarget)

	// Back from target goes to service.
	updated, _ = wm.Update(BackMsg{})
	wm = updated.(WizardModel)

	_, isService := wm.screen.(*ServiceScreen)
	assert.True(t, isService)
	assert.Empty(t, wm.state.Entry.Name)
	assert.Nil(t, wm.state.Targets)
}

func TestWizardModel_BackFromScopeGoesToTarget(t *testing.T) {
	cb := testCallbacksWithTargets(testMockTargetsWithScopes())
	model := NewWizardModel(cb, "1.0.0")
	model.height = 20

	// Navigate through to scope screen.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	entry := catalog.FromCurated(service.Service{Name: "sentry"})
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	targets := testMockTargetsWithScopes()
	updated, _ = wm.Update(targetSelectMsg{targets: targets})
	wm = updated.(WizardModel)

	_, isScope := wm.screen.(*ScopeScreen)
	require.True(t, isScope)

	// Back from scope goes to target screen.
	updated, _ = wm.Update(BackMsg{})
	wm = updated.(WizardModel)

	_, isTarget := wm.screen.(*TargetScreen)
	assert.True(t, isTarget)

	// Targets should be preserved for re-selection.
	assert.Len(t, wm.state.Targets, len(targets))
}

func TestWizardModel_TargetSummary(t *testing.T) {
	targets := testMockTargets()

	assert.Equal(t, "", targetSummary(nil))
	assert.Equal(t, "Claude Code", targetSummary(targets[:1]))
	assert.Equal(t, "Claude Code +1", targetSummary(targets[:2]))
	assert.Equal(t, "Claude Code +2", targetSummary(targets[:3]))
}

func TestWizardModel_AnyTargetSupportsProjectScope(t *testing.T) {
	assert.False(t, anyTargetSupportsProjectScope(testMockTargets()))
	assert.True(t, anyTargetSupportsProjectScope(testMockTargetsWithScopes()))
	assert.False(t, anyTargetSupportsProjectScope(nil))
}

func TestWizardModel_ReviewBreadcrumbNoScope(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")
	model.height = 20

	// Full flow without scope.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	entry := catalog.FromCurated(service.Service{Name: "sentry"})
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	targets := testMockTargets()[:2]
	updated, _ = wm.Update(targetSelectMsg{targets: targets})
	wm = updated.(WizardModel)

	// Review screen: Service ✓, Targets ✓ (no scope step).
	_, isReview := wm.screen.(*ReviewScreen)
	require.True(t, isReview)
	require.Len(t, wm.steps, 2)
	assert.True(t, wm.steps[0].Completed)
	assert.True(t, wm.steps[1].Completed)
}

func TestWizardModel_ReviewBreadcrumbWithProjectScope(t *testing.T) {
	cb := testCallbacksWithTargets(testMockTargetsWithScopes())
	model := NewWizardModel(cb, "1.0.0")
	model.height = 20

	// Full flow with scope.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	entry := catalog.FromCurated(service.Service{Name: "sentry"})
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	updated, _ = wm.Update(targetSelectMsg{targets: testMockTargetsWithScopes()})
	wm = updated.(WizardModel)

	updated, _ = wm.Update(scopeSelectMsg{scope: targetpkg.ConfigScopeProject})
	wm = updated.(WizardModel)

	// Review screen: Service ✓, Targets ✓, Scope ✓.
	_, isReview := wm.screen.(*ReviewScreen)
	require.True(t, isReview)
	require.Len(t, wm.steps, 3)
	assert.True(t, wm.steps[0].Completed)
	assert.True(t, wm.steps[1].Completed)
	assert.True(t, wm.steps[2].Completed)
	assert.Equal(t, "project", wm.steps[2].Value)
}

func TestWizardModel_NilAllTargetsCallback(t *testing.T) {
	cb := testCallbacks()
	cb.AllTargets = nil
	model := NewWizardModel(cb, "1.0.0")
	model.height = 20

	// Navigate to target screen.
	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	entry := catalog.FromCurated(service.Service{Name: "sentry"})
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	// Should show target screen with empty list.
	_, isTarget := wm.screen.(*TargetScreen)
	assert.True(t, isTarget)
}

// --- Review screen integration tests ---

func navigateToReview(t *testing.T, cb Callbacks) WizardModel {
	t.Helper()
	model := NewWizardModel(cb, "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	entry := catalog.FromCurated(service.Service{Name: "sentry", Description: "Error tracking"})
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	targets := testMockTargets()[:2]
	updated, _ = wm.Update(targetSelectMsg{targets: targets})
	wm = updated.(WizardModel)

	_, isReview := wm.screen.(*ReviewScreen)
	require.True(t, isReview)
	return wm
}

func TestWizardModel_ReviewApplyGoesToApplyScreen(t *testing.T) {
	wm := navigateToReview(t, testCallbacks())

	updated, _ := wm.Update(reviewConfirmMsg{confirmed: true})
	wm = updated.(WizardModel)

	// Apply goes to the apply screen.
	_, isApply := wm.screen.(*ApplyScreen)
	assert.True(t, isApply)
}

func TestWizardModel_ReviewCancelGoesBackToTarget(t *testing.T) {
	wm := navigateToReview(t, testCallbacks())

	updated, _ := wm.Update(reviewConfirmMsg{confirmed: false})
	wm = updated.(WizardModel)

	// Cancel goes back to target screen (no scope support).
	_, isTarget := wm.screen.(*TargetScreen)
	assert.True(t, isTarget)
}

func TestWizardModel_ReviewCancelGoesBackToScope(t *testing.T) {
	cb := testCallbacksWithTargets(testMockTargetsWithScopes())
	model := NewWizardModel(cb, "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	entry := catalog.FromCurated(service.Service{Name: "sentry"})
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	updated, _ = wm.Update(targetSelectMsg{targets: testMockTargetsWithScopes()})
	wm = updated.(WizardModel)

	updated, _ = wm.Update(scopeSelectMsg{scope: targetpkg.ConfigScopeProject})
	wm = updated.(WizardModel)

	_, isReview := wm.screen.(*ReviewScreen)
	require.True(t, isReview)

	// Cancel goes back to scope screen when targets support scopes.
	updated, _ = wm.Update(reviewConfirmMsg{confirmed: false})
	wm = updated.(WizardModel)

	_, isScope := wm.screen.(*ScopeScreen)
	assert.True(t, isScope)
	assert.Empty(t, wm.state.Scope) // scope cleared
}

func TestWizardModel_ReviewBackMsgBehavesLikeCancel(t *testing.T) {
	wm := navigateToReview(t, testCallbacks())

	updated, _ := wm.Update(BackMsg{})
	wm = updated.(WizardModel)

	_, isTarget := wm.screen.(*TargetScreen)
	assert.True(t, isTarget)
}

func TestWizardModel_ReviewViewShowsSummary(t *testing.T) {
	wm := navigateToReview(t, testCallbacks())

	view := wm.screen.View()
	assert.Contains(t, view, "Install")
	assert.Contains(t, view, "sentry")
	assert.Contains(t, view, "Claude Code")
	assert.Contains(t, view, "mcp-wire install sentry")
}

func TestWizardModel_ReviewUninstallFlow(t *testing.T) {
	model := NewWizardModel(testCallbacks(), "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Uninstall service"})
	wm := updated.(WizardModel)

	entry := catalog.FromCurated(service.Service{Name: "sentry"})
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	targets := testMockTargets()[:1]
	updated, _ = wm.Update(targetSelectMsg{targets: targets})
	wm = updated.(WizardModel)

	_, isReview := wm.screen.(*ReviewScreen)
	require.True(t, isReview)

	view := wm.screen.View()
	assert.Contains(t, view, "Uninstall")
	assert.Contains(t, view, "mcp-wire uninstall sentry")
	assert.NotContains(t, view, "Credentials")
}

// --- Credential and Apply screen integration tests ---

func testCallbacksWithCredentials() Callbacks {
	cb := testCallbacks()
	cb.CatalogEntryToService = func(e catalog.Entry) (service.Service, bool) {
		if e.Curated != nil {
			return *e.Curated, true
		}
		return service.Service{}, false
	}
	cb.ResolveCredential = func(envName string) (string, string, bool) {
		return "", "", false // nothing pre-resolved
	}
	cb.InstallTarget = func(_ service.Service, _ map[string]string, _ targetpkg.Target, _ targetpkg.ConfigScope) error {
		return nil
	}
	cb.UninstallTarget = func(_ string, _ targetpkg.Target, _ targetpkg.ConfigScope) error {
		return nil
	}
	return cb
}

func navigateToReviewWithEnvVars(t *testing.T) WizardModel {
	t.Helper()
	cb := testCallbacksWithCredentials()
	model := NewWizardModel(cb, "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	svc := service.Service{
		Name:        "sentry",
		Description: "Error tracking",
		Env: []service.EnvVar{
			{Name: "SENTRY_TOKEN", Description: "Auth token", Required: true},
		},
	}
	entry := catalog.FromCurated(svc)
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	targets := testMockTargets()[:1]
	updated, _ = wm.Update(targetSelectMsg{targets: targets})
	wm = updated.(WizardModel)

	_, isReview := wm.screen.(*ReviewScreen)
	require.True(t, isReview)
	return wm
}

func TestWizardModel_ReviewApplyWithCredentials(t *testing.T) {
	wm := navigateToReviewWithEnvVars(t)

	// Apply → should go to credential screen (unresolved env var).
	updated, _ := wm.Update(reviewConfirmMsg{confirmed: true})
	wm = updated.(WizardModel)

	_, isCred := wm.screen.(*CredentialScreen)
	assert.True(t, isCred)
}

func TestWizardModel_CredentialDoneGoesToApply(t *testing.T) {
	wm := navigateToReviewWithEnvVars(t)

	updated, _ := wm.Update(reviewConfirmMsg{confirmed: true})
	wm = updated.(WizardModel)

	_, isCred := wm.screen.(*CredentialScreen)
	require.True(t, isCred)

	// Simulate credential done.
	resolved := map[string]string{"SENTRY_TOKEN": "tok123"}
	updated, _ = wm.Update(credentialDoneMsg{resolvedEnv: resolved})
	wm = updated.(WizardModel)

	_, isApply := wm.screen.(*ApplyScreen)
	assert.True(t, isApply)
	assert.Equal(t, "tok123", wm.state.ResolvedEnv["SENTRY_TOKEN"])
}

func TestWizardModel_NoCredentialsNeeded_SkipsToApply(t *testing.T) {
	cb := testCallbacksWithCredentials()
	model := NewWizardModel(cb, "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	// Service with no env vars.
	svc := service.Service{Name: "context7", Description: "Docs MCP"}
	entry := catalog.FromCurated(svc)
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	targets := testMockTargets()[:1]
	updated, _ = wm.Update(targetSelectMsg{targets: targets})
	wm = updated.(WizardModel)

	// Apply → should skip credentials and go directly to apply.
	updated, _ = wm.Update(reviewConfirmMsg{confirmed: true})
	wm = updated.(WizardModel)

	_, isApply := wm.screen.(*ApplyScreen)
	assert.True(t, isApply)
}

func TestWizardModel_PreResolvedCredentials_SkipsToApply(t *testing.T) {
	cb := testCallbacksWithCredentials()
	cb.ResolveCredential = func(envName string) (string, string, bool) {
		if envName == "SENTRY_TOKEN" {
			return "pre-resolved-tok", "env", true
		}
		return "", "", false
	}
	model := NewWizardModel(cb, "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	svc := service.Service{
		Name: "sentry",
		Env:  []service.EnvVar{{Name: "SENTRY_TOKEN", Required: true}},
	}
	entry := catalog.FromCurated(svc)
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	targets := testMockTargets()[:1]
	updated, _ = wm.Update(targetSelectMsg{targets: targets})
	wm = updated.(WizardModel)

	// Apply → should skip credentials (already resolved).
	updated, _ = wm.Update(reviewConfirmMsg{confirmed: true})
	wm = updated.(WizardModel)

	_, isApply := wm.screen.(*ApplyScreen)
	assert.True(t, isApply)
	assert.Equal(t, "pre-resolved-tok", wm.state.ResolvedEnv["SENTRY_TOKEN"])
}

func TestWizardModel_UninstallSkipsCredentials(t *testing.T) {
	cb := testCallbacksWithCredentials()
	model := NewWizardModel(cb, "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Uninstall service"})
	wm := updated.(WizardModel)

	svc := service.Service{
		Name: "sentry",
		Env:  []service.EnvVar{{Name: "SENTRY_TOKEN", Required: true}},
	}
	entry := catalog.FromCurated(svc)
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	targets := testMockTargets()[:1]
	updated, _ = wm.Update(targetSelectMsg{targets: targets})
	wm = updated.(WizardModel)

	// Apply → uninstall should skip credentials entirely.
	updated, _ = wm.Update(reviewConfirmMsg{confirmed: true})
	wm = updated.(WizardModel)

	_, isApply := wm.screen.(*ApplyScreen)
	assert.True(t, isApply)
}

func TestWizardModel_BackFromCredentials(t *testing.T) {
	wm := navigateToReviewWithEnvVars(t)

	updated, _ := wm.Update(reviewConfirmMsg{confirmed: true})
	wm = updated.(WizardModel)

	_, isCred := wm.screen.(*CredentialScreen)
	require.True(t, isCred)

	// Back from credentials goes to review.
	updated, _ = wm.Update(BackMsg{})
	wm = updated.(WizardModel)

	_, isReview := wm.screen.(*ReviewScreen)
	assert.True(t, isReview)
	assert.Nil(t, wm.state.ResolvedEnv)
}

func TestWizardModel_BackFromApplyBlocked(t *testing.T) {
	cb := testCallbacksWithCredentials()
	model := NewWizardModel(cb, "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	svc := service.Service{Name: "context7"}
	entry := catalog.FromCurated(svc)
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	targets := testMockTargets()[:1]
	updated, _ = wm.Update(targetSelectMsg{targets: targets})
	wm = updated.(WizardModel)

	updated, _ = wm.Update(reviewConfirmMsg{confirmed: true})
	wm = updated.(WizardModel)

	_, isApply := wm.screen.(*ApplyScreen)
	require.True(t, isApply)

	// Back from apply should be blocked (still apply screen).
	updated, _ = wm.Update(BackMsg{})
	wm = updated.(WizardModel)

	_, stillApply := wm.screen.(*ApplyScreen)
	assert.True(t, stillApply)
}

func TestWizardModel_ApplyPostActionAnother(t *testing.T) {
	wm := navigateToReview(t, testCallbacksWithCredentials())

	updated, _ := wm.Update(reviewConfirmMsg{confirmed: true})
	wm = updated.(WizardModel)

	// "Another" restarts wizard.
	updated, _ = wm.Update(applyPostActionMsg{action: "another"})
	wm = updated.(WizardModel)

	_, isService := wm.screen.(*ServiceScreen)
	assert.True(t, isService)
	assert.Equal(t, "install", wm.state.Action)
}

func TestWizardModel_ApplyPostActionMenu(t *testing.T) {
	wm := navigateToReview(t, testCallbacksWithCredentials())

	updated, _ := wm.Update(reviewConfirmMsg{confirmed: true})
	wm = updated.(WizardModel)

	// "Menu" goes back to main menu.
	updated, _ = wm.Update(applyPostActionMsg{action: "menu"})
	wm = updated.(WizardModel)

	_, isMenu := wm.screen.(*MenuScreen)
	assert.True(t, isMenu)
	assert.Empty(t, wm.state.Action)
}

func TestWizardModel_ApplyPostActionExit(t *testing.T) {
	wm := navigateToReview(t, testCallbacksWithCredentials())

	updated, _ := wm.Update(reviewConfirmMsg{confirmed: true})
	wm = updated.(WizardModel)

	// "Exit" sends quit.
	_, cmd := wm.Update(applyPostActionMsg{action: "exit"})
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestWizardModel_ConvertEntryFails_ShowsError(t *testing.T) {
	cb := testCallbacks()
	cb.CatalogEntryToService = func(_ catalog.Entry) (service.Service, bool) {
		return service.Service{}, false
	}
	model := NewWizardModel(cb, "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	entry := catalog.FromCurated(service.Service{Name: "broken"})
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	targets := testMockTargets()[:1]
	updated, _ = wm.Update(targetSelectMsg{targets: targets})
	wm = updated.(WizardModel)

	// Apply → conversion fails → error output.
	updated, _ = wm.Update(reviewConfirmMsg{confirmed: true})
	wm = updated.(WizardModel)

	_, isOutput := wm.screen.(*OutputScreen)
	assert.True(t, isOutput)
	assert.Contains(t, wm.screen.View(), "Cannot resolve service definition")
}

func TestWizardModel_CredentialBreadcrumb(t *testing.T) {
	wm := navigateToReviewWithEnvVars(t)

	updated, _ := wm.Update(reviewConfirmMsg{confirmed: true})
	wm = updated.(WizardModel)

	_, isCred := wm.screen.(*CredentialScreen)
	require.True(t, isCred)

	// Should have Credentials breadcrumb.
	found := false
	for _, step := range wm.steps {
		if step.Label == "Credentials" && step.Active {
			found = true
		}
	}
	assert.True(t, found, "expected active Credentials breadcrumb")
}

func TestWizardModel_ApplyBreadcrumb(t *testing.T) {
	cb := testCallbacksWithCredentials()
	model := NewWizardModel(cb, "1.0.0")
	model.height = 20

	updated, _ := model.Update(menuSelectMsg{item: "Install service"})
	wm := updated.(WizardModel)

	svc := service.Service{Name: "context7"}
	entry := catalog.FromCurated(svc)
	updated, _ = wm.Update(serviceSelectMsg{entry: entry})
	wm = updated.(WizardModel)

	targets := testMockTargets()[:1]
	updated, _ = wm.Update(targetSelectMsg{targets: targets})
	wm = updated.(WizardModel)

	updated, _ = wm.Update(reviewConfirmMsg{confirmed: true})
	wm = updated.(WizardModel)

	// Should have Apply breadcrumb.
	found := false
	for _, step := range wm.steps {
		if step.Label == "Apply" && step.Active {
			found = true
		}
	}
	assert.True(t, found, "expected active Apply breadcrumb")
}
