# Changelog

## [Unreleased]

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
