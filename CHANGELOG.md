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
- Target implementation for OpenCode using JSON/JSONC config (`~/.config/opencode/opencode.json` / `opencode.jsonc`) with install, uninstall, and list behavior.
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
- `list` commands for services and targets.
- `install` command with target filtering, `--no-prompt`, credential resolution chain wiring, and per-target install results.
- `uninstall` command with target filtering and per-target uninstall results.
- Optional interactive credential cleanup during uninstall for matching service environment keys.
- `status` command with service × target matrix output for installed targets.
- Initial bundled service definition files in `services/` (`context7`, `jira`, `sentry`).
- Service registry support for loading `services/` from the current working directory.
- Credential file delete helpers (`Delete`, `DeleteMany`) for cleanup flows.
- Improved Claude target detection across multiple binary names and local fallback installation paths.
- OpenCode target detection via PATH and local fallback install path (`~/.opencode/bin/opencode`).
- JSONC parsing dependency for OpenCode config compatibility (`github.com/tidwall/jsonc`).
- Test coverage for install, uninstall, status, and credential cleanup flows.
- Test coverage for OpenCode target install, uninstall, list, JSONC parsing, and detection behavior.
- Integration test harness for CLI lifecycle flows (`list`, `install`, `status`, `uninstall`) using a sandboxed environment.
- `make test-integration` target for opt-in integration tests.
- Claude config path compatibility for both `~/.claude.json` and `~/.claude/settings.json`.
- GoReleaser configuration for multi-platform release artifacts and checksums.
- Homebrew tap publishing through GoReleaser (`andreagrandi/homebrew-tap`, `Formula/mcp-wire.rb`).
- Tag-triggered release workflow (`.github/workflows/release.yml`) that runs tests and publishes releases.

### Changed
- README expanded with implementation status, local run instructions, and next-step roadmap.
- README updated to reflect interactive credential flow progress.
- README updated to reflect current CLI command coverage and bundled services.
- README updated to reflect OpenCode target support.
- README updated with Homebrew installation instructions and release automation status.
- OpenCode config parsing now accepts JSONC-style content in both `.json` and `.jsonc` config files.
- Claude status discovery now includes both global and project-scoped MCP server entries.
- Project logo added and refreshed, with scaled rendering in README.
- Repository development guidance file (`AGENTS.md`) added.
