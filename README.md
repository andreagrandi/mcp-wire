# mcp-wire

<img src="mcp-wire-logo.png" alt="mcp-wire logo" width="50%" />

mcp-wire is a Go CLI that installs and configures MCP (Model Context Protocol) servers across multiple AI coding tools from one interface.

## Interactive in practice (text walkthrough)

No manual config editing needed. In a terminal, `mcp-wire` opens a guided menu.

```text
$ mcp-wire
Use Up/Down arrows, Enter to select.
> Main Menu
  Install service
  Uninstall service
  Status
  List services
  List targets
  Exit
```

Install flow (example):

```text
$ mcp-wire install
Install Wizard

Step 1/4: Service
Use Up/Down arrows, Enter to select. Type to filter.
> Select service
  jira - Atlassian Rovo MCP server (OAuth)

Step 2/4: Targets
Detected targets:
  OpenCode (opencode) [installed]
  Codex (codex) [installed]
Use Up/Down arrows, Space to toggle, Enter to confirm. Type to filter.
[x] OpenCode (opencode)
[ ] Codex (codex)

Step 3/4: Review
Service: jira
Targets: OpenCode
Credentials: prompt as needed
Use Up/Down arrows, Enter to select.
> Apply changes?
  Yes
  No

Step 4/4: Apply
Installing to: OpenCode
  OpenCode: configured
Equivalent command: mcp-wire install jira --target opencode
```

Quick checks:

```bash
mcp-wire status
mcp-wire list services
```

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap andreagrandi/tap
brew install mcp-wire
```

### From source

```bash
git clone https://github.com/andreagrandi/mcp-wire
cd mcp-wire
make build
./bin/mcp-wire --help
```

## Run locally

```bash
make test
make test-integration
make build
go run ./cmd/mcp-wire list services
go run ./cmd/mcp-wire list targets
go run ./cmd/mcp-wire status
go run ./cmd/mcp-wire --help
```

## Next steps

- Add more verified service definition files under `services/`
- Improve output UX (status formatting/symbols and summaries)
- Expand user docs for service and target contribution workflows
- Add additional target implementations (for example Gemini CLI)
