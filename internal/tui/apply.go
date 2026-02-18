package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/andreagrandi/mcp-wire/internal/service"
	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
)

const (
	applySubStateRunning = iota
	applySubStateCredCleanup
	applySubStateDone
)

// applyResultMsg carries the result of a single target operation.
type applyResultMsg struct {
	index    int
	err      error
	authHint string
}

// applyPostActionMsg is sent when the user picks a post-completion action.
type applyPostActionMsg struct {
	action string // "another", "menu", "exit"
}

// targetResult tracks the status of an operation on a single target.
type targetResult struct {
	name     string
	slug     string
	status   string // "pending", "running", "done", "failed"
	err      error
	authHint string
}

// ApplyCallbacks provides functions the apply screen needs to perform operations.
type ApplyCallbacks struct {
	InstallTarget           func(svc service.Service, env map[string]string, t targetpkg.Target, scope targetpkg.ConfigScope) error
	UninstallTarget         func(name string, t targetpkg.Target, scope targetpkg.ConfigScope) error
	ServiceUsesOAuth        func(svc service.Service) bool
	OAuthManualHint         func(t targetpkg.Target) string
	RemoveStoredCredentials func(envNames []string) (int, error)
}

// ApplyScreen shows per-target progress during install/uninstall and
// presents post-completion actions.
type ApplyScreen struct {
	theme       Theme
	state       WizardState
	svc         service.Service
	resolvedEnv map[string]string
	callbacks   ApplyCallbacks

	results  []targetResult
	subState int // applySubStateRunning, applySubStateCredCleanup, or applySubStateDone
	cursor   int // cursor for post-completion choices
	width    int

	hasFailures       bool
	credCleanupCursor int    // 0 = No, 1 = Yes
	credCleanupMsg    string // result message after credential cleanup
}

// NewApplyScreen creates a new apply screen for the given wizard state.
func NewApplyScreen(
	theme Theme,
	state WizardState,
	svc service.Service,
	resolvedEnv map[string]string,
	callbacks ApplyCallbacks,
) *ApplyScreen {
	results := make([]targetResult, len(state.Targets))
	for i, t := range state.Targets {
		results[i] = targetResult{
			name:   t.Name(),
			slug:   t.Slug(),
			status: "pending",
		}
	}

	return &ApplyScreen{
		theme:       theme,
		state:       state,
		svc:         svc,
		resolvedEnv: resolvedEnv,
		callbacks:   callbacks,
		results:     results,
	}
}

func (a *ApplyScreen) Init() tea.Cmd {
	if len(a.results) == 0 {
		a.subState = applySubStateDone
		return nil
	}

	a.results[0].status = "running"
	return a.dispatchTarget(0)
}

func (a *ApplyScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		return a, nil

	case applyResultMsg:
		return a.handleResult(msg)

	case tea.KeyMsg:
		switch a.subState {
		case applySubStateDone:
			return a.updateDone(msg)
		case applySubStateCredCleanup:
			return a.updateCredCleanup(msg)
		}
	}

	return a, nil
}

func (a *ApplyScreen) handleResult(msg applyResultMsg) (Screen, tea.Cmd) {
	if msg.index < 0 || msg.index >= len(a.results) {
		return a, nil
	}

	if msg.err != nil {
		a.results[msg.index].status = "failed"
		a.results[msg.index].err = msg.err
		a.hasFailures = true
	} else {
		a.results[msg.index].status = "done"
		a.results[msg.index].authHint = msg.authHint
	}

	// Start next target if any.
	next := msg.index + 1
	if next < len(a.results) {
		a.results[next].status = "running"
		return a, a.dispatchTarget(next)
	}

	// All done.
	if a.shouldShowCredCleanup() {
		a.subState = applySubStateCredCleanup
		a.credCleanupCursor = 0 // default to No, matching CLI's [y/N]
		return a, nil
	}

	a.subState = applySubStateDone
	return a, nil
}

func (a *ApplyScreen) updateDone(msg tea.KeyMsg) (Screen, tea.Cmd) {
	choices := a.postActionChoices()

	switch msg.String() {
	case "left", "h":
		if a.cursor > 0 {
			a.cursor--
		}
	case "right", "l":
		if a.cursor < len(choices)-1 {
			a.cursor++
		}
	case "enter":
		action := choices[a.cursor].action
		return a, func() tea.Msg {
			return applyPostActionMsg{action: action}
		}
	case "esc":
		return a, func() tea.Msg {
			return applyPostActionMsg{action: "menu"}
		}
	}

	return a, nil
}

