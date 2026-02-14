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

Create a new file in `internal/target/` implementing the `Target` interface (Name, Slug, IsInstalled, Install, Uninstall, List). Register it in `AllTargets()` in `registry.go`. Follow the `claude.go` pattern — read JSON as `map[string]any`, modify, write back.

## Implementation plan

See `mcp-wire-plan.md` for the full phased roadmap and design decisions.

## Release process

When asked to create a new release, follow this exact sequence:

1. Ensure tests pass without errors:

```bash
make test
make test-integration
```

2. Set the release version in `internal/app/app.go`:

- Use the provided version if one is given.
- Otherwise bump the patch version (example: `0.1.3` -> `0.1.4`).
- Update `var Version = "..."` in `internal/app/app.go`.

3. Update `CHANGELOG.md`:

- Add a short bullet-point summary of changes since the last release.
- Follow the existing changelog format.

4. Commit the release-prep changes.
5. Push the release-prep commit.
6. Create the tag using the version from `internal/app/app.go`:

```bash
git tag v<version>
```

7. Push the tag:

```bash
git push origin v<version>
```

Notes:

- The tag push triggers `.github/workflows/release.yml`.
- Do not manually create a GitHub release before the workflow runs.
