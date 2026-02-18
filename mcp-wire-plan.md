# mcp-wire — Implementation Plan

## Overview

**mcp-wire** is a CLI tool written in Go that lets users install and configure MCP (Model Context Protocol) servers across multiple AI coding CLI tools (Claude Code, Codex, Gemini CLI, OpenCode, etc.) from a single interface.

The architecture has two dimensions:

- **Services**: what to install (e.g., Sentry, Jira, Stripe). Defined as YAML files — no Go code needed to add one.
- **Targets**: where to install (e.g., Claude Code, Codex). Each target is a Go implementation that knows how to read/write that tool's config file.

The CLI combines the two: the user picks a service, the tool resolves credentials, and writes the config into one or more targets.

Interaction modes:

- **Guided mode (default)**: running `mcp-wire` with no subcommand starts an interactive menu/wizard for common workflows.
- **Explicit mode (advanced/CI)**: command-based syntax (for example `mcp-wire install <service> --target <slug> --no-prompt`) remains fully supported for scripting.

UX principle:

- Optimize for users who do not remember command syntax.
- At the end of guided flows, print the equivalent explicit command for reproducibility.

---

## Project Structure

```
mcp-wire/
├── cmd/
│   └── root.go                  # Cobra root command
│   └── install.go               # install command
│   └── uninstall.go             # uninstall command
│   └── list.go                  # list command (services, targets)
│   └── status.go                # status command
├── internal/
│   ├── service/
│   │   ├── service.go           # Service struct, YAML parsing
│   │   └── registry.go          # Discovers and lists available service definitions
│   ├── target/
│   │   ├── target.go            # Target interface definition
│   │   ├── registry.go          # Target discovery (which are installed on this machine)
│   │   ├── claude.go            # Claude Code target implementation
│   │   └── codex.go             # Codex target implementation
│   ├── credential/
│   │   ├── resolver.go          # Credential resolution chain
│   │   ├── env.go               # Environment variable source
│   │   └── file.go              # File-based credential store
│   └── config/
│       └── config.go            # mcp-wire's own config (paths, defaults)
├── services/                    # Community-contributed service definitions
│   ├── sentry.yaml
│   ├── jira.yaml
│   └── context7.yaml
├── main.go                      # Entrypoint, calls cmd/root.go
├── go.mod
├── go.sum
└── README.md
```

---

## Phase 1: Core Data Model ✅

### 1.1 — Service Definition (YAML schema) ✅

Create `internal/service/service.go` with the struct that maps to service YAML files.

```go
type Service struct {
    Name        string    `yaml:"name"`
    Description string    `yaml:"description"`
    Transport   string    `yaml:"transport"` // "sse" or "stdio"
    URL         string    `yaml:"url,omitempty"`         // for SSE transport
    Command     string    `yaml:"command,omitempty"`     // for stdio transport
    Args        []string  `yaml:"args,omitempty"`        // for stdio transport
    Env         []EnvVar  `yaml:"env,omitempty"`
}

type EnvVar struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description"`
    Required    bool   `yaml:"required"`
    SetupURL    string `yaml:"setup_url,omitempty"`
    SetupHint   string `yaml:"setup_hint,omitempty"`
}
```

A service YAML file looks like this:

```yaml
name: sentry
description: "Sentry error tracking MCP"
transport: sse
url: "https://mcp.sentry.dev/sse"
env:
  - name: SENTRY_AUTH_TOKEN
    description: "Sentry authentication token"
    required: true
    setup_url: "https://sentry.io/settings/account/api/auth-tokens/"
    setup_hint: "Create a token with project:read and event:read scopes"
