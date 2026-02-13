# mcp-wire

<img src="mcp-wire-logo.png" alt="mcp-wire logo" width="50%" />

mcp-wire is a Go CLI that installs and configures MCP (Model Context Protocol) servers across multiple AI coding tools from one interface.

## What is implemented so far

- Cobra CLI foundation with version/help support
- Core service data model in `internal/service/service.go`
- Service registry loader with YAML parsing, validation, and path precedence in `internal/service/registry.go`
- Target abstraction in `internal/target/target.go`
- Target registry with discovery helpers in `internal/target/registry.go`
- First target implementation in `internal/target/claudecode.go`
- Unit tests for service loading/validation and target behavior

## Current behavior

- Service definitions can be loaded from multiple directories and validated before use
- Validation supports `sse` and `stdio` transports
- Duplicate service names are resolved by load order (later paths override earlier paths)
- The current target implementation can detect installation, install/update entries, uninstall entries, and list configured services

## Run locally

```bash
go test ./...
go run ./cmd/mcp-wire --help
```

## Next steps

- Add `list`, `install`, `uninstall`, and `status` commands
- Implement credential resolution (environment + file store)
- Add more target implementations
- Add initial service definition files under `services/`
