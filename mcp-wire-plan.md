# mcp-wire — Implementation Plan

## Overview

**mcp-wire** is a CLI tool written in Go that lets users install and configure MCP (Model Context Protocol) servers across multiple AI coding CLI tools (Claude Code, Codex, Gemini CLI, OpenCode, etc.) from a single interface.

The architecture has two dimensions:

- **Services**: what to install (e.g., Sentry, Jira, Stripe). Defined as YAML files — no Go code needed to add one.
- **Targets**: where to install (e.g., Claude Code, Codex). Each target is a Go implementation that knows how to read/write that tool's config file.

The CLI combines the two: the user picks a service, the tool resolves credentials, and writes the config into one or more targets.

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
│   │   ├── claudecode.go        # Claude Code target implementation
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

## Phase 1: Core Data Model

### 1.1 — Service Definition (YAML schema)

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

### 1.2 — Service Registry

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

### 1.3 — Target Interface

Create `internal/target/target.go`.

```go
type Target interface {
    // Name returns the human-readable name (e.g., "Claude Code")
    Name() string

    // Slug returns the CLI-friendly identifier (e.g., "claudecode")
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

### 1.4 — Target Registry

Create `internal/target/registry.go`.

A simple slice of all known targets. On startup, iterate and call `IsInstalled()` to determine which are available.

```go
func AllTargets() []Target
func InstalledTargets() []Target
func FindTarget(slug string) (Target, bool)
```

---

## Phase 2: Target Implementations

Each target follows the same pattern: locate the config file, read it as JSON (preserving unknown keys), add/remove the MCP entry, write it back.

**Critical rule**: always preserve unknown keys. Use `map[string]any` or `json.RawMessage` when reading config files. Never deserialize into a strict struct that would drop fields the user set manually.

### 2.1 — Claude Code Target

Create `internal/target/claudecode.go`.

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

### 2.2 — Codex Target

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

## Phase 3: Credential Resolution

### 3.1 — Credential Source Interface

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

### 3.2 — Environment Source

Create `internal/credential/env.go`.

Simple wrapper around `os.Getenv`. `Store()` returns `ErrNotSupported`.

### 3.3 — File Source

Create `internal/credential/file.go`.

Reads/writes `~/.config/mcp-wire/credentials` in a simple `KEY=VALUE` format (one per line). This file should be created with `0600` permissions.

Format:

```
SENTRY_AUTH_TOKEN=snt_abc123...
JIRA_API_TOKEN=jira_xyz789...
```

On `Get()`: read the file, parse lines, return matching value.
On `Store()`: read the file, update or append the key, write back.

### 3.4 — Interactive Credential Flow

This lives in the install command logic (not in the credential package). When a required env var is not found by the resolver:

1. Print the env var name and description.
2. If `setup_url` is provided, print it and offer to open in browser (use `open` / `xdg-open` / `start` depending on OS).
3. If `setup_hint` is provided, print it.
4. Prompt the user to paste the value (mask input for security).
5. Ask where to store: file store or skip (manage manually).
6. If file store chosen, call `source.Store()`.

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

## Phase 4: CLI Commands

Use `github.com/spf13/cobra` for command structure.

### 4.1 — `mcp-wire list services`

List all available service definitions found in the services directory.

Output:

```
Available services:

  sentry       Sentry error tracking MCP
  jira         Jira project management MCP (Atlassian)
  context7     Context7 documentation lookup MCP
  filesystem   Local filesystem access MCP
```

### 4.2 — `mcp-wire list targets`

List all known targets and whether they are detected on this system.

Output:

```
Targets:

  claudecode   Claude Code   ✓ installed
  codex        Codex CLI     ✓ installed
  geminicli    Gemini CLI    ✗ not found
  opencode     OpenCode      ✗ not found