```

```yaml
name: context7
description: "Context7 documentation lookup MCP"
transport: sse
url: "https://mcp.context7.com/mcp"
env: []
```

```yaml
name: filesystem
description: "Local filesystem access MCP"
transport: stdio
command: npx
args: ["-y", "@modelcontextprotocol/server-filesystem", "/home/user/projects"]
env: []
```

### 1.2 — Service Registry ✅

Create `internal/service/registry.go`.

This module discovers and loads all `.yaml` files from the `services/` directory. It should:

1. Accept a base path (default: `services/` relative to the binary, but also support `~/.config/mcp-wire/services/` for user-added definitions).
2. Read each `.yaml` file and unmarshal into a `Service` struct.
3. Validate required fields (name, transport, and either url or command depending on transport).
4. Return a map of `name → Service`.
5. If two files define the same `name`, the user-local one takes precedence.

Key function signatures:

```go
func LoadServices(paths ...string) (map[string]Service, error)
func ValidateService(s Service) error
```

### 1.3 — Target Interface ✅

Create `internal/target/target.go`.

```go
type Target interface {
    // Name returns the human-readable name (e.g., "Claude Code")
    Name() string

    // Slug returns the CLI-friendly identifier (e.g., "claude")
    Slug() string

    // IsInstalled checks if this CLI tool is present on the system
    IsInstalled() bool

    // Install writes the service config into this target's config file.
    // resolvedEnv contains env var names mapped to their resolved values.
    Install(svc service.Service, resolvedEnv map[string]string) error

    // Uninstall removes the service config from this target's config file.
    Uninstall(serviceName string) error

    // List returns the names of currently configured MCP services in this target.
    List() ([]string, error)
}
```

### 1.4 — Target Registry ✅

Create `internal/target/registry.go`.

A simple slice of all known targets. On startup, iterate and call `IsInstalled()` to determine which are available.

```go
func AllTargets() []Target
func InstalledTargets() []Target
func FindTarget(slug string) (Target, bool)
```

---

## Phase 2: Target Implementations ✅

Each target follows the same pattern: locate the config file, read it as JSON (preserving unknown keys), add/remove the MCP entry, write it back.

**Critical rule**: always preserve unknown keys. Use `map[string]any` or `json.RawMessage` when reading config files. Never deserialize into a strict struct that would drop fields the user set manually.

### 2.1 — Claude Code Target ✅

Create `internal/target/claude.go`.

Config file location: `~/.claude/settings.json` (global) or `.claude/settings.json` (project-local). Start with global only.

Config structure (relevant portion):

```json
{
  "mcpServers": {
    "sentry": {
      "type": "sse",
      "url": "https://mcp.sentry.dev/sse",
      "env": {
        "SENTRY_AUTH_TOKEN": "the-token-value"
      }
    }
  }
}
```

For stdio-based services:

```json
{
  "mcpServers": {
    "filesystem": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/home/user/projects"]
    }
  }
}
```

Implementation:

- `IsInstalled()`: check if `claude` binary is in PATH (use `exec.LookPath`).
- `Install()`: read `~/.claude/settings.json` → unmarshal to `map[string]any` → add entry under `mcpServers` → write back with `json.MarshalIndent`. Create the file/directory if it doesn't exist.
- `Uninstall()`: same flow, delete the key from `mcpServers`.
- `List()`: read `mcpServers` keys.

### 2.2 — Codex Target ✅

Create `internal/target/codex.go`.

Before implementing, research the current Codex CLI config file location and format. As of early 2025, Codex uses `~/.codex/config.json` or similar. The MCP server config structure needs to be verified.

**Action**: search for the Codex CLI documentation or config format before implementing. The structure is likely similar to Claude Code but may differ in key names or nesting.

Implementation follows the same pattern as Claude Code.

### 2.3 — Future Targets (not in v0.1)

Each of these would be a new file in `internal/target/`:

- `geminicli.go` — Gemini CLI
- `opencode.go` — OpenCode

Research their config formats when adding support. Each should take roughly 50-100 lines of Go following the established pattern.

---

## Phase 3: Credential Resolution ✅

### 3.1 — Credential Source Interface ✅

Create `internal/credential/resolver.go`.

```go
type Source interface {
    Name() string
    Get(envName string) (string, bool)
    Store(envName string, value string) error // may return ErrNotSupported
}

type Resolver struct {
    sources []Source
}

