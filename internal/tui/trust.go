package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/andreagrandi/mcp-wire/internal/catalog"
)

// trustConfirmMsg is sent when the user confirms or rejects the trust warning.
type trustConfirmMsg struct {
	confirmed bool
}

// TrustScreen displays registry entry metadata and asks for explicit
// confirmation before proceeding with installation.
type TrustScreen struct {
	theme  Theme
	entry  catalog.Entry
	cursor int // 0 = No, 1 = Yes
	width  int
}

// NewTrustScreen creates a trust warning screen for the given entry.
func NewTrustScreen(theme Theme, entry catalog.Entry) *TrustScreen {
	return &TrustScreen{
		theme: theme,
		entry: entry,
	}
}

func (t *TrustScreen) Init() tea.Cmd { return nil }

func (t *TrustScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		return t, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			if t.cursor > 0 {
				t.cursor--
			}
		case "right", "l":
			if t.cursor < 1 {
				t.cursor++
			}
		case "enter":
			confirmed := t.cursor == 1
			return t, func() tea.Msg {
				return trustConfirmMsg{confirmed: confirmed}
			}
		case "esc":
			return t, func() tea.Msg { return BackMsg{} }
		}
	}

	return t, nil
}

func (t *TrustScreen) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(t.theme.Warning.Render("  \u26a0 Registry Service \u2014 not curated by mcp-wire"))
	b.WriteString("\n\n")

	// Service name and description.
	displayName := t.entry.DisplayName()
	if displayName != "" {
		b.WriteString("  " + t.theme.Active.Render(displayName) + "\n")
	}
	if desc := t.entry.Description(); desc != "" {
		b.WriteString("  " + desc + "\n")
	}
	if displayName != "" || t.entry.Description() != "" {
		b.WriteString("\n")
	}

	// Metadata lines.
	b.WriteString(t.metaLine("Source", string(t.entry.Source)+" (community, not vetted by mcp-wire)"))

	if installType := t.entry.InstallType(); installType != "" {
		b.WriteString(t.metaLine("Install", installType))
	}

	if t.entry.HasPackages() && t.entry.Registry != nil {
		pkg := t.entry.Registry.Server.Packages[0]
		identifier := pkg.Identifier
		if pkg.Version != "" {
			identifier += "@" + pkg.Version
		}

		b.WriteString(t.metaLine("Package", pkg.RegistryType+" ("+identifier+")"))

		if pkg.RuntimeHint != "" {
			b.WriteString(t.metaLine("Runtime", pkg.RuntimeHint))
		}
	}

	if transport := t.entry.Transport(); transport != "" {
		b.WriteString(t.metaLine("Transport", transport))
	}

	// Remote URL.
	if t.entry.HasRemotes() && t.entry.Registry != nil && len(t.entry.Registry.Server.Remotes) > 0 {
		url := t.entry.Registry.Server.Remotes[0].URL
		if url != "" {
			b.WriteString(t.metaLine("URL", t.theme.Active.Render(url)))
		}
	}

	var secretNames []string
	for _, v := range t.entry.EnvVars() {
		if v.Required {
			secretNames = append(secretNames, v.Name)
		}
	}
	if len(secretNames) > 0 {
		b.WriteString(t.metaLine("Secrets", strings.Join(secretNames, ", ")))
	}

	if repoURL := t.entry.RepositoryURL(); repoURL != "" {
		b.WriteString(t.metaLine("Repo", repoURL))
	}

	// Caution text.
	b.WriteString("\n")
	b.WriteString(t.theme.Warning.Render("  Registry services are community-published. Review before proceeding."))
	b.WriteString("\n\n")

	// Confirmation prompt.
	b.WriteString("  Proceed with this registry service?\n\n")
	b.WriteString(t.renderChoices())

	return b.String()
}

func (t *TrustScreen) metaLine(label, value string) string {
	return t.theme.Dim.Render("  "+label+":") + "  " + value + "\n"
}

func (t *TrustScreen) renderChoices() string {
	labels := []string{"No, go back", "Yes, proceed"}
	var parts []string

	for i, label := range labels {
		if i == t.cursor {
			if t.width > 0 {
				parts = append(parts, t.theme.Highlight.Render(" "+label+" "))
			} else {
				parts = append(parts, t.theme.Cursor.Render("["+label+"]"))
			}
		} else {
			parts = append(parts, t.theme.Dim.Render(" "+label+" "))
		}
	}

	return "  " + strings.Join(parts, "  ")
}

func (t *TrustScreen) StatusHints() []KeyHint {
	return []KeyHint{
		{Key: "\u2190\u2192", Desc: "choose"},
		{Key: "Enter", Desc: "confirm"},
		{Key: "Esc", Desc: "back"},
	}
}

// Cursor returns the current cursor position (for testing).
func (t *TrustScreen) Cursor() int {
	return t.cursor
}

// registryEntryNeedsConfirmation returns true if the entry requires a trust
// confirmation screen before proceeding.
func registryEntryNeedsConfirmation(entry catalog.Entry) bool {
	return entry.Source == catalog.SourceRegistry
}
