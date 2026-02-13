package cli

import (
	"bufio"
	"bytes"
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

	svc, err := pickServiceInteractive(&output, reader, services)
	if err != nil {
		t.Fatalf("expected service picker to succeed: %v", err)
	}

	if svc.Name != "context7" {
		t.Fatalf("expected context7 from search flow, got %q", svc.Name)
	}
}