// Resolve tries each source in order. Returns the value and which source it came from.
func (r *Resolver) Resolve(envName string) (value string, source string, found bool)
```

Resolution order:

1. Environment variable (highest priority — user explicitly set it)
2. File store (`~/.config/mcp-wire/credentials`)
3. Not found → trigger interactive prompt

### 3.2 — Environment Source ✅

Create `internal/credential/env.go`.

Simple wrapper around `os.Getenv`. `Store()` returns `ErrNotSupported`.

### 3.3 — File Source ✅

Create `internal/credential/file.go`.

Reads/writes `~/.config/mcp-wire/credentials` in a simple `KEY=VALUE` format (one per line). This file should be created with `0600` permissions.

Format:

```
SENTRY_AUTH_TOKEN=snt_abc123...
JIRA_API_TOKEN=jira_xyz789...
```

On `Get()`: read the file, parse lines, return matching value.
On `Store()`: read the file, update or append the key, write back.

### 3.4 — Interactive Credential Flow ✅

This lives in the install command logic (not in the credential package). When a required env var is not found by the resolver:

1. Print the env var name, description, and why it is needed.
2. If `setup_url` is provided, print it and offer to open in browser (use `open` / `xdg-open` / `start` depending on OS).
3. If `setup_hint` is provided, print it.
4. Show progress when multiple credentials are required (example: `1/2`, `2/2`).
5. Prompt the user to paste the value (mask input for security).
6. Ask where to store: file store or skip (manage manually).
7. If file store chosen, call `source.Store()`.

Example output:

```
🔧 Configuring: Sentry error tracking MCP

  SENTRY_AUTH_TOKEN is required.
  → Create one here: https://sentry.io/settings/account/api/auth-tokens/
    Tip: Create a token with project:read and event:read scopes

  Open URL in browser? [Y/n]: y

  Paste your token: ****************************

  Save to mcp-wire credential store? [Y/n]: y
  ✓ Saved

  Installing to: Claude Code, Codex
  ✓ Claude Code — configured
  ✓ Codex — configured
```

---

## Phase 4: CLI Commands ✅

Use `github.com/spf13/cobra` for command structure.

### 4A — Guided Interactive UX (high priority) ✅

#### 4A.1 — Main interactive entry (`mcp-wire`) ✅

When the user runs `mcp-wire` with no subcommand, open an interactive menu:

1. Install a service
2. Uninstall a service
3. Show status
4. List services
5. List targets
6. Exit

#### 4A.2 — Install wizard flow ✅

1. **Service selection**
   - Show a searchable/filterable list (name + description).
   - Support quick search by substring.
2. **Target selection**
   - Show detected installed targets first.
   - Allow multi-select.
3. **Credential step (for each required env var)**
   - Show description.
   - Show setup URL and ask whether to open it.
   - Show setup hint/scopes.
   - Prompt with masked input.
   - Ask whether to save in credential store.
4. **Confirmation step**
   - Show selected service, targets, and credential source summary.
5. **Apply + results**
   - Print per-target success/failure.
   - Print equivalent explicit command.

#### 4A.3 — Uninstall wizard flow ✅

1. Service selection (searchable)
2. Target selection (multi-select)
3. Confirmation
4. Optional credential cleanup prompt

#### 4A.4 — Fallback behavior for explicit commands ✅

- `mcp-wire install` with no service argument should enter the service picker.
- `mcp-wire uninstall` with no service argument should enter the service picker.
- Explicit args/flags continue to work unchanged.

### 4.1 — `mcp-wire list services` ✅

List all available service definitions found in the services directory.

Output:

```
Available services:

  sentry       Sentry error tracking MCP
  jira         Jira project management MCP (Atlassian)
  context7     Context7 documentation lookup MCP
  filesystem   Local filesystem access MCP
```

### 4.2 — `mcp-wire list targets` ✅

List all known targets and whether they are detected on this system.

Output:

```
Targets:

  claude       Claude Code   ✓ installed
  codex        Codex CLI     ✓ installed
  geminicli    Gemini CLI    ✗ not found
  opencode     OpenCode      ✗ not found
```

### 4.3 — `mcp-wire install <service>` ✅

Flags:

- `--target <slug>` — install to a specific target only (can be repeated). Default: all installed targets.
- `--no-prompt` — fail if credentials are not already resolved (for CI/scripting).

Flow:

1. Load the service definition by name.
2. If `<service>` is omitted, enter interactive service selection.
3. For each required env var, run the credential resolver.
4. If not found and `--no-prompt` is not set, run the interactive prompt (Phase 3.4).
5. If not found and `--no-prompt` is set, exit with error.
6. For each target (filtered by `--target` or all installed), call `target.Install()`.
7. Print results.

### 4.4 — `mcp-wire uninstall <service>` ✅

Flags:

- `--target <slug>` — same as install.

Flow:

1. For each target, call `target.Uninstall(serviceName)`.
2. If `<service>` is omitted, enter interactive service selection.
3. Optionally ask if the user wants to remove stored credentials for this service.

### 4.5 — `mcp-wire status` ✅

Show a matrix of services × targets.

Output:

```
                Claude Code    Codex
  sentry        ✓              ✓
  jira          ✓              ✗
  context7      ✗              ✗
