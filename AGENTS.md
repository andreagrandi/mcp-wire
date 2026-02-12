This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
make build              # Build binary to bin/mcp-wire
make test               # Run all tests
make test-verbose       # Run tests with verbose output
go test ./internal/target/  # Run tests for a single package
go test -run TestClaudeCodeInstallSSE ./internal/target/  # Run a single test
make vet                # Static analysis
make fmt                # Format code
make build-all          # Cross-compile (linux, darwin amd64/arm64, windows)
```

## Architecture

mcp-wire is a CLI tool that installs MCP (Model Context Protocol) servers across multiple AI coding tools from a single interface. Two independent dimensions:

- **Services** (`internal/service/`): *what* to install. Defined as YAML files in `services/` or `~/.config/mcp-wire/services/`. No Go code needed to add a service. User-local definitions override bundled ones by name.
- **Targets** (`internal/target/`): *where* to install. Each target implements the `Target` interface and knows how to read/write a specific tool's config file. Currently: Claude Code. Planned: Codex, Gemini CLI, OpenCode.

The CLI (`internal/cli/`) combines the two: user picks a service, tool resolves credentials, writes config into target(s).

### Config file safety

When reading a target's config file, always use `map[string]any` — never a strict struct. This preserves any keys the user set manually. This is the most important implementation detail; getting it wrong destroys user config.

### Key packages

- `internal/app` — version constants (overridable via ldflags)
- `internal/cli` — Cobra commands
- `internal/service` — `Service`/`EnvVar` structs, YAML loading, validation
- `internal/target` — `Target` interface, registry, per-tool implementations
- `cmd/mcp-wire` — entrypoint

## Adding a new service

Create a YAML file in `services/`. No Go changes required. Two transport types:

```yaml
# SSE transport
name: example
description: "Example MCP"
transport: sse
url: "https://mcp.example.com/sse"
env:
  - name: EXAMPLE_TOKEN
    description: "API token"
    required: true
    setup_url: "https://example.com/tokens"
    setup_hint: "Create a read-only token"
```

```yaml
# stdio transport
name: example
description: "Example MCP"
transport: stdio
command: npx
args: ["-y", "@example/mcp-server"]
env: []
```

## Adding a new target

Create a new file in `internal/target/` implementing the `Target` interface (Name, Slug, IsInstalled, Install, Uninstall, List). Register it in `AllTargets()` in `registry.go`. Follow the `claudecode.go` pattern — read JSON as `map[string]any`, modify, write back.

## Implementation plan

See `mcp-wire-plan.md` for the full phased roadmap and design decisions.