func (a *ApplyScreen) updateCredCleanup(msg tea.KeyMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "left", "h":
		if a.credCleanupCursor > 0 {
			a.credCleanupCursor--
		}
	case "right", "l":
		if a.credCleanupCursor < 1 {
			a.credCleanupCursor++
		}
	case "enter":
		if a.credCleanupCursor == 1 && a.callbacks.RemoveStoredCredentials != nil {
			removed, err := a.callbacks.RemoveStoredCredentials(a.envVarNames())
			if err != nil {
				a.credCleanupMsg = "Error removing credentials: " + err.Error()
			} else if removed == 0 {
				a.credCleanupMsg = "No stored credentials found."
			} else {
				a.credCleanupMsg = "Stored credentials removed."
			}
		}
		a.subState = applySubStateDone
	case "esc":
		a.subState = applySubStateDone
	}

	return a, nil
}

// shouldShowCredCleanup returns true when the post-uninstall credential
// removal prompt should be shown.
func (a *ApplyScreen) shouldShowCredCleanup() bool {
	if a.state.Action != "uninstall" {
		return false
	}
	if a.hasFailures {
		return false
	}
	if a.callbacks.RemoveStoredCredentials == nil {
		return false
	}
	for _, ev := range a.svc.Env {
		if strings.TrimSpace(ev.Name) != "" {
			return true
		}
	}
	return false
}