```

Implementation: for each installed target, call `target.List()` and cross-reference with known services.

---

## Phase 5: Initial Service Definitions ✅

Create YAML files in the `services/` directory for at least these services:

### sentry.yaml

```yaml
name: sentry
description: "Sentry error tracking MCP"
transport: sse
url: "https://mcp.sentry.dev/sse"
env:
  - name: SENTRY_AUTH_TOKEN
    description: "Sentry authentication token"
    required: true
    setup_url: "https://sentry.io/settings/account/api/auth-tokens/"
    setup_hint: "Create a token with project:read and event:read scopes"
```

### jira.yaml

```yaml
name: jira
description: "Jira project management MCP (Atlassian)"
transport: sse
url: "https://mcp.atlassian.com/v1/sse"
env:
  - name: JIRA_API_TOKEN
    description: "Atlassian API token"
    required: true
    setup_url: "https://id.atlassian.com/manage-profile/security/api-tokens"
    setup_hint: "Create an API token from your Atlassian account settings"
```

### context7.yaml

```yaml
name: context7
description: "Context7 documentation lookup MCP"
transport: sse
url: "https://mcp.context7.com/mcp"
env: []
```

### Research needed for additional services

Before writing more service definitions, verify the current MCP configuration format for each service. The URLs and env var names above are based on known configurations as of early 2025 and should be verified against current documentation.

---

## Phase 6: Build, Test, Release ✅

### 6.1 — Go Module Setup ✅

```
go mod init github.com/<your-username>/mcp-wire
go get github.com/spf13/cobra
go get gopkg.in/yaml.v3
```

No other external dependencies should be needed for v0.1.

### 6.2 — Testing Strategy ✅

- **Service loading**: test YAML parsing with valid and invalid files, test validation logic, test precedence when multiple paths are provided.
- **Target implementations**: test Install/Uninstall/List against temporary config files. Create a temp dir, write a sample config, run the operation, assert the result.
- **Credential resolver**: test the chain order (env takes precedence over file), test the file store read/write.
- **Integration test**: write a test that loads a service YAML, creates a temp config file, runs Install, reads back the config, and verifies the MCP entry is correct.
- **Guided interactive flow tests**: test menu navigation, service search/filter behavior, target multi-select, and credential prompt flow using mocked input/output streams.
- **Sandboxed CLI integration tests**: run end-to-end install/status/uninstall flows in temporary HOME/PATH to avoid touching user config.
- **Manual QA checklist**: test on a machine with existing MCP configs and a clean machine; verify no unrelated config keys are removed.

### 6.3 — Build ✅

Standard Go cross-compilation:

```bash
# Local
go build -o mcp-wire .

# Cross-compile
GOOS=darwin GOARCH=arm64 go build -o mcp-wire-darwin-arm64 .
GOOS=linux GOARCH=amd64 go build -o mcp-wire-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o mcp-wire-windows-amd64.exe .
```

### 6.4 — Release ✅

Use GoReleaser or GitHub Actions for automated releases. Provide binaries for macOS (arm64 + amd64), Linux (amd64), and Windows (amd64). Consider a Homebrew tap for macOS users.

---

## Implementation Order

Implement in this exact order to have something working as early as possible:

1. ✅ **Go module + main.go + Cobra skeleton** — just `mcp-wire --help` works.
2. ✅ **Service struct + YAML loader + validation** — can parse service files.
3. ✅ **`list services` command** — first working command, proves the YAML loading.
4. ✅ **Target interface + Claude Code implementation** — can read/write Claude Code config.
5. ✅ **`list targets` command** — detects installed tools.
6. ✅ **Credential resolver (env + file sources)** — can resolve and store tokens.
7. ✅ **`install` command with interactive prompt** — the core feature, end-to-end flow.
8. ✅ **`uninstall` command** — straightforward once install works.
9. ✅ **`status` command** — reads from all targets, displays matrix.
10. ✅ **Codex target implementation** — second target, validates the abstraction.
11. ✅ **Initial service YAML files** — ship with 3-5 verified services.
12. ✅ **README, contributing guide** — explain how to add services and targets.
13. ✅ **Guided main menu** — `mcp-wire` with no args opens interactive navigation.
14. ✅ **Install wizard UX** — searchable service picker, target selection, credential guidance.
15. ✅ **Uninstall wizard parity** — same guided UX model as install.
16. ✅ **Interactive UX tests + sandbox integration tests** — prevent regressions in guided flows.
17. ✅ **Equivalent-command summary in guided mode** — print scriptable command at the end of each workflow.

---

## Design Decisions and Constraints

### Config file safety

When reading a target's config file, always use `map[string]any` (not a strict struct). This ensures that any keys the user added manually are preserved when mcp-wire writes back to the file. This is the single most important implementation detail — getting it wrong means the tool destroys user config.

### Service YAML as the contribution surface

The primary way community members contribute is by adding YAML files to `services/`. This must require zero Go knowledge. The YAML schema should be well-documented in CONTRIBUTING.md with examples for both SSE and stdio transports.

### Credential storage is opt-in

The user is always asked before storing credentials to the file store. The `--no-prompt` flag makes credentials required from environment or file store (no interactive prompting), which is useful for CI/automation.

### No daemon, no server, no background process

mcp-wire is a pure CLI tool. It reads files, writes files, and exits. No running processes, no state between invocations beyond the credential file and the target config files.

### Embedded vs external service definitions

For v0.1, service YAML files are shipped alongside the binary (in the repo). The tool looks for them relative to the binary location and also in `~/.config/mcp-wire/services/`. This avoids any network dependency. A future version could fetch definitions from a remote registry.

---

## Phase 7: Official MCP Registry Integration (feature-gated, incremental)

This phase adds optional integration with the Official MCP Registry (`https://registry.modelcontextprotocol.io`) without changing the current safe defaults.

