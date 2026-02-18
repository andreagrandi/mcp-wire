package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
)

// reviewConfirmMsg is sent when the user confirms or cancels from the review screen.
type reviewConfirmMsg struct {
	confirmed bool
}

// ReviewScreen shows a summary of all wizard selections and offers
// Apply/Cancel before proceeding.
type ReviewScreen struct {
	theme           Theme
	state           WizardState
	registryEnabled bool
	cursor          int // 0 = Cancel, 1 = Apply
	width           int
}

// NewReviewScreen creates a review screen summarising the wizard state.
func NewReviewScreen(theme Theme, state WizardState, registryEnabled bool) *ReviewScreen {
	return &ReviewScreen{
		theme:           theme,
		state:           state,
		registryEnabled: registryEnabled,
		cursor:          0, // default to Apply (first choice)
	}
}

func (r *ReviewScreen) Init() tea.Cmd { return nil }

func (r *ReviewScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		r.width = msg.Width
		return r, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			if r.cursor > 0 {
				r.cursor--
			}
		case "right", "l":
			if r.cursor < 1 {
				r.cursor++
			}
		case "enter":
			confirmed := r.cursor == 0
			return r, func() tea.Msg {
				return reviewConfirmMsg{confirmed: confirmed}
			}
		case "esc":
			return r, func() tea.Msg { return BackMsg{} }
		}
	}

	return r, nil
}

func (r *ReviewScreen) View() string {
	var b strings.Builder

	b.WriteString("\n")

	// Summary lines.
	if r.registryEnabled {
		b.WriteString(r.summaryLine("Source", sourceValueLabel(r.state.Source)))
	}

	b.WriteString(r.summaryLine("Service", r.serviceLabel()))
	b.WriteString(r.summaryLine("Targets", r.targetNames()))

	if anyTargetSupportsProjectScope(r.state.Targets) {
		b.WriteString(r.summaryLine("Scope", scopeLabel(r.state.Scope)))
	}

	if r.state.Action != "uninstall" {
		b.WriteString(r.summaryLine("Credentials", "prompt as needed"))
	}

	// Equivalent command.
	b.WriteString("\n")
	b.WriteString(r.summaryLine("Command", r.equivalentCommand()))

	// Confirmation choices.
	b.WriteString("\n")
	b.WriteString(r.renderChoices())

	return b.String()
}

func (r *ReviewScreen) summaryLine(label, value string) string {
	return r.theme.Dim.Render("  "+label+":") + "  " + value + "\n"
}

func (r *ReviewScreen) serviceLabel() string {
	desc := r.state.Entry.Description()
	if desc != "" {
		return r.state.Entry.Name + " \u2014 " + desc
	}
	return r.state.Entry.Name
}

func (r *ReviewScreen) targetNames() string {
	names := make([]string, 0, len(r.state.Targets))
	for _, t := range r.state.Targets {
		names = append(names, t.Name())
	}
	return strings.Join(names, ", ")
}

func (r *ReviewScreen) equivalentCommand() string {
	cmd := "mcp-wire " + r.state.Action + " " + r.state.Entry.Name
	for _, t := range r.state.Targets {
		cmd += " --target " + t.Slug()
	}
	if r.state.Scope == targetpkg.ConfigScopeProject {
		cmd += " --scope project"
	}
	return cmd
}

func (r *ReviewScreen) renderChoices() string {
	labels := []string{"Apply", "Cancel"}
	var parts []string

	for i, label := range labels {
		if i == r.cursor {
			if r.width > 0 {
				parts = append(parts, r.theme.Highlight.Render(" "+label+" "))
			} else {
				parts = append(parts, r.theme.Cursor.Render("["+label+"]"))
			}
		} else {
			parts = append(parts, r.theme.Dim.Render(" "+label+" "))
		}
	}

	return "  " + strings.Join(parts, "  ")
}

func (r *ReviewScreen) StatusHints() []KeyHint {
	return []KeyHint{
		{Key: "\u2190\u2192", Desc: "choose"},
		{Key: "Enter", Desc: "confirm"},
		{Key: "Esc", Desc: "back"},
	}
}

// Cursor returns the current cursor position (for testing).
func (r *ReviewScreen) Cursor() int {
	return r.cursor
}

// scopeLabel returns a display string for a config scope.
func scopeLabel(scope targetpkg.ConfigScope) string {
	switch scope {
	case targetpkg.ConfigScopeProject:
		return "Project (current directory only)"
	case targetpkg.ConfigScopeUser:
		return "User (for targets that support it)"
	default:
		return string(scope)
	}
}
