package target

import (
	"io"

	"github.com/andreagrandi/mcp-wire/internal/service"
)

// ConfigScope controls where a target applies configuration when supported.
type ConfigScope string

const (
	ConfigScopeUser      ConfigScope = "user"
	ConfigScopeProject   ConfigScope = "project"
	ConfigScopeEffective ConfigScope = "effective"
)

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

// ScopedTarget can install/list/uninstall services in multiple scopes.
// Targets that do not implement this interface are treated as user/global only.
type ScopedTarget interface {
	SupportedScopes() []ConfigScope
	InstallWithScope(svc service.Service, resolvedEnv map[string]string, scope ConfigScope) error
	UninstallWithScope(serviceName string, scope ConfigScope) error
	ListWithScope(scope ConfigScope) ([]string, error)
}

// AuthTarget can perform an interactive authentication flow for a configured service.
type AuthTarget interface {
	Authenticate(serviceName string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error
}