### 7.0 — Product guardrails (lock these first)

- **Curated stays default**: mcp-wire bundled/user YAML services remain the default discovery/install source.
- **Registry is opt-in**: registry support is disabled by default behind a one-time local setting until the feature is stable.
- **No hidden references when disabled**: if registry is disabled, do not show registry flags/options/help/menu entries anywhere.
- **Latest-only selection**: when a user selects a registry MCP, resolve details from the `latest` version only.
- **Runtime checks are warnings by default**: missing package managers should warn, not block installation.

### 7.1 — Feature gate and local settings ✅

Add mcp-wire local settings file (for example `~/.config/mcp-wire/config.json`) with:

```json
{
  "features": {
    "registry": false
  }
}
```

Suggested commands:

- `mcp-wire feature enable registry`
- `mcp-wire feature disable registry`
- `mcp-wire feature list` (optional but useful)

Acceptance criteria:

- Fresh install behavior is unchanged.
- With `registry=false`, all commands and guided flows behave exactly as today.

### 7.2 — Official registry API client (read-only) ✅

Create `internal/registry/` for typed API client logic.

Use these endpoints:

- List latest servers: `GET /v0.1/servers?version=latest&limit=...&cursor=...`
- Search latest servers: `GET /v0.1/servers?version=latest&search=...`
- Get selected server details (latest only): `GET /v0.1/servers/{serverName}/versions/latest`

Important details:

- URL-encode `serverName` path values (`/` must become `%2F`).
- Parse `application/problem+json` errors and return friendly messages.
- Do not expose historical version selection in this phase.

### 7.3 — Caching and indexing for responsive UX ✅

Add a local cache/index (for example under `~/.cache/mcp-wire/`) to avoid network-per-keystroke UI.

Requirements:

- Cold sync with pagination from `version=latest` list endpoint.
- Incremental sync using `updated_since`.
- Local in-memory filtering/search for interactive pickers.
- Graceful fallback to stale cache when network fails.
- Background sync at CLI startup when registry is enabled, with non-blocking UX and visible sync status in registry flows.
- Provide a cache reset command for local recovery (`mcp-wire cache clear`).

Acceptance criteria:

- Guided and non-guided search feel immediate.
- Registry API latency does not block every filter action.

### 7.4 — Unified catalog model with explicit source labels ✅

Introduce an internal catalog type that supports both curated and registry entries.

Each entry should carry source metadata:

- `source: curated` (maintained/tested by mcp-wire)
- `source: registry` (from Official MCP Registry, not vetted by mcp-wire)

Keep existing `service.Service` and YAML format intact for curated entries.

### 7.5 — Source selection in explicit CLI mode ✅

When registry is enabled, add source filtering:

- `--source curated|registry|all` (default: `curated`)

Apply this to:

- `mcp-wire list services`
- install flows that pick services interactively

When registry is disabled:

- Do not expose `--source`.
- Do not mention registry in help/usage text.

### 7.6 — Guided UI changes (only when enabled) ✅

Add a source step before service selection:

1. Curated services (recommended)
2. Registry services (community)
3. Both

