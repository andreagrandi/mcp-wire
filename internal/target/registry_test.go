package target

import (
	"testing"

	"github.com/andreagrandi/mcp-wire/internal/service"
)

type fakeTarget struct {
	name      string
	slug      string
	installed bool
}

func (f fakeTarget) Name() string {
	return f.name
}

func (f fakeTarget) Slug() string {
	return f.slug
}

func (f fakeTarget) IsInstalled() bool {
	return f.installed
}

func (f fakeTarget) Install(_ service.Service, _ map[string]string) error {
	return nil
}

func (f fakeTarget) Uninstall(_ string) error {
	return nil
}

func (f fakeTarget) List() ([]string, error) {
	return nil, nil
}

func TestAllTargetsReturnsCopyOfKnownTargets(t *testing.T) {
	setKnownTargetsForTest(t, []Target{
		fakeTarget{name: "Alpha CLI", slug: "alpha-cli", installed: true},
		fakeTarget{name: "Beta CLI", slug: "beta-cli", installed: false},
	})

	targets := AllTargets()
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}

	targets[0] = fakeTarget{name: "Mutated", slug: "mutated-cli", installed: true}

	if knownTargets[0].Slug() != "alpha-cli" {
		t.Fatalf("expected known targets to remain unchanged, got %q", knownTargets[0].Slug())
	}
}

func TestInstalledTargetsReturnsOnlyInstalledTargets(t *testing.T) {
	setKnownTargetsForTest(t, []Target{
		fakeTarget{name: "Alpha CLI", slug: "alpha-cli", installed: true},
		fakeTarget{name: "Beta CLI", slug: "beta-cli", installed: false},
		fakeTarget{name: "Gamma CLI", slug: "gamma-cli", installed: true},
	})

	installed := InstalledTargets()
	if len(installed) != 2 {
		t.Fatalf("expected 2 installed targets, got %d", len(installed))
	}

	if installed[0].Slug() != "alpha-cli" {
		t.Fatalf("expected first installed target alpha-cli, got %q", installed[0].Slug())
	}

	if installed[1].Slug() != "gamma-cli" {
		t.Fatalf("expected second installed target gamma-cli, got %q", installed[1].Slug())
	}
}

func TestFindTargetMatchesSlug(t *testing.T) {
	setKnownTargetsForTest(t, []Target{
		fakeTarget{name: "Alpha CLI", slug: "alpha-cli", installed: true},
		fakeTarget{name: "Beta CLI", slug: "beta-cli", installed: true},
	})

	target, ok := FindTarget("  ALPHA-CLI ")
	if !ok {
		t.Fatal("expected target to be found")
	}

	if target.Slug() != "alpha-cli" {
		t.Fatalf("expected alpha-cli slug, got %q", target.Slug())
	}
}

func TestFindTargetReturnsFalseWhenMissing(t *testing.T) {
	setKnownTargetsForTest(t, []Target{
		fakeTarget{name: "Alpha CLI", slug: "alpha-cli", installed: true},
	})

	target, ok := FindTarget("unknown-cli")
	if ok {
		t.Fatal("expected target lookup to fail")
	}

	if target != nil {
		t.Fatal("expected missing target to be nil")
	}
}

func setKnownTargetsForTest(t *testing.T, targets []Target) {
	t.Helper()

	originalTargets := knownTargets
	knownTargets = targets

	t.Cleanup(func() {
		knownTargets = originalTargets
	})
}
