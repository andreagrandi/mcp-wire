# Changelog

## [Unreleased]

### Added
- Install and uninstall wizards now include a source selection step (Curated, Registry, Both) when the registry feature is enabled, in both plain and survey UIs.
- `list services --source all` now marks curated entries with a `*` prefix and prints a legend line; single-source views show no markers.
- `--source` flag on `list services` is now visible in help output when the registry feature is enabled, and hidden otherwise.
- Selecting a registry service now shows a trust/safety summary (source, install type, transport, required secrets, repository URL) and requires explicit confirmation before proceeding, in both plain and survey UIs.
- Registry entries with `streamable-http` or `sse` remotes can now be installed end-to-end: URL/header variable prompting, placeholder substitution, and per-target config generation.
- `Service` model now carries explicit `Headers` (for registry remote header templates) and `EnvVar.Default` (for variable defaults from the registry schema).
- Claude Code target emits a `headers` map in server config when explicit headers are defined.
- OpenCode target uses explicit `svc.Headers` for registry remotes instead of deriving headers from resolved env vars.
- Codex target skips `bearer_token_env_var` for registry remotes (where env vars are URL/header substitution variables, not bearer tokens).
- `--no-prompt` mode now silently applies default values for required and optional env vars, enabling non-interactive installs of registry services with defaulted variables.
- Optional env vars with defaults are populated into resolved env so URL/header substitution can use them.

### Changed
- Selecting a registry service in the install/uninstall wizard shows a rejection message and returns to source selection instead of looping indefinitely.
- Catalog display in wizards and `list services` uses `*` curated markers instead of `[registry]` suffix tags when showing mixed sources.
- Registry rejection message now reads "This registry service has no supported remote transport. Package-based install is not yet supported." instead of the generic "Registry services cannot be installed yet."
- Duplicate env vars from URL variables and header variables are now merged with OR on `Required`, matching the existing `envVarsFromRegistry` dedup pattern.

## v0.1.3 - 2026-02-14

### Changed
- Install, uninstall, and status flows are now scope-aware for Claude Code, with explicit `--scope` support (`effective`, `user`, `project`) and guided prompts where applicable.
- Renamed the Claude Code target slug from `claudecode` to `claude` to align with the official CLI naming.
- OAuth UX now provides clearer follow-up guidance, including a Claude-specific `/mcp` next-step hint when automatic CLI auth is unavailable.
- Post-install output now highlights next steps and equivalent commands more clearly in guided flows.
- Added first-class `http` transport support across service validation and target config generation (alongside `sse` and `stdio`).
- Bundled OAuth services (`jira`, `sentry`, `context7`) now use `transport: http`, and contributor docs were updated with clearer transport guidance and examples.

## v0.1.2 - 2026-02-13

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