If user selects "Both", show one merged searchable list with clear markers:

- `*` = curated by mcp-wire
- unmarked (or explicit label) = registry

Always print the legend near the list when markers are shown.

### 7.7 — Trust/safety messaging for registry entries ✅

Before confirming installation of a registry entry, show a short summary:

- Source (`registry`)
- Install type (`remote` or `package`)
- Transport (`streamable-http`, `sse`, `stdio`)
- Required secrets/credentials
- Repository URL (if available)

Interactive mode should require explicit confirmation for registry entries.

### 7.8 — Install strategy (split delivery)

#### 7.8.1 — Remote-first support ✅

First support registry entries that map cleanly to remote installs:

- `remotes` with `streamable-http` or `sse`
- URL/header variable prompting and substitution

#### 7.8.2 — Package support ✅

Then support package-backed installs:

- `npm`, `pypi`, `oci`, `nuget`, `mcpb`
- map registry metadata into target-compatible stdio/local config

For selected registry MCPs, always fetch/install using latest details only (`versions/latest`).

### 7.9 — Runtime/package-manager preflight (warning-only default)

For package-backed installs, preflight-check required runtime commands (for example `npx`, `uvx`, `docker`, `bun`, `dnx`, `pipx`, `python3`).

Behavior:

- Default: warn and continue.
- Optional strict mode (for CI): fail fast when runtime is missing (for example `--strict-runtime-check`).

### 7.10 — Status/uninstall parity for mixed sources

Ensure `status` and `uninstall` handle registry-installed services as first-class entries, not only curated YAML service names.

Requirements:

- Status should not hide registry-installed entries.
- Uninstall should work by installed service key regardless of source.

### 7.11 — Incremental implementation slices

Suggested order for shipping gradually:

1. ✅ Feature gate + settings file + no-registry-references enforcement.
2. ✅ Registry read client + latest-only detail lookup + basic tests.
3. ✅ Cache/index + local search.
4. ✅ `list services` source filter (`curated` default).
5. ✅ Guided UI source step and merged list markers.
6. ✅ Registry remote-install support.
7. Runtime preflight warnings.
8. ✅ Registry package-install support.
9. Status/uninstall parity for mixed sources.
10. Hardening, integration tests, docs.

---

## Phase 8: TUI Redesign — Bubble Tea Full-Screen UI

Replace the `survey/v2`-based interactive UI with a full-screen Bubble Tea TUI featuring breadcrumb navigation, fixed-height content area, live-filtered search, multi-select checkboxes, and progress indicators. The plain-text fallback (`guided.go`) and explicit CLI mode (`mcp-wire install sentry --target claude`) remain unchanged.

### Architecture

**Wizard state machine with composable screens.** A single root Bubble Tea model owns the layout shell (title bar, breadcrumb, status bar) and delegates 13-15 content lines to the active screen sub-model. Each screen implements a `Screen` interface. Navigation via message passing (`NavigateMsg`, `BackMsg`). A screen stack enables Esc-back.

New package structure:

```
internal/tui/
    app.go              -- Root model (state machine, layout shell)
    theme.go            -- Lip Gloss styles, color palette, layout constants
    screen.go           -- Screen interface, message types, ScreenID enum
    breadcrumb.go       -- Breadcrumb bar renderer
    statusbar.go        -- Bottom status bar renderer
    menu.go             -- Screen 1: Main Menu
    source.go           -- Screen 2: Source Selection
    service.go          -- Screen 3: Service Selection (with search)
    trust.go            -- Screen 4: Registry Trust Warning
    target.go           -- Screen 5: Target Multi-select
    scope.go            -- Screen 5b: Scope Selection
    review.go           -- Screen 6: Review
    apply.go            -- Screen 7: Apply (progress/result)
    credential.go       -- Credential prompting within TUI
```

Each file gets a corresponding `_test.go`.

Key types:

```go
type Screen interface {
    Init() tea.Cmd
    Update(msg tea.Msg) (Screen, tea.Cmd)
    View() string
    StatusHints() []KeyHint
}

type WizardState struct {
    Action  string              // "install" or "uninstall"
    Source  string              // "curated", "registry", "all"
    Service service.Service
    Entry   *catalog.Entry
    Targets []target.Target
    Scope   target.ConfigScope
    Results []TargetResult
}
```

New dependencies:

```
github.com/charmbracelet/bubbletea  v1.x
github.com/charmbracelet/lipgloss   v1.x
github.com/charmbracelet/bubbles    v0.x
```

