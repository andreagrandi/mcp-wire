# TUI Walkthrough

This guide shows the full-screen Bubble Tea TUI in action. The recordings were made with real `mcp-wire` runs using isolated fake targets so your own config is never touched.

## Install a service

Run `mcp-wire` without arguments to open the wizard:

```bash
mcp-wire
```

The install flow walks through source selection (when the registry feature is enabled), live-filtered service search, target multi-select, scope selection for supported targets, a review screen, and the final apply step.

![mcp-wire install TUI flow](assets/install.svg)

Key moments in the recording:

1. **Main menu** — choose *Install service*.
2. **Service search** — type to filter the curated catalog (here `playwright`).
3. **Target selection** — toggle the targets to write to with `Space`, then confirm with `Enter`.
4. **Scope** — for Claude Code you can pick `user` (default) or `project` scope.
5. **Review** — confirm the equivalent command before applying.
6. **Apply** — watch per-target progress and copy the equivalent command for scripts.

## Uninstall a service

The uninstall flow starts with the target, then shows only the services that are actually installed there.

![mcp-wire uninstall TUI flow](assets/uninstall.svg)

Key moments:

1. **Main menu** — choose *Uninstall service*.
2. **Target selection** — pick the target that has the service.
3. **Installed services** — only services present in that target are listed.
4. **Scope** — choose the scope to remove from, when the target supports it.
5. **Review** — confirm the uninstall.
6. **Apply** — the service is removed and, if the service had stored credentials, you are asked whether to delete them.

## Replay or re-record

The recordings are generated from maintainer scripts so they stay in sync with the actual TUI:

```bash
cd docs/assets
./record.sh
```

Requirements: `asciinema`, `expect`, and `svg-term-cli`.
