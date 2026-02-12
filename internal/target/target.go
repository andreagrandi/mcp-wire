package target

import "github.com/andreagrandi/mcp-wire/internal/service"

// Target defines how a CLI integration manages MCP services.
type Target interface {
	// Name returns a human-readable target name.
	Name() string

	// Slug returns a CLI-friendly target identifier.
	Slug() string

	// IsInstalled reports whether this target is available on the system.
	IsInstalled() bool

	// Install writes a service configuration into the target.
	// resolvedEnv maps environment variable names to resolved values.
	Install(svc service.Service, resolvedEnv map[string]string) error

	// Uninstall removes a service configuration from the target.
	Uninstall(serviceName string) error

	// List returns the names of currently configured services.
	List() ([]string, error)
}
