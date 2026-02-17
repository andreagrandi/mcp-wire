package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/andreagrandi/mcp-wire/internal/catalog"
)

const serviceHeaderLines = 3 // search input + count line + blank

// catalogLoadedMsg is sent when the catalog finishes loading.
type catalogLoadedMsg struct {
	catalog *catalog.Catalog
	err     error
}

// syncStatusMsg carries the latest registry sync status line.
type syncStatusMsg struct {
	status string
}

// ServiceScreen provides live-filtered search over a catalog of services.
type ServiceScreen struct {
	theme       Theme
	search      textinput.Model
	cat         *catalog.Catalog
	filtered    []catalog.Entry
	cursor      int
	offset      int
	viewHeight  int
	width       int
	source      string
	showMarkers bool
	syncStatus  string
	loading     bool
	loadErr     error
	loadFn      func(string) (*catalog.Catalog, error)
	syncFn      func() string
}

// NewServiceScreen creates a new service selection screen.
func NewServiceScreen(theme Theme, source string, viewHeight int, loadFn func(string) (*catalog.Catalog, error), syncFn func() string) *ServiceScreen {
	ti := textinput.New()
	ti.Prompt = "  Search > "
	ti.Placeholder = "type to filter..."
	ti.CharLimit = 100
	ti.Focus() // Focus immediately so keys are accepted (Init returns the blink cmd).

	return &ServiceScreen{
		theme:       theme,
		search:      ti,
		viewHeight:  viewHeight,
		source:      source,
		showMarkers: source == "all",
		loading:     true,
		loadFn:      loadFn,
		syncFn:      syncFn,
	}
}

func (s *ServiceScreen) Init() tea.Cmd {
	focusCmd := s.search.Focus()
	cmds := []tea.Cmd{focusCmd, s.loadCatalogCmd()}
	if s.syncFn != nil {
		cmds = append(cmds, s.tickSyncStatus())
	}
	return tea.Batch(cmds...)
}

func (s *ServiceScreen) loadCatalogCmd() tea.Cmd {
	loadFn := s.loadFn
	source := s.source
	return func() tea.Msg {
		if loadFn == nil {
			return catalogLoadedMsg{}
		}
		cat, err := loadFn(source)
		return catalogLoadedMsg{catalog: cat, err: err}
	}
}

func (s *ServiceScreen) tickSyncStatus() tea.Cmd {
	syncFn := s.syncFn
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return syncStatusMsg{status: syncFn()}
	})
}

func (s *ServiceScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.viewHeight = contentHeightFromTerminal(msg.Height)
		s.width = msg.Width
		return s, nil

	case catalogLoadedMsg:
		s.loading = false
		if msg.err != nil {
			s.loadErr = msg.err
			return s, nil
		}
		s.cat = msg.catalog
		s.applyFilter()
		return s, nil

	case syncStatusMsg:
		s.syncStatus = msg.status
		if msg.status != "" {
			return s, s.tickSyncStatus()
		}
		return s, nil

	case tea.KeyMsg:
		if s.loading || s.loadErr != nil {
			if msg.String() == "esc" {
				return s, func() tea.Msg { return BackMsg{} }
			}
			return s, nil
		}
		return s.handleKey(msg)
	}

	// Forward remaining messages to textinput (e.g. cursor blink).
	var cmd tea.Cmd
	s.search, cmd = s.search.Update(msg)
	return s, cmd
}

func (s *ServiceScreen) handleKey(msg tea.KeyMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "up":
		if s.cursor > 0 {
			s.cursor--
			s.ensureVisible()
		}
		return s, nil
	case "down":
		if s.cursor < len(s.filtered)-1 {
			s.cursor++
			s.ensureVisible()
		}
		return s, nil
	case "enter":
		if len(s.filtered) > 0 && s.cursor < len(s.filtered) {
			entry := s.filtered[s.cursor]
			return s, func() tea.Msg {
				return serviceSelectMsg{entry: entry}
			}
		}
		return s, nil
	case "esc":
		return s, func() tea.Msg { return BackMsg{} }
	}

	// All other keys go to search input.
	prevValue := s.search.Value()
	var cmd tea.Cmd
	s.search, cmd = s.search.Update(msg)
	if s.search.Value() != prevValue {
		s.applyFilter()
	}
	return s, cmd
}

func (s *ServiceScreen) applyFilter() {
	if s.cat == nil {
		s.filtered = nil
		s.cursor = 0
		s.offset = 0
		return
	}
	s.filtered = s.cat.Search(s.search.Value())
	s.cursor = 0
	s.offset = 0
}

