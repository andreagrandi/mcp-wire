# mcp-wire

<img src="mcp-wire-logo.png" alt="mcp-wire logo" width="50%" />

mcp-wire is a Go CLI that installs and configures MCP (Model Context Protocol) servers across multiple AI coding tools from one interface.

## What is implemented so far

- Cobra CLI foundation with version/help support
- Core service data model in `internal/service/service.go`
- Service registry loader with YAML parsing, validation, and path precedence in `internal/service/registry.go`
- Target abstraction in `internal/target/target.go`
- Target registry with discovery helpers in `internal/target/registry.go`
- Target implementation for Claude Code in `internal/target/claudecode.go`
- Target implementation for Codex CLI in `internal/target/codex.go`
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
- CI workflow via GitHub Actions
- Changelog initialized in `CHANGELOG.md`
- Unit tests for service loading, targets, and credentials

## Current behavior

- Service definitions load from multiple directories and are validated before use
- Validation supports `sse` and `stdio` transports
- Duplicate service names are resolved by load order (later paths override earlier paths)
- Claude Code and Codex target implementations can detect installation, install/update entries, uninstall entries, and list configured services
- Target config writes preserve unknown user-defined keys by using map-based parsing
- Credential resolution supports environment variables first, then file-based credentials at `~/.config/mcp-wire/credentials` (stored with `0600` permissions)
- Interactive credential prompts can collect missing required values with optional setup URL opening and optional storage in the credential file store

## Run locally

```bash
make test
make build
go test ./...
go run ./cmd/mcp-wire --help
```

## Next steps

- Add `list`, `install`, `uninstall`, and `status` commands
- Wire interactive credential flow and resolver into the install command end-to-end
- Add initial service definition files under `services/`
- Add additional target implementations
