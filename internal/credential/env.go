package credential

import (
	"os"
	"strings"
)

// EnvSource resolves credentials from process environment variables.
type EnvSource struct{}

// NewEnvSource creates a source backed by environment variables.
func NewEnvSource() EnvSource {
	return EnvSource{}
}

// Name returns a stable source name.
func (s EnvSource) Name() string {
	return "environment"
}

// Get returns the environment variable value when present.
func (s EnvSource) Get(envName string) (string, bool) {
	trimmedName := strings.TrimSpace(envName)
	if trimmedName == "" {
		return "", false
	}

	return os.LookupEnv(trimmedName)
}

// Store is not supported for environment variables.
func (s EnvSource) Store(_ string, _ string) error {
	return ErrNotSupported
}