```

### 4.3 — `mcp-wire install <service>`

Flags:

- `--target <slug>` — install to a specific target only (can be repeated). Default: all installed targets.
- `--no-prompt` — fail if credentials are not already resolved (for CI/scripting).

Flow:

1. Load the service definition by name.
2. For each required env var, run the credential resolver.
3. If not found and `--no-prompt` is not set, run the interactive prompt (Phase 3.4).
4. If not found and `--no-prompt` is set, exit with error.
5. For each target (filtered by `--target` or all installed), call `target.Install()`.
6. Print results.

### 4.4 — `mcp-wire uninstall <service>`

Flags:

- `--target <slug>` — same as install.

Flow:

1. For each target, call `target.Uninstall(serviceName)`.
2. Optionally ask if the user wants to remove stored credentials for this service.

### 4.5 — `mcp-wire status`

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

## Phase 5: Initial Service Definitions

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

## Phase 6: Build, Test, Release

### 6.1 — Go Module Setup

```
go mod init github.com/<your-username>/mcp-wire
go get github.com/spf13/cobra
go get gopkg.in/yaml.v3
```

No other external dependencies should be needed for v0.1.

### 6.2 — Testing Strategy

- **Service loading**: test YAML parsing with valid and invalid files, test validation logic, test precedence when multiple paths are provided.
- **Target implementations**: test Install/Uninstall/List against temporary config files. Create a temp dir, write a sample config, run the operation, assert the result.
- **Credential resolver**: test the chain order (env takes precedence over file), test the file store read/write.
- **Integration test**: write a test that loads a service YAML, creates a temp config file, runs Install, reads back the config, and verifies the MCP entry is correct.

### 6.3 — Build

Standard Go cross-compilation:

```bash
# Local
go build -o mcp-wire .

# Cross-compile
GOOS=darwin GOARCH=arm64 go build -o mcp-wire-darwin-arm64 .
GOOS=linux GOARCH=amd64 go build -o mcp-wire-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o mcp-wire-windows-amd64.exe .
```

### 6.4 — Release

Use GoReleaser or GitHub Actions for automated releases. Provide binaries for macOS (arm64 + amd64), Linux (amd64), and Windows (amd64). Consider a Homebrew tap for macOS users.

---

## Implementation Order

Implement in this exact order to have something working as early as possible:

1. **Go module + main.go + Cobra skeleton** — just `mcp-wire --help` works.
2. **Service struct + YAML loader + validation** — can parse service files.
3. **`list services` command** — first working command, proves the YAML loading.
4. **Target interface + Claude Code implementation** — can read/write Claude Code config.
5. **`list targets` command** — detects installed tools.
6. **Credential resolver (env + file sources)** — can resolve and store tokens.
7. **`install` command with interactive prompt** — the core feature, end-to-end flow.
8. **`uninstall` command** — straightforward once install works.
9. **`status` command** — reads from all targets, displays matrix.
10. **Codex target implementation** — second target, validates the abstraction.
11. **Initial service YAML files** — ship with 3-5 verified services.
12. **README, contributing guide** — explain how to add services and targets.

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

## Future Roadmap (post v0.1)

These are features to consider after the core works. Not to be implemented now.

1. **Manifest file (`mcp-wire.yaml`)** — declare desired services in a file, run `mcp-wire apply` to sync all targets at once. Enables version-controlling your MCP setup.
2. **Target auto-detection improvements** — check config file existence in addition to binary presence, handle edge cases like Homebrew vs manual installs.
3. **OS keychain integration** — use `zalando/go-keyring` to store credentials in macOS Keychain / Linux Secret Service / Windows Credential Manager instead of a plaintext file.
4. **`mcp-wire credentials list`** — show stored credentials (masked).
5. **`mcp-wire credentials rotate <service>`** — re-open setup URL, re-prompt, update all targets.
6. **Remote service registry** — `mcp-wire update` fetches latest service definitions from the GitHub repo without requiring a binary update.
7. **Gemini CLI target** — add when config format is stable and documented.
8. **OpenCode target** — add when config format is stable and documented.
9. **`mcp-wire diff`** — show what's configured in targets vs what a manifest says, dry-run mode.
10. **Profiles** — "work" vs "personal" service sets, each with their own credentials and target selection.
11. **Service health check** — `mcp-wire check <service>` verifies the MCP endpoint responds.
12. **Shell completions** — Cobra has built-in support for bash/zsh/fish completions.
