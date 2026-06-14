# Troubleshooting

Common setup problems and how to recover from them.

## `mcp-wire` says no targets are installed

mcp-wire detects each target by looking for its CLI binary in `PATH`:

| Target | Binary names checked |
|--------|----------------------|
| Claude Code | `claude`, `claude-code` |
| Codex CLI | `codex` |
| OpenCode | `opencode` |

If a target you expect is missing:

1. Install the target CLI (e.g. `npm install -g @anthropic-ai/claude-code` for Claude Code).
2. Make sure the binary directory is in your `PATH` in the shell where you run `mcp-wire`.
3. Run `mcp-wire doctor` to confirm detection and see the config path for each target.

## Registry search is empty or stale

If the registry feature is enabled but the service list looks wrong, the local cache may be stale or corrupt:

```bash
mcp-wire cache clear
```

Then run `mcp-wire` again. The registry syncs in the background on startup, so the first run after clearing may take a moment to repopulate.

To check whether the registry feature is enabled:

```bash
mcp-wire doctor
```

If it is disabled, enable it with:

```bash
mcp-wire feature enable registry
```

## Credentials are prompted every time

mcp-wire resolves credentials in this order, stopping at the first match:

1. Process environment variables.
2. `~/.config/mcp-wire/credentials`.
3. Interactive prompt (skipped with `--no-prompt`).

If you are prompted repeatedly:

- Check that the environment variable name matches exactly (case-sensitive).
- Verify the credentials file exists and contains the variable:
  ```bash
  cat ~/.config/mcp-wire/credentials
  ```
- Check permissions. The file should be `0600` and the directory `0700`:
  ```bash
  ls -ld ~/.config/mcp-wire
  ls -l ~/.config/mcp-wire/credentials
  ```
- Re-install the service and answer **Yes** at the "Save to credential store?" prompt.

To remove stored credentials, uninstall the service and answer **Yes** at the credential-cleanup prompt, or edit the file directly.

## OAuth service installed but not authenticated

Some targets can complete OAuth automatically; others need a manual follow-up step.

| Target | Automatic OAuth | Manual next step |
|--------|-----------------|------------------|
| Claude Code | No | Run `/mcp` inside Claude Code and follow the auth prompts for the installed server. |
| Codex CLI | Yes (`codex mcp login <service>`) | If it fails, run the command manually. |
| OpenCode | Yes (`opencode mcp auth <service>`) | If it fails, run the command manually. |

If automatic OAuth fails, the apply screen prints a hint like:

```
[!] Claude Code: In Claude Code, run /mcp to complete OAuth authentication.
```

## `--no-prompt` fails with missing credentials

`--no-prompt` never asks for input. Required env vars must already be set or have a `default` value in the service definition. To fix:

```bash
export SERVICE_TOKEN="your-token"
mcp-wire install <service> --target <target> --no-prompt
```

## Service not found

- For curated services, check the exact name with `mcp-wire metadata`.
- For registry services, make sure the registry feature is enabled and the cache is up to date:
  ```bash
  mcp-wire feature enable registry
  mcp-wire cache clear
  ```
- Direct installs from the registry work by exact registry name: `mcp-wire install ai.example/service --target claude`.

## Config changes look unexpected

mcp-wire reads each target config as `map[string]any` and only modifies the MCP server entries it manages. It preserves any other keys you have set manually. If a target's config looks wrong after install:

1. Run `mcp-wire doctor` to see the exact file path.
2. Back up the file before making manual edits.
3. Report an issue if mcp-wire removed or overwrote keys it should have preserved.

## Scope confusion (Claude Code)

Claude Code supports two scopes:

- `user` (default): server is available across projects, written to `~/.claude.json`.
- `project`: server is only for the current directory, written under the matching project key in the same file.

If a service is missing in one project but present in others, check whether it was installed with `--scope project` and whether the current directory matches the project path stored in the config.

## General diagnostics

`mcp-wire doctor` is read-only and shows:

- detected targets and their config paths
- whether each config file exists
- enabled feature flags
- mcp-wire config, credentials, user services, and registry cache paths
- hints for missing targets or disabled features

Run it first when something is not working as expected.
