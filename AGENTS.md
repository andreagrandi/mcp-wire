This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Session Start Workflow

Before making code or documentation changes in this repo:

1. Switch back to `master`.
2. Pull the latest changes with `git pull --ff-only`.
3. Create a new branch with a short, descriptive name related to the feature being added or the bug being fixed.
4. Make the requested changes on that branch.

Do not start work from an old feature branch unless the user explicitly asks to continue that branch.

## Adding a Ticket, Issue, or Bug

When the user asks to add a "ticket", "issue", or "bug" to this repo, all of the
steps below are required — creating the GitHub issue alone is not enough.

**1. Create the issue** with the repo label and an area label:

```bash
gh issue create --repo andreagrandi/mcp-wire \
  --title "<concise title>" \
  --body "<description>" \
  --label "mcp-wire" \
  --label "<area>"
```

- Always apply the `mcp-wire` label — every work item in this repo carries it.
- Add the matching area label when one exists: `feature`, `ux`, `agent`,
  `docs`, `security`, `testing`, `release`. The `Reliability` and `Packaging`
  areas have no label; for those, set only the project Area field (step 3).
- Add a type label when it fits: `bug` (bug report), `enhancement` (feature
  request), or `documentation` (docs-only work).

**2. Add the issue to the "CLI Tools" project** and capture the item ID
(https://github.com/users/andreagrandi/projects/1):

```bash
ITEM_ID=$(gh project item-add 1 --owner andreagrandi \
  --url <issue-url> --format json --jq .id)
```

**3. Set Priority, Area, and Status** on the project item. The project and
field IDs are identical for every repo on this board:

- Project ID: `PVT_kwHOAAm1584BYDlZ`
- Priority — field `PVTSSF_lAHOAAm1584BYDlZzhTMDck`:
  High `ed3787e3`, Medium `3e3ea407`, Low `994234f4`
- Area — field `PVTSSF_lAHOAAm1584BYDlZzhTMDco`:
  Reliability `6595432d`, Packaging `6895c50a`, UX `2bc024bb`,
  Testing `0d5bc016`, Feature `6390f97d`, Agent `3a2d6f7e`,
  Docs `f5c50514`, Security `062d12a3`, Release `b344aeab`
- Status — field `PVTSSF_lAHOAAm1584BYDlZzhTMDNQ`:
  Todo `f75ad846`, In Progress `47fc9ee4`, Done `98236657`

```bash
# Priority — always set it
gh project item-edit --id "$ITEM_ID" --project-id PVT_kwHOAAm1584BYDlZ \
  --field-id PVTSSF_lAHOAAm1584BYDlZzhTMDck \
  --single-select-option-id <priority-option-id>

# Area — match the area label from step 1
gh project item-edit --id "$ITEM_ID" --project-id PVT_kwHOAAm1584BYDlZ \
  --field-id PVTSSF_lAHOAAm1584BYDlZzhTMDco \
  --single-select-option-id <area-option-id>

# Status — new tickets start as Todo
gh project item-edit --id "$ITEM_ID" --project-id PVT_kwHOAAm1584BYDlZ \
  --field-id PVTSSF_lAHOAAm1584BYDlZzhTMDNQ \
  --single-select-option-id f75ad846
```

If the user does not state a priority or area, ask before creating the issue.
Follow the conventions of existing project issues — do not invent new labels or
fields.

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

Always format the code with gofmt before commit and push

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

## Changelog updates

`CHANGELOG.md` follows the Keep a Changelog format with an `[Unreleased]` section on top. Every PR that introduces a user-visible change must add an entry under `[Unreleased]` in the same commit/PR — do not wait for release time.

A change is user-visible if it affects any of:

- CLI commands, flags, prompts, or output (TUI or plain-text)
- Bundled service definitions (added/removed/renamed services, transport or env changes)
- Target support (new targets, scope behavior, config file layout)
- Install/uninstall/credential/OAuth flows
- Packaging, distribution, install paths, release artifacts
- Public Go types or interfaces that other targets/services rely on
- Documented behavior in `README.md` or the service/target schema

A change is NOT user-visible (and does not require an entry) when it only touches:

- Tests, fixtures, or test helpers
- Internal refactors with no behavior change
- CI/workflow files, lint config, or developer tooling
- Comments, formatting, or non-user-facing docs (`CLAUDE.md`, `AGENTS.md`, internal plans)

### How to write the entry

- Add a bullet under `## [Unreleased]`, grouped by `### Added`, `### Changed`, `### Removed`, `### Fixed`, or `### Security` (create the subsection if it does not exist yet).
- Be concise: one sentence per bullet, describing the user-visible effect — not the implementation.
- Lead with the verb (`Added`, `Removed`, `Renamed`, `Fixed`) and name the thing as users see it (command, flag, service name, target slug, screen).
- Do not invent a new version header; only the release process promotes `[Unreleased]` to a versioned section.

Example:

```markdown
## [Unreleased]

### Added
- New `mcp-wire doctor` command that prints detected target config paths and their write status.

### Changed
- `install` now prints the resolved config path for each target after a successful write.
```

### Enforcement

CI runs a `changelog` check on every PR (`.github/workflows/changelog.yml`) that fails when source code under `internal/`, `cmd/`, `services/`, or `README.md` changes without a matching update to `CHANGELOG.md`. The check is skipped automatically for PRs labeled `skip-changelog` or whose title contains `[no-changelog]` — use these only for changes that are genuinely not user-visible (test-only, refactor-only, CI-only).

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

- Rename the `## [Unreleased]` header to `## v<version> - <YYYY-MM-DD>`, keeping all bullets in place.
- Add a fresh empty `## [Unreleased]` header above it for the next cycle.
- If `[Unreleased]` is empty at release time, that means changelog discipline slipped during the cycle — review the commits since the last tag and backfill bullets before tagging.

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
