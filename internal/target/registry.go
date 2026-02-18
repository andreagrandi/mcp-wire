package target

import "strings"

var knownTargets = []Target{
	NewClaudeCodeTarget(),
	NewCodexTarget(),
	NewGeminiCLITarget(),
	NewOpenCodeTarget(),
}

// AllTargets returns all known targets.
func AllTargets() []Target {
	targets := make([]Target, len(knownTargets))
	copy(targets, knownTargets)

	return targets
}

// InstalledTargets returns the subset of targets available on this system.
func InstalledTargets() []Target {
	installedTargets := make([]Target, 0, len(knownTargets))

	for _, target := range knownTargets {
		if !target.IsInstalled() {
			continue
		}

		installedTargets = append(installedTargets, target)
	}

	return installedTargets
}

// FindTarget looks up a target by slug.
func FindTarget(slug string) (Target, bool) {
	normalizedSlug := strings.ToLower(strings.TrimSpace(slug))

	for _, target := range knownTargets {
		targetSlug := strings.ToLower(strings.TrimSpace(target.Slug()))
		if targetSlug == normalizedSlug {
			return target, true
		}
	}

	return nil, false
}