func (s *ServiceScreen) maxVisibleEntries() int {
	lines := s.viewHeight - serviceHeaderLines
	if lines < 2 {
		return 1
	}
	return lines / 2
}

func (s *ServiceScreen) ensureVisible() {
	maxVisible := s.maxVisibleEntries()
	if s.cursor >= s.offset+maxVisible {
		s.offset = s.cursor - maxVisible + 1
	}
	if s.cursor < s.offset {
		s.offset = s.cursor
	}
}

func (s *ServiceScreen) View() string {
	var b strings.Builder

	// Line 1: search input.
	b.WriteString(s.search.View())
	b.WriteString("\n")

	// Line 2: count + sync status.
	b.WriteString(s.countLine())
	b.WriteString("\n")

	if s.loading {
		b.WriteString("\n")
		b.WriteString(s.theme.Dim.Render("  Loading..."))
		return b.String()
	}

	if s.loadErr != nil {
		b.WriteString("\n")
		b.WriteString(s.theme.Error.Render("  Error: " + s.loadErr.Error()))
		return b.String()
	}

	if len(s.filtered) == 0 {
		b.WriteString("\n")
		b.WriteString(s.theme.Dim.Render("  No matching services"))
		return b.String()
	}

	// Calculate visible entry range.
	maxEntries := s.maxVisibleEntries()
	end := s.offset + maxEntries
	if end > len(s.filtered) {
		end = len(s.filtered)
	}

	hasMore := end < len(s.filtered)
	if hasMore && end-s.offset > 1 {
		end-- // reserve space for scroll indicator
	}

	// Blank line before entries.
	b.WriteString("\n")

	for i := s.offset; i < end; i++ {
		entry := s.filtered[i]
		name := entry.Name

		if s.showMarkers {
			if entry.Source == catalog.SourceCurated {
				name = "* " + name
			} else {
				name = "  " + name
			}
		}

		if i == s.cursor {
			label := "  \u276f " + name
			if s.width > 0 {
				b.WriteString(s.theme.Highlight.Width(s.width).Render(label))
			} else {
				b.WriteString(s.theme.Cursor.Render(label))
			}
		} else {
			b.WriteString("    " + name)
		}
		b.WriteString("\n")

		// Description line.
		desc := entry.Description()
		if desc != "" {
			b.WriteString(s.theme.Dim.Render("      " + desc))
		}
		b.WriteString("\n")
	}

	if hasMore {
		remaining := len(s.filtered) - end
		b.WriteString(s.theme.Dim.Render("  \u25bc " + strings.Repeat(".", 3) + " " + itoa(remaining) + " more"))
	}

	return b.String()
}

func (s *ServiceScreen) countLine() string {
	if s.loading {
		return s.theme.Dim.Render("  Loading catalog...")
	}
	if s.loadErr != nil || s.cat == nil {
		return ""
	}

	total := s.cat.Count()
	filtered := len(s.filtered)

	var count string
	query := strings.TrimSpace(s.search.Value())
	if query == "" || filtered == total {
		count = itoa(total) + " services"
	} else {
		count = itoa(filtered) + " matches"
	}

	// Build the line: sync status on left, count on right.
	if s.width > 0 {
		left := ""
		if s.syncStatus != "" {
			left = "  " + s.syncStatus
		}
		right := count
		gap := s.width - len(left) - len(right)
		if gap < 1 {
			gap = 1
		}
		return s.theme.Dim.Render(left + strings.Repeat(" ", gap) + right)
	}

	// Fallback (no terminal width known).
	if s.syncStatus != "" {
		return s.theme.Dim.Render("  " + s.syncStatus + "  " + count)
	}
	return s.theme.Dim.Render("  " + count)
}

func (s *ServiceScreen) StatusHints() []KeyHint {
	if s.loading || s.loadErr != nil {
		return []KeyHint{
			{Key: "Esc", Desc: "back"},
		}
	}
	return []KeyHint{
		{Key: "\u2191\u2193", Desc: "move"},
		{Key: "Enter", Desc: "select"},
		{Key: "Esc", Desc: "back"},
	}
}

// Testing accessors.

func (s *ServiceScreen) CursorPos() int            { return s.cursor }
func (s *ServiceScreen) OffsetPos() int            { return s.offset }
func (s *ServiceScreen) Filtered() []catalog.Entry { return s.filtered }
func (s *ServiceScreen) IsLoading() bool           { return s.loading }
func (s *ServiceScreen) LoadError() error          { return s.loadErr }
func (s *ServiceScreen) SyncStatusText() string    { return s.syncStatus }
