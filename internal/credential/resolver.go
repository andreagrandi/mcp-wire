package credential

import (
	"errors"
	"strings"
)

// ErrNotSupported is returned by sources that do not support persisting values.
var ErrNotSupported = errors.New("store operation not supported")

// Source defines a credential source.
type Source interface {
	Name() string
	Get(envName string) (string, bool)
	Store(envName string, value string) error
}

// Resolver resolves credentials by checking sources in order.
type Resolver struct {
	sources []Source
}

// NewResolver creates a resolver with a fixed source order.
func NewResolver(sources ...Source) *Resolver {
	resolverSources := make([]Source, len(sources))
	copy(resolverSources, sources)

	return &Resolver{sources: resolverSources}
}

// Resolve tries each source in order.
//
// It returns the value, source name, and whether a value was found.
func (r *Resolver) Resolve(envName string) (value string, source string, found bool) {
	if r == nil {
		return "", "", false
	}

	trimmedName := strings.TrimSpace(envName)
	if trimmedName == "" {
		return "", "", false
	}

	for _, src := range r.sources {
		if src == nil {
			continue
		}

		resolvedValue, ok := src.Get(trimmedName)
		if !ok {
			continue
		}

		return resolvedValue, src.Name(), true
	}

	return "", "", false
}