// envVarNames returns deduplicated env var names from the service.
func (a *ApplyScreen) envVarNames() []string {
	seen := make(map[string]struct{})
	var names []string
	for _, ev := range a.svc.Env {
		name := strings.TrimSpace(ev.Name)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names
}

func (a *ApplyScreen) dispatchTarget(idx int) tea.Cmd {
	svc := a.svc
	resolvedEnv := a.resolvedEnv
	target := a.state.Targets[idx]
	scope := a.state.Scope
	action := a.state.Action
	callbacks := a.callbacks

	return func() tea.Msg {
		var err error

		if action == "uninstall" {
			if callbacks.UninstallTarget != nil {
				err = callbacks.UninstallTarget(svc.Name, target, scope)
			}
		} else {
			if callbacks.InstallTarget != nil {
				err = callbacks.InstallTarget(svc, resolvedEnv, target, scope)
			}
		}

		var authHint string
		if err == nil && action != "uninstall" {
			if callbacks.ServiceUsesOAuth != nil && callbacks.ServiceUsesOAuth(svc) {
				if callbacks.OAuthManualHint != nil {
					authHint = callbacks.OAuthManualHint(target)
				}
			}
		}

		return applyResultMsg{
			index:    idx,
			err:      err,
			authHint: authHint,
		}
	}
}

func (a *ApplyScreen) View() string {
	var b strings.Builder

	b.WriteString("\n")

	if a.subState == applySubStateRunning {
		if a.state.Action == "uninstall" {
			b.WriteString("  Removing from targets...\n")
		} else {
			b.WriteString("  Installing to targets...\n")
		}
	} else {
		b.WriteString(a.doneHeader())
	}

	b.WriteString("\n")

	// Per-target status rows.
	for _, r := range a.results {
		b.WriteString(a.renderTargetRow(r))
		b.WriteString("\n")
	}

	if a.subState == applySubStateCredCleanup {
		b.WriteString("\n")
		b.WriteString("  Remove stored credentials?\n\n")
		b.WriteString(a.renderCredCleanupChoices())
		b.WriteString("\n")
	}

	if a.subState == applySubStateDone {
		// Auth hints.
		for _, r := range a.results {
			if r.authHint != "" && r.status == "done" {
				b.WriteString(a.theme.Warning.Render(fmt.Sprintf("  [!] %s: %s", r.name, r.authHint)))
				b.WriteString("\n")
			}
		}

		// Credential cleanup result.
		if a.credCleanupMsg != "" {
			b.WriteString("\n")
			b.WriteString("  " + a.credCleanupMsg)
			b.WriteString("\n")
		}

		// Equivalent command.
		b.WriteString("\n")
		b.WriteString(a.theme.Dim.Render("  Equivalent command:"))
		b.WriteString("\n")
		b.WriteString("    " + a.equivalentCommand())
		b.WriteString("\n\n")

		// Post-action choices.
		b.WriteString(a.renderPostActionChoices())
	}

	return b.String()
}

func (a *ApplyScreen) doneHeader() string {
	if a.hasFailures {
		allFailed := true
		for _, r := range a.results {
			if r.status == "done" {
				allFailed = false
				break
			}
		}
		if allFailed {
			return a.theme.Error.Render("  Operation failed.") + "\n"
		}
		return a.theme.Warning.Render("  Completed with errors.") + "\n"
	}

	if a.state.Action == "uninstall" {
		return a.theme.Completed.Render("  Uninstall complete!") + "\n"
	}
	return a.theme.Completed.Render("  Install complete!") + "\n"
}

func (a *ApplyScreen) renderTargetRow(r targetResult) string {
	var icon string
	switch r.status {
	case "pending":
		icon = a.theme.Dim.Render("  \u25cc")
	case "running":
		icon = a.theme.Active.Render("  \u25cc")
	case "done":
		icon = a.theme.Completed.Render("  \u2713")
	case "failed":
		icon = a.theme.Error.Render("  \u2717")
	}

	statusLabel := r.status
	if r.status == "running" {
		if a.state.Action == "uninstall" {
			statusLabel = "removing..."
		} else {
			statusLabel = "configuring..."
		}
	} else if r.status == "done" {
		if a.state.Action == "uninstall" {
			statusLabel = "removed"
		} else {
			statusLabel = "configured"
		}
	} else if r.status == "failed" && r.err != nil {
		statusLabel = fmt.Sprintf("failed (%s)", r.err.Error())
	}

	return fmt.Sprintf("%s %-16s %s", icon, r.name, statusLabel)
}

func (a *ApplyScreen) equivalentCommand() string {
	cmd := "mcp-wire " + a.state.Action + " " + a.state.Entry.Name
	for _, t := range a.state.Targets {
		cmd += " --target " + t.Slug()
	}
	if a.state.Scope == targetpkg.ConfigScopeProject {
		cmd += " --scope project"
	}
	return cmd
}

type postActionChoice struct {
	label  string
	action string
}

func (a *ApplyScreen) postActionChoices() []postActionChoice {
	var actionLabel string
	if a.state.Action == "uninstall" {
		actionLabel = "Uninstall another"
	} else {
		actionLabel = "Install another"
	}

	return []postActionChoice{
		{label: actionLabel, action: "another"},
		{label: "Back to menu", action: "menu"},
		{label: "Exit", action: "exit"},
	}
}

func (a *ApplyScreen) renderPostActionChoices() string {
	choices := a.postActionChoices()
	var parts []string

	for i, c := range choices {
		if i == a.cursor {
			if a.width > 0 {
				parts = append(parts, a.theme.Highlight.Render(" "+c.label+" "))
			} else {
				parts = append(parts, a.theme.Cursor.Render("["+c.label+"]"))
			}
		} else {
			parts = append(parts, a.theme.Dim.Render(" "+c.label+" "))
		}
	}

	return "  " + strings.Join(parts, "  ")
}

func (a *ApplyScreen) renderCredCleanupChoices() string {
	labels := []string{"No", "Yes"}
	var parts []string

	for i, label := range labels {
		if i == a.credCleanupCursor {
			if a.width > 0 {
				parts = append(parts, a.theme.Highlight.Render(" "+label+" "))
			} else {
				parts = append(parts, a.theme.Cursor.Render("["+label+"]"))
			}
		} else {
			parts = append(parts, a.theme.Dim.Render(" "+label+" "))
		}
	}

	return "  " + strings.Join(parts, "  ")
}

func (a *ApplyScreen) StatusHints() []KeyHint {
	if a.subState == applySubStateCredCleanup {
		return []KeyHint{
			{Key: "\u2190\u2192", Desc: "choose"},
			{Key: "Enter", Desc: "confirm"},
			{Key: "Esc", Desc: "skip"},
		}
	}

	if a.subState == applySubStateDone {
		return []KeyHint{
			{Key: "\u2190\u2192", Desc: "choose"},
			{Key: "Enter", Desc: "confirm"},
		}
	}

	return []KeyHint{}
}

// Results returns the target results (for testing).
func (a *ApplyScreen) Results() []targetResult {
	cp := make([]targetResult, len(a.results))
	copy(cp, a.results)
	return cp
}

// PostCursor returns the post-action cursor position (for testing).
func (a *ApplyScreen) PostCursor() int {
	return a.cursor
}

// SubState returns the current sub-state (for testing).
func (a *ApplyScreen) ApplySubState() int {
	return a.subState
}

// CredCleanupCursor returns the credential cleanup cursor position (for testing).
func (a *ApplyScreen) CredCleanupCursor() int {
	return a.credCleanupCursor
}

// CredCleanupMsg returns the credential cleanup result message (for testing).
func (a *ApplyScreen) CredCleanupMsg() string {
	return a.credCleanupMsg
}
