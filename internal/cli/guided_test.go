package cli

import (
	"bufio"
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/andreagrandi/mcp-wire/internal/service"
	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
)

func TestFilterServicesMatchesNameAndDescription(t *testing.T) {
	services := []service.Service{
		{Name: "jira", Description: "Issue tracking"},
		{Name: "context7", Description: "Documentation lookup"},
	}

	nameMatches := filterServices(services, "jir")
	if len(nameMatches) != 1 || nameMatches[0].Name != "jira" {
		t.Fatalf("expected jira name match, got %#v", nameMatches)
	}

	descriptionMatches := filterServices(services, "lookup")
	if len(descriptionMatches) != 1 || descriptionMatches[0].Name != "context7" {
		t.Fatalf("expected context7 description match, got %#v", descriptionMatches)
	}
}

func TestParseTargetSelectionAllIncludesOnlyInstalledTargets(t *testing.T) {
	targets := []targetpkg.Target{
		fakeListTarget{name: "Alpha", slug: "alpha", installed: true},
		fakeListTarget{name: "Beta", slug: "beta", installed: false},
		fakeListTarget{name: "Gamma", slug: "gamma", installed: true},
	}

	selected, err := parseTargetSelection("all", targets)
	if err != nil {
		t.Fatalf("expected all selection to succeed: %v", err)
	}

	if len(selected) != 2 {
		t.Fatalf("expected only installed targets selected, got %d", len(selected))
	}
}

func TestParseTargetSelectionRejectsNotInstalledTarget(t *testing.T) {
	targets := []targetpkg.Target{
		fakeListTarget{name: "Alpha", slug: "alpha", installed: true},
		fakeListTarget{name: "Beta", slug: "beta", installed: false},
	}

	_, err := parseTargetSelection("2", targets)
	if err == nil {
		t.Fatal("expected selection of non-installed target to fail")
	}
}

func TestPickSourceInteractiveSelectsCurated(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("1\n"))
	var output bytes.Buffer

	source, err := pickSourceInteractive(&output, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if source != "curated" {
		t.Fatalf("expected curated, got %q", source)
	}
}

func TestPickSourceInteractiveSelectsRegistry(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("2\n"))
	var output bytes.Buffer

	source, err := pickSourceInteractive(&output, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if source != "registry" {
		t.Fatalf("expected registry, got %q", source)
	}
}

func TestPickSourceInteractiveSelectsBoth(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("3\n"))
	var output bytes.Buffer

	source, err := pickSourceInteractive(&output, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if source != "all" {
		t.Fatalf("expected all, got %q", source)
	}
}

func TestPickSourceInteractiveDefaultsCurated(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	var output bytes.Buffer

	source, err := pickSourceInteractive(&output, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if source != "curated" {
		t.Fatalf("expected curated as default, got %q", source)
	}
}

func TestPickSourceInteractiveRejectsInvalid(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("x\n1\n"))
	var output bytes.Buffer

	source, err := pickSourceInteractive(&output, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if source != "curated" {
		t.Fatalf("expected curated after retry, got %q", source)
	}

	if !strings.Contains(output.String(), "Invalid selection") {
		t.Fatalf("expected invalid selection message, got %q", output.String())
	}
}

func TestPickServiceInteractiveCatalogRegistryOnlyReturnsError(t *testing.T) {
	stubLoadServicesForCatalog(t)
	stubLoadRegistryCache(t, fakeRegistryServers())

	reader := bufio.NewReader(strings.NewReader("\n1\n"))
	var output bytes.Buffer

	_, err := pickServiceInteractiveCatalog(&output, reader, "registry")
	if !errors.Is(err, errRegistryOnly) {
		t.Fatalf("expected errRegistryOnly, got %v", err)
	}

	if !strings.Contains(output.String(), "Registry services cannot be installed yet") {
		t.Fatalf("expected rejection message, got %q", output.String())
	}
}

func TestPickServiceInteractiveSupportsSearch(t *testing.T) {
	services := map[string]service.Service{
		"jira": {
			Name:        "jira",
			Description: "Issue tracker",
		},
		"context7": {
			Name:        "context7",
			Description: "Documentation",
		},
	}

	reader := bufio.NewReader(strings.NewReader("doc\n1\n"))
	var output bytes.Buffer

	svc, err := pickServiceInteractive(&output, reader, services, false, "curated")
	if err != nil {
		t.Fatalf("expected service picker to succeed: %v", err)
	}

	if svc.Name != "context7" {
		t.Fatalf("expected context7 from search flow, got %q", svc.Name)
	}
}
