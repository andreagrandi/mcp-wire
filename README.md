# mcp-wire

<img src="mcp-wire-logo.png" alt="mcp-wire logo" width="50%" />

mcp-wire is a Go CLI that installs and configures MCP (Model Context Protocol) servers across multiple AI coding tools from one interface.

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap andreagrandi/tap
brew install mcp-wire
```

### From source

```bash
git clone https://github.com/andreagrandi/mcp-wire
cd mcp-wire
make build
./bin/mcp-wire --help
```

## What is implemented so far

- Cobra CLI foundation with version/help support
- Core service data model in `internal/service/service.go`
- Service registry loader with YAML parsing, validation, and path precedence in `internal/service/registry.go`
- Target abstraction in `internal/target/target.go`
- Target registry with discovery helpers in `internal/target/registry.go`
- Target implementation for Claude Code in `internal/target/claudecode.go`
- Target implementation for Codex CLI in `internal/target/codex.go`
- Target implementation for OpenCode in `internal/target/opencode.go`
- Integration test suite in `internal/integration/cli_integration_test.go` (build tag: `integration`)
- Credential resolution foundation in `internal/credential/`:
  - resolver chain (`resolver.go`)
  - environment source (`env.go`)
  - file source (`file.go`)
- Interactive credential prompt helpers in `internal/cli/install_credentials.go`:
  - required-variable prompting
  - setup URL + hint display
  - optional browser open
  - masked secret input on terminal
  - optional persistence to file credentials store
- CLI commands:
  - `mcp-wire list services`
  - `mcp-wire list targets`
  - `mcp-wire install <service> [--target ...] [--no-prompt]`
  - `mcp-wire uninstall <service> [--target ...]`
  - `mcp-wire status`
- Initial bundled service definitions in `services/`:
  - `context7.yaml`
  - `jira.yaml`
  - `sentry.yaml`
- CI workflow via GitHub Actions
- Release automation via GoReleaser (`.goreleaser.yaml`) and tag-based GitHub workflow (`.github/workflows/release.yml`)
- Changelog initialized in `CHANGELOG.md`
- Unit tests for service loading, targets, and credentials

## Current behavior

- Service definitions load from executable-relative `services/`, working-directory `services/`, and `~/.config/mcp-wire/services/`
- Validation supports `sse` and `stdio` transports
- Duplicate service names are resolved by load order (later paths override earlier paths)
- Claude Code, Codex, and OpenCode target implementations can detect installation, install/update entries, uninstall entries, and list configured services
- Target config writes preserve unknown user-defined keys by using map-based parsing
- Credential resolution supports environment variables first, then file-based credentials at `~/.config/mcp-wire/credentials` (stored with `0600` permissions)
- Interactive credential prompts can collect missing required values with optional setup URL opening and optional storage in the credential file store
- `status` prints a service × target matrix for installed targets
- `uninstall` can optionally remove matching stored credentials for the selected service (interactive terminals)
- OpenCode target supports `~/.config/opencode/opencode.json` and `~/.config/opencode/opencode.jsonc` MCP configuration
- OpenCode config parsing supports JSONC-style content (comments and trailing commas) in both `.json` and `.jsonc` files
- Claude Code target supports both `~/.claude.json` and `~/.claude/settings.json`, and status includes project-scoped MCP entries

## Run locally

```bash
make test
make test-integration
make build
go run ./cmd/mcp-wire list services
go run ./cmd/mcp-wire list targets
go run ./cmd/mcp-wire status
go run ./cmd/mcp-wire --help
```

## Next steps

- Add more verified service definition files under `services/`
- Improve output UX (status formatting/symbols and summaries)
- Expand user docs for service and target contribution workflows
- Add additional target implementations (for example Gemini CLI)
