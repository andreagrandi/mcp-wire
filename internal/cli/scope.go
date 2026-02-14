package cli

import (
	"fmt"
	"strings"

	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
)

func parseInstallUninstallScope(value string) (targetpkg.ConfigScope, error) {
	scope := targetpkg.ConfigScope(strings.ToLower(strings.TrimSpace(value)))
	if scope == "" {
		scope = targetpkg.ConfigScopeUser
	}

	switch scope {
	case targetpkg.ConfigScopeUser, targetpkg.ConfigScopeProject:
		return scope, nil
	default:
		return "", fmt.Errorf("invalid scope %q (supported: user, project)", value)
	}
}

func parseStatusScope(value string) (targetpkg.ConfigScope, error) {
	scope := targetpkg.ConfigScope(strings.ToLower(strings.TrimSpace(value)))
	if scope == "" {
		scope = targetpkg.ConfigScopeEffective
	}

	switch scope {
	case targetpkg.ConfigScopeUser, targetpkg.ConfigScopeProject, targetpkg.ConfigScopeEffective:
		return scope, nil
	default:
		return "", fmt.Errorf("invalid scope %q (supported: user, project, effective)", value)
	}
}

func targetSupportsScope(targetDefinition targetpkg.Target, scope targetpkg.ConfigScope) bool {
	scopedTarget, ok := targetDefinition.(targetpkg.ScopedTarget)
	if !ok {
		return false
	}

	for _, supportedScope := range scopedTarget.SupportedScopes() {
		if supportedScope == scope {
			return true
		}
	}

	return false
}

func anyTargetSupportsProjectScope(targetDefinitions []targetpkg.Target) bool {
	for _, targetDefinition := range targetDefinitions {
		if targetSupportsScope(targetDefinition, targetpkg.ConfigScopeProject) {
			return true
		}
	}

	return false
}

func scopeDescription(scope targetpkg.ConfigScope) string {
	switch scope {
	case targetpkg.ConfigScopeProject:
		return "project"
	case targetpkg.ConfigScopeUser:
		return "user"
	case targetpkg.ConfigScopeEffective:
		return "effective"
	default:
		return string(scope)
	}
}
