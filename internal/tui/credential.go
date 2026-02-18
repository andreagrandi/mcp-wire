package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/andreagrandi/mcp-wire/internal/service"
)

const (
	credSubStateInput = iota
	credSubStateSave
)

// credentialDoneMsg is sent when all credentials have been resolved.
type credentialDoneMsg struct {
	resolvedEnv map[string]string
}

// CredentialScreen steps through unresolved environment variables,
// prompting the user for each value with masked input.
type CredentialScreen struct {
	theme   Theme
	envVars []service.EnvVar // only unresolved required vars

	current  int // index of current env var being prompted
	resolved map[string]string

	textInput textinput.Model
	subState  int // credSubStateInput or credSubStateSave

	saveCursor       int // 0 = No, 1 = Yes
	lastEnteredValue string

	storeCredential func(name, value string) error
	openURL         func(string) error

	width int
}

// NewCredentialScreen creates a credential prompting screen for the given
// unresolved environment variables.
func NewCredentialScreen(
	theme Theme,
	envVars []service.EnvVar,
	preResolved map[string]string,
	storeCredential func(name, value string) error,
	openURL func(string) error,
) *CredentialScreen {
	resolved := make(map[string]string)
	for k, v := range preResolved {
		resolved[k] = v
	}

	ti := textinput.New()
	ti.Prompt = "  Enter value: "
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '*'
	ti.CharLimit = 500
	ti.Focus()

	return &CredentialScreen{
		theme:           theme,
		envVars:         envVars,
		resolved:        resolved,
		textInput:       ti,
		storeCredential: storeCredential,
		openURL:         openURL,
		saveCursor:      1, // default to Yes
	}
}

func (c *CredentialScreen) Init() tea.Cmd {
	return c.textInput.Focus()
}

func (c *CredentialScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.width = msg.Width
		return c, nil

	case tea.KeyMsg:
		switch c.subState {
		case credSubStateInput:
			return c.updateInput(msg)
		case credSubStateSave:
			return c.updateSave(msg)
		}
	}

	// Forward to textinput when in input state.
	if c.subState == credSubStateInput {
		var cmd tea.Cmd
		c.textInput, cmd = c.textInput.Update(msg)
		return c, cmd
	}

	return c, nil
}

func (c *CredentialScreen) updateInput(msg tea.KeyMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "enter":
		value := strings.TrimSpace(c.textInput.Value())
		if value == "" {
			return c, nil
		}

		c.lastEnteredValue = value

		// If we can store credentials, ask whether to save.
		if c.storeCredential != nil {
			c.subState = credSubStateSave
			c.saveCursor = 1 // default to Yes
			return c, nil
		}

		// No store available — accept and advance.
		return c.acceptAndAdvance()

	case "esc":
		return c, func() tea.Msg { return BackMsg{} }

	case "ctrl+o":
		ev := c.envVars[c.current]
		url := strings.TrimSpace(ev.SetupURL)
		if url != "" && c.openURL != nil {
			_ = c.openURL(url)
		}
		return c, nil
	}

	// Forward all other keys to textinput.
	var cmd tea.Cmd
	c.textInput, cmd = c.textInput.Update(msg)
	return c, cmd
}

func (c *CredentialScreen) updateSave(msg tea.KeyMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "left", "h":
		if c.saveCursor > 0 {
			c.saveCursor--
		}
	case "right", "l":
		if c.saveCursor < 1 {
			c.saveCursor++
		}
	case "enter":
		if c.saveCursor == 1 && c.storeCredential != nil {
			ev := c.envVars[c.current]
			_ = c.storeCredential(strings.TrimSpace(ev.Name), c.lastEnteredValue)
		}
		return c.acceptAndAdvance()
	case "esc":
		// Skip saving and advance.
		return c.acceptAndAdvance()
	}

	return c, nil
}