After completion, `AlecAivazis/survey/v2` and its transitive deps are removed.

### 8.0 — Foundation (dependencies and layout shell) ✅

Add Bubble Tea, Lip Gloss, and Bubbles to `go.mod`. Create the package skeleton with the root model, theme, and a placeholder main-menu screen. Wire into `root.go` alongside the existing dispatch logic, guarded by a feature flag so the old survey path remains reachable during development.

Create:

- `internal/tui/app.go` — Root `WizardModel` (tea.Model). Owns terminal dimensions, renders layout frame (title row, breadcrumb, content, status bar). Delegates to active `Screen`.
- `internal/tui/screen.go` — `Screen` interface, `ScreenID` enum, navigation message types.
- `internal/tui/theme.go` — `Theme` struct with Lip Gloss styles matching mockup palette (cyan active, green completed, dim future, yellow warning, red error). Layout constant: `ContentHeight = 13`.
- `internal/tui/breadcrumb.go` — Renders from `WizardState`. Completed = green ✓ + value. Active = bold cyan. Future = dim.
- `internal/tui/statusbar.go` — Renders `[]KeyHint` on bottom row.
- `internal/tui/menu.go` — Placeholder main menu.

Modify:

- `go.mod` / `go.sum` — Add bubbletea, lipgloss, bubbles.
- `internal/cli/root.go` — Add TUI branch in `runGuidedMainMenu()`, guarded by feature flag.

Acceptance: `make test` passes. `bin/mcp-wire` shows the Bubble Tea menu.

### 8.1 — Main Menu (Screen 1) ✅

Full menu with ↑↓ navigation, Enter select, q quit. Non-TUI actions (Status, List services, List targets) captured into an output viewer screen that displays pre-rendered text with "press any key to return".

Acceptance: Menu navigable. Status/List output renders in TUI. Exit quits cleanly.

### 8.2 — Source Selection (Screen 2) ✅

Conditional on registry feature flag. Three options with inline descriptions: Curated (recommended), Registry, Both.

Modify `internal/tui/app.go` to skip source screen when registry feature is off, defaulting to "curated".

Acceptance: With registry enabled → source screen shown. Without → skipped directly to service selection.

### 8.3 — Service Selection (Screen 3) — hardest screen ✅

Live-filtered search over curated and/or registry catalog.

- Uses Bubbles `textinput.Model` for search bar.
- `allEntries`, `filtered`, `cursor`, `offset` for scroll state.
- Right-aligned count ("786 services" / "2 matches").
- Two-line entries: name + description.
- Scroll indicator ("▼ N more").
- `Init()` loads catalog via existing `loadCatalog()` from `catalog_helpers.go`.
- Registry sync: poll `registrySyncStatusLine()` via `tea.Tick` every 500ms (replaces the blocking `waitForRegistrySyncInSurvey`).

Acceptance: Search filters live. Scroll works. Enter selects.

### 8.4 — Registry Trust Warning (Screen 4) ✅

Shown after selecting a non-curated service. Displays service metadata (source, transport, URL, repo) and horizontal Yes/No choice.

- On "Yes": fetch latest via `refreshRegistryEntry()`, convert via `catalogEntryToService()`, proceed to targets.
- On "No": back to service list.

Polished view: ⚠ warning header, service name (cyan/bold) + description, remote URL in cyan, caution text, descriptive choice labels ("No, go back" / "Yes, proceed").

Acceptance: Registry service → trust shown. Curated service → skipped.

### 8.5 — Target Multi-select + Scope (Screens 5, 5b) ✅

Target screen:

- `[x]`/`[ ]` checkboxes. Space toggles, `a` all installed, `n` none.
- Not-installed targets dimmed and non-selectable.
- Enter confirms (≥1 required).

Scope screen:

- User/Project choice. Only shown if any selected target supports project scope.

Acceptance: Full flow from source → service → target → scope works.

### 8.6 — Review (Screen 6) ✅

Summary of all selections: Source, Service, Targets, Scope, Credentials mode. Shows equivalent CLI command (reuse `buildEquivalentInstallCommand` logic from `guided.go`). Horizontal Apply/Cancel choice.

Acceptance: Review shows correct summary. Apply proceeds to execution.

### 8.7 — Apply + Credentials (Screen 7) — second hardest ✅

Credential prompting in TUI:

