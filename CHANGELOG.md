# Changelog

## [Unreleased]

### Added
- Bundled service YAML files are now embedded into the binary, so `mcp-wire` can always list and install built-in services even when no external `services/` directory is present.
- Install flows now automatically trigger OAuth authentication for OAuth-marked services on targets that support it (currently Codex and OpenCode) when running interactively.
- Added a bundled `playwright` service definition using the official stdio setup (`npx @playwright/mcp@latest`) compatible with Claude Code, Codex, and OpenCode.

### Changed
- Service definitions now support an optional `auth` field (for example `auth: oauth`) to drive post-install authentication behavior.
- Guided target selection now starts with no preselected targets, so users explicitly choose where to install.
- Survey target picker now documents quick controls (`Right` to select all, `Left` to clear all), while plain prompts require explicit target numbers (or `all`).
- Guided Survey wizards now support `Esc` back navigation (review -> targets, targets -> service, service -> cancel wizard).
- Pressing `Esc` in the guided main menu now keeps the user in the menu instead of failing the prompt.

## v0.1.1 - 2026-02-13

### Added
- Guided interactive UX with Survey for TTY sessions, including a main menu and step-by-step install/uninstall wizards.
- Service and target selection now support filtering, multi-select, review, and equivalent command output.
- Expanded CLI test coverage for guided flows, interactive selection paths, and install output behavior.
- README now includes an interactive terminal walkthrough at the top.

### Changed
- Bundled `jira`, `sentry`, and `context7` services now use OAuth-first remote endpoints, removing install-time env credential requirements.
- Hidden credential input UX now keeps cursor visibility and prints explicit guidance before masked entry (`Input hidden. Paste and press Enter.`).
- Install output no longer prints target-native OAuth commands; guided output focuses on `mcp-wire` workflow and equivalent command hints.
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
