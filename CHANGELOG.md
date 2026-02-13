# Changelog

## [Unreleased]

### Added
- Initial project scaffolding for a Go-based CLI tool.
- Cobra CLI foundation (`cmd/mcp-wire`) with root command, help/version support, and tests.
- Core service data model and YAML parsing support in `internal/service`.
- Service registry with multi-path loading, validation, and user-local precedence rules.
- Target abstraction (`Target` interface) and target registry/discovery helpers.
- Target implementation for Claude Code with install, uninstall, list, and config-preservation behavior.
- Target implementation for Codex CLI using TOML config (`~/.codex/config.toml`) with install, uninstall, and list behavior.
- Credential resolution foundation in `internal/credential`:
  - source interface and resolver chain
  - environment variable source
  - file-based credentials source at `~/.config/mcp-wire/credentials` with `0600` permissions
- Interactive credential flow helpers in `internal/cli` for required variables:
  - setup URL and setup hint prompts
  - optional browser opening (`open` / `xdg-open` / `start`)
  - masked secret input on terminal sessions
  - optional persistence to the file-based credential store
- Test coverage for the interactive credential flow and prompt utilities.
- Terminal secret input dependency (`golang.org/x/term`).
- Automated CI workflow via GitHub Actions.

### Changed
- README expanded with implementation status, local run instructions, and next-step roadmap.
- README updated to reflect interactive credential flow progress.
- Project logo added and refreshed, with scaled rendering in README.
- Repository development guidance file (`AGENTS.md`) added.
