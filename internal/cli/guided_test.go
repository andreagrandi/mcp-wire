package cli

import (
	"bufio"
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/andreagrandi/mcp-wire/internal/catalog"
	"github.com/andreagrandi/mcp-wire/internal/registry"
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

	reader := bufio.NewReader(strings.NewReader("\n1\ny\n"))
	var output bytes.Buffer

	_, err := pickServiceInteractiveCatalog(&output, reader, "registry")
	if !errors.Is(err, errRegistryOnly) {
		t.Fatalf("expected errRegistryOnly, got %v", err)
	}

	if !strings.Contains(output.String(), "Registry Service Information:") {
		t.Fatalf("expected trust summary in output, got %q", output.String())
	}

	if !strings.Contains(output.String(), "This registry service has no supported install method") {
		t.Fatalf("expected rejection message, got %q", output.String())
	}
}

func TestPickServiceInteractiveCatalogDeclineTrustGoesBack(t *testing.T) {
	stubLoadServicesForCatalog(t)
	stubLoadRegistryCache(t, []registry.ServerResponse{
		{
			Server: registry.ServerJSON{
				Name:        "gamma",
				Description: "Gamma from registry",
				Remotes: []registry.Transport{
					{Type: "sse", URL: "https://gamma.example.com/sse"},
				},
			},
		},
	})

	// Search all, select 1 (gamma), decline trust, search all again, select 1, accept trust
	reader := bufio.NewReader(strings.NewReader("\n1\nn\n\n1\ny\n"))
	var output bytes.Buffer

	svc, err := pickServiceInteractiveCatalog(&output, reader, "registry")
	if err != nil {
		t.Fatalf("expected service to be returned after re-accepting trust, got %v", err)
	}

	if svc.Name != "gamma" {
		t.Fatalf("expected service name %q, got %q", "gamma", svc.Name)
	}

	outputStr := output.String()

	// Trust summary should appear twice (once per selection attempt)
	firstIdx := strings.Index(outputStr, "Registry Service Information:")
	if firstIdx == -1 {
		t.Fatalf("expected trust summary in output, got %q", outputStr)
	}

	secondIdx := strings.Index(outputStr[firstIdx+1:], "Registry Service Information:")
	if secondIdx == -1 {
		t.Fatalf("expected second trust summary after declining, got %q", outputStr)
	}
}

func TestPickServiceInteractiveCatalogTrustSummaryShowsTransport(t *testing.T) {
	stubLoadServicesForCatalog(t)
	stubLoadRegistryCache(t, []registry.ServerResponse{
		{
			Server: registry.ServerJSON{
				Name:        "gamma",
				Description: "Gamma from registry",
				Version:     "1.0.0",
				Remotes: []registry.Transport{
					{Type: "streamable-http", URL: "https://gamma.example.com/mcp"},
				},
				Repository: &registry.Repository{
					URL: "https://github.com/example/gamma",
				},
			},
		},
	})

	reader := bufio.NewReader(strings.NewReader("\n1\ny\n"))
	var output bytes.Buffer

	_, _ = pickServiceInteractiveCatalog(&output, reader, "registry")
	outputStr := output.String()

	if !strings.Contains(outputStr, "Transport: streamable-http") {
		t.Fatalf("expected transport in trust summary, got %q", outputStr)
	}

	if !strings.Contains(outputStr, "Repo:      https://github.com/example/gamma") {
		t.Fatalf("expected repo URL in trust summary, got %q", outputStr)
	}
}

func TestPickServiceInteractiveCatalogNoTrustForCurated(t *testing.T) {
	stubLoadServicesForCatalog(t)
	stubLoadRegistryCache(t, fakeRegistryServers())

	// source="all": curated alpha, beta + registry gamma, delta
	// Select alpha (curated) -> should NOT show trust summary
	reader := bufio.NewReader(strings.NewReader("alpha\n1\n"))
	var output bytes.Buffer

	svc, err := pickServiceInteractiveCatalog(&output, reader, "all")
	if err != nil {
		t.Fatalf("expected curated service to succeed: %v", err)
	}

	if svc.Name != "alpha" {
		t.Fatalf("expected alpha, got %q", svc.Name)
	}

	if strings.Contains(output.String(), "Registry Service Information:") {
		t.Fatalf("expected no trust summary for curated entry, got %q", output.String())
	}
}

func TestConfirmInstallSelectionShowsTrustForRegistry(t *testing.T) {
	entry := &catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "test-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:    "test-server",
				Version: "1.0.0",
				Remotes: []registry.Transport{
					{Type: "sse", URL: "https://example.com/sse"},
				},
			},
		},
	}

	svc := service.Service{Name: "test-server"}
	targets := []targetpkg.Target{
		fakeListTarget{name: "TestTarget", slug: "test", installed: true},
	}

	reader := bufio.NewReader(strings.NewReader("y\n"))
	var output bytes.Buffer

	confirmed, err := confirmInstallSelection(&output, reader, svc, targets, false, targetpkg.ConfigScopeUser, entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !confirmed {
		t.Fatal("expected confirmation to succeed")
	}

	if !strings.Contains(output.String(), "Registry Service Information:") {
		t.Fatalf("expected trust summary in confirmation output, got %q", output.String())
	}
}

func TestConfirmInstallSelectionNoTrustForCurated(t *testing.T) {
	svc := service.Service{Name: "test-service"}
	targets := []targetpkg.Target{
		fakeListTarget{name: "TestTarget", slug: "test", installed: true},
	}

	reader := bufio.NewReader(strings.NewReader("y\n"))
	var output bytes.Buffer

	confirmed, err := confirmInstallSelection(&output, reader, svc, targets, false, targetpkg.ConfigScopeUser, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !confirmed {
		t.Fatal("expected confirmation to succeed")
	}

	if strings.Contains(output.String(), "Registry Service Information:") {
		t.Fatalf("expected no trust summary for nil entry, got %q", output.String())
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
