# Changelog

## [Unreleased]

### Added
- Guided interactive UX with Survey for TTY sessions:
  - `mcp-wire` now opens a main menu (`Install service`, `Uninstall service`, `Status`, `List services`, `List targets`, `Exit`).
  - `mcp-wire install` and `mcp-wire uninstall` without positional args now open step-by-step wizards.
  - Service selection supports type-to-filter; target selection supports multi-select with sensible defaults.
  - Wizards include review and explicit equivalent non-interactive command output.
- Post-install OpenCode OAuth hints for OAuth services (`jira`, `sentry`, `context7`) to guide immediate authentication.
- Expanded CLI test coverage for guided flows, interactive selection paths, and OAuth hint behavior.

### Changed
- Bundled services migrated to OAuth-first remote endpoints:
  - `jira` -> `https://mcp.atlassian.com/v1/mcp`
  - `sentry` -> `https://mcp.sentry.dev/mcp`
  - `context7` -> `https://mcp.context7.com/mcp/oauth`
  and their install-time env credential requirements were removed.
- Hidden credential input UX now keeps cursor visibility and prints explicit guidance before masked entry (`Input hidden. Paste and press Enter.`).
- README now starts with interactive terminal walkthrough examples that demonstrate the guided install flow and expected output.
- Disabled `setup-go` cache in CI and release workflows for more deterministic pipeline behavior.

## v0.1.0 - 2026-02-13

### Added
- Core CLI commands for MCP management: `list`, `install`, `uninstall`, and `status`.
- Service definition system with YAML loading, validation, and layered resolution from bundled plus user-local directories.
- Target support for Claude Code, Codex, and OpenCode with safe config updates that preserve unknown keys.
- Credential resolution chain with environment and file store support, plus interactive prompting for required values.
- Initial bundled service definitions: `context7`, `jira`, and `sentry`.
- Integration test harness (`make test-integration`) and expanded unit test coverage across services, targets, credentials, and CLI flows.
- Release automation with GoReleaser, GitHub release workflow, and Homebrew tap publication.

### Changed
- Claude Code config handling now supports both `~/.claude.json` and `~/.claude/settings.json`, and status discovery includes project-scoped MCP entries.
- OpenCode config parsing now accepts JSONC-style syntax in both `.json` and `.jsonc` configuration files.
- README and contributor guidance were expanded with current implementation status, installation instructions, and release workflow details.