// acceptAndAdvance stores the entered value and moves to the next credential
// or finishes.
func (c *CredentialScreen) acceptAndAdvance() (Screen, tea.Cmd) {
	ev := c.envVars[c.current]
	c.resolved[strings.TrimSpace(ev.Name)] = c.lastEnteredValue

	c.current++
	if c.current >= len(c.envVars) {
		// All credentials resolved.
		resolved := make(map[string]string)
		for k, v := range c.resolved {
			resolved[k] = v
		}
		return c, func() tea.Msg {
			return credentialDoneMsg{resolvedEnv: resolved}
		}
	}

	// Reset for next credential.
	c.subState = credSubStateInput
	c.lastEnteredValue = ""
	c.textInput.Reset()
	c.textInput.Focus()
	c.saveCursor = 1

	return c, c.textInput.Focus()
}

func (c *CredentialScreen) View() string {
	var b strings.Builder

	ev := c.envVars[c.current]
	envName := strings.TrimSpace(ev.Name)
	description := strings.TrimSpace(ev.Description)
	setupURL := strings.TrimSpace(ev.SetupURL)
	setupHint := strings.TrimSpace(ev.SetupHint)

	b.WriteString("\n")

	// Progress indicator.
	progress := fmt.Sprintf("[%d/%d]", c.current+1, len(c.envVars))
	if description != "" {
		b.WriteString(fmt.Sprintf("  %s %s required (%s).\n", progress, envName, description))
	} else {
		b.WriteString(fmt.Sprintf("  %s %s required.\n", progress, envName))
	}

	if setupURL != "" {
		b.WriteString(c.theme.Dim.Render("      URL: "+setupURL) + "\n")
	}
	if setupHint != "" {
		b.WriteString(c.theme.Dim.Render("      Hint: "+setupHint) + "\n")
	}

	b.WriteString("\n")

	switch c.subState {
	case credSubStateInput:
		b.WriteString(c.textInput.View())
		b.WriteString("\n")

	case credSubStateSave:
		b.WriteString(c.theme.Completed.Render("  Value entered."))
		b.WriteString("\n\n")
		b.WriteString("  Save to credential store?\n\n")
		b.WriteString(c.renderSaveChoices())
		b.WriteString("\n")
	}

	return b.String()
}

func (c *CredentialScreen) renderSaveChoices() string {
	labels := []string{"No", "Yes"}
	var parts []string

	for i, label := range labels {
		if i == c.saveCursor {
			if c.width > 0 {
				parts = append(parts, c.theme.Highlight.Render(" "+label+" "))
			} else {
				parts = append(parts, c.theme.Cursor.Render("["+label+"]"))
			}
		} else {
			parts = append(parts, c.theme.Dim.Render(" "+label+" "))
		}
	}

	return "  " + strings.Join(parts, "  ")
}

func (c *CredentialScreen) StatusHints() []KeyHint {
	if c.subState == credSubStateSave {
		return []KeyHint{
			{Key: "\u2190\u2192", Desc: "choose"},
			{Key: "Enter", Desc: "confirm"},
			{Key: "Esc", Desc: "skip"},
		}
	}

	hints := []KeyHint{
		{Key: "Enter", Desc: "submit"},
		{Key: "Esc", Desc: "back"},
	}

	ev := c.envVars[c.current]
	if strings.TrimSpace(ev.SetupURL) != "" {
		hints = append(hints, KeyHint{Key: "Ctrl+O", Desc: "open URL"})
	}

	return hints
}

// Current returns the index of the credential being prompted (for testing).
func (c *CredentialScreen) Current() int {
	return c.current
}

// SubState returns the current sub-state (for testing).
func (c *CredentialScreen) SubState() int {
	return c.subState
}

// SaveCursor returns the save prompt cursor position (for testing).
func (c *CredentialScreen) SaveCursor() int {
	return c.saveCursor
}

// Resolved returns a copy of the currently resolved credentials (for testing).
func (c *CredentialScreen) Resolved() map[string]string {
	cp := make(map[string]string)
	for k, v := range c.resolved {
		cp[k] = v
	}
	return cp
}