- `CredentialScreen` steps through unresolved `EnvVar` entries sequentially.
- Uses `textinput.Model` with `EchoMode = EchoNone` for secrets.
- Shows setup URL/hint before each prompt.
- On completion, stores resolved values and proceeds to apply.

Apply screen:

- Per-target status rows: `◌` pending, `◌` configuring, `✓` done, `✗` failed.
- Runs each `target.Install()` via `tea.Cmd` (goroutine), receives progress messages.
- Success: green display + equivalent command + "Install another / Back to menu / Exit".
- Partial failure: warning + error details + "Retry failed / Back to menu / Exit".
- OAuth: runs `Authenticate()` in a `tea.Cmd`, shows "authenticating..." status.

Modify `internal/cli/install.go` — extract `installSingleTarget()` from `executeInstall()` loop so both CLI and TUI paths can call it.

Acceptance: Full end-to-end install with credential prompting and per-target progress.

### 8.8 — Uninstall Flow ✅

Reuses most install screens. `WizardState.Action = "uninstall"` controls rendering and business logic differences in Review and Apply screens. Add post-uninstall credential removal prompt (mirrors `maybeRemoveStoredCredentials` from `uninstall.go`).

Acceptance: Full uninstall flow works end-to-end in the TUI.

### 8.8.1 — Mockup alignment pass ✅

Visual polish across all TUI screens to match design mockups: source screen question header and inline descriptions, service search prompt and scroll indicator formatting, scope heading with target names and footer, review em-dash labels with Apply-first choice order and inline command display, apply service-name headers with per-state icons (✓/△/✗) and retry choice on failure.

### 8.9 — Cleanup and Survey Removal

Remove survey dependency and transition code.

Delete:

- `internal/cli/survey_ui.go`
- `internal/cli/survey_escape_input.go`
- `internal/cli/survey_escape_input_unix.go`
- `internal/cli/survey_escape_input_other.go`
- `internal/cli/survey_escape_input_test.go`

Modify:

- `internal/cli/root.go` — Remove survey branch and feature-flag guard. TTY → TUI, non-TTY → plain text.
- `go.mod` — Remove `AlecAivazis/survey/v2` and transitive deps.

Acceptance: `make test` passes. `survey` gone from `go.mod`. TUI, plain-text, and CLI flag modes all work.

### Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| Credential prompting in Bubble Tea | Use Bubbles `textinput` with `EchoNone` — hidden input native to the event loop. `term.ReadPassword` stays in `install_credentials.go` for plain-text fallback only. |
| Registry sync blocking | Replace `waitForRegistrySyncInSurvey()` with `tea.Tick` polling of `registrySyncStatusLine()`. Sync already runs in a background goroutine. |
| OAuth subprocess | Run `Authenticate()` inside a `tea.Cmd` (goroutine). Browser opens outside terminal control — no conflict. |
| Terminal resize | Handle `tea.WindowSizeMsg` in root model. Degrade gracefully if height is too small. |
| Coexistence during development | Feature flag toggles between survey and TUI. Default stays survey until step 8.9. |

---

## Future Roadmap (post v0.1)

These are features to consider after the core works. Not to be implemented now.

1. **Manifest file (`mcp-wire.yaml`)** — declare desired services in a file, run `mcp-wire apply` to sync all targets at once. Enables version-controlling your MCP setup.
2. **Target auto-detection improvements** — check config file existence in addition to binary presence, handle edge cases like Homebrew vs manual installs.
3. **OS keychain integration** — use `zalando/go-keyring` to store credentials in macOS Keychain / Linux Secret Service / Windows Credential Manager instead of a plaintext file.
4. **`mcp-wire credentials list`** — show stored credentials (masked).
5. **`mcp-wire credentials rotate <service>`** — re-open setup URL, re-prompt, update all targets.
6. **Additional registry providers** — support third-party/private registries that implement the MCP Registry API, with per-registry trust controls.
7. **Gemini CLI target** — add when config format is stable and documented.
8. **Target capability parity** — expand and test Claude/Codex/OpenCode parity for advanced MCP config shapes (for example richer header mapping, variable substitution, and transport-specific options).
9. **`mcp-wire diff`** — show what's configured in targets vs what a manifest says, dry-run mode.
10. **Profiles** — "work" vs "personal" service sets, each with their own credentials and target selection.
11. **Service health check** — `mcp-wire check <service>` verifies the MCP endpoint responds.
12. **Shell completions** — Cobra has built-in support for bash/zsh/fish completions.
