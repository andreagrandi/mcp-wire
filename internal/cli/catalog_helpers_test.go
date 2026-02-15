package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/andreagrandi/mcp-wire/internal/catalog"
	"github.com/andreagrandi/mcp-wire/internal/registry"
	"github.com/andreagrandi/mcp-wire/internal/service"
)

func stubLoadServicesForCatalog(t *testing.T) {
	t.Helper()

	original := loadServices
	t.Cleanup(func() { loadServices = original })

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"alpha": {Name: "alpha", Description: "Alpha service"},
			"beta":  {Name: "beta", Description: "Beta service"},
		}, nil
	}
}

func stubLoadRegistryCache(t *testing.T, servers []registry.ServerResponse) {
	t.Helper()

	original := loadRegistryCache
	t.Cleanup(func() { loadRegistryCache = original })

	loadRegistryCache = func() []registry.ServerResponse {
		return servers
	}
}

func fakeRegistryServers() []registry.ServerResponse {
	return []registry.ServerResponse{
		{
			Server: registry.ServerJSON{
				Name:        "gamma",
				Description: "Gamma from registry",
			},
		},
		{
			Server: registry.ServerJSON{
				Name:        "delta",
				Description: "Delta from registry",
			},
		},
	}
}

func TestLoadCatalogCuratedOnly(t *testing.T) {
	stubLoadServicesForCatalog(t)
	stubLoadRegistryCache(t, fakeRegistryServers())

	cat, err := loadCatalog("curated", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := cat.All()
	if len(entries) != 2 {
		t.Fatalf("expected 2 curated entries, got %d", len(entries))
	}

	for _, e := range entries {
		if e.Source != catalog.SourceCurated {
			t.Fatalf("expected curated source, got %q", e.Source)
		}
	}
}

func TestLoadCatalogAllWithRegistryEnabled(t *testing.T) {
	stubLoadServicesForCatalog(t)
	stubLoadRegistryCache(t, fakeRegistryServers())

	cat, err := loadCatalog("all", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := cat.All()
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries (2 curated + 2 registry), got %d", len(entries))
	}

	curatedCount := 0
	registryCount := 0
	for _, e := range entries {
		switch e.Source {
		case catalog.SourceCurated:
			curatedCount++
		case catalog.SourceRegistry:
			registryCount++
		}
	}

	if curatedCount != 2 {
		t.Fatalf("expected 2 curated entries, got %d", curatedCount)
	}

	if registryCount != 2 {
		t.Fatalf("expected 2 registry entries, got %d", registryCount)
	}
}

func TestLoadCatalogRegistryOnly(t *testing.T) {
	stubLoadServicesForCatalog(t)
	stubLoadRegistryCache(t, fakeRegistryServers())

	cat, err := loadCatalog("registry", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := cat.All()
	if len(entries) != 2 {
		t.Fatalf("expected 2 registry entries, got %d", len(entries))
	}

	for _, e := range entries {
		if e.Source != catalog.SourceRegistry {
			t.Fatalf("expected registry source, got %q", e.Source)
		}
	}
}

func TestLoadCatalogAllWithRegistryDisabled(t *testing.T) {
	stubLoadServicesForCatalog(t)
	stubLoadRegistryCache(t, fakeRegistryServers())

	cat, err := loadCatalog("all", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := cat.All()
	if len(entries) != 2 {
		t.Fatalf("expected 2 curated-only entries when registry disabled, got %d", len(entries))
	}

	for _, e := range entries {
		if e.Source != catalog.SourceCurated {
			t.Fatalf("expected curated source, got %q", e.Source)
		}
	}
}

func TestPrintCatalogEntriesFormatsOutput(t *testing.T) {
	entries := []catalog.Entry{
		{Source: catalog.SourceCurated, Name: "alpha", Curated: &service.Service{Name: "alpha", Description: "Alpha desc"}},
		{Source: catalog.SourceRegistry, Name: "beta", Registry: &registry.ServerResponse{Server: registry.ServerJSON{Name: "beta", Description: "Beta desc"}}},
	}

	var buf bytes.Buffer
	printCatalogEntries(&buf, entries)

	output := buf.String()

	if !strings.Contains(output, "Available services:") {
		t.Fatalf("expected heading in output, got %q", output)
	}

	if !strings.Contains(output, "alpha") {
		t.Fatalf("expected alpha in output, got %q", output)
	}

	if !strings.Contains(output, "beta [registry]") {
		t.Fatalf("expected beta [registry] tag in output, got %q", output)
	}

	if !strings.Contains(output, "Alpha desc") {
		t.Fatalf("expected Alpha desc in output, got %q", output)
	}

	if !strings.Contains(output, "Beta desc") {
		t.Fatalf("expected Beta desc in output, got %q", output)
	}
}

func TestPrintCatalogEntriesEmptyList(t *testing.T) {
	var buf bytes.Buffer
	printCatalogEntries(&buf, nil)

	output := buf.String()
	if !strings.Contains(output, "(none)") {
		t.Fatalf("expected empty state marker, got %q", output)
	}
}

func TestPrintCatalogEntriesSortsByName(t *testing.T) {
	entries := []catalog.Entry{
		{Source: catalog.SourceCurated, Name: "zeta", Curated: &service.Service{Name: "zeta", Description: "Zeta"}},
		{Source: catalog.SourceCurated, Name: "alpha", Curated: &service.Service{Name: "alpha", Description: "Alpha"}},
	}

	var buf bytes.Buffer
	printCatalogEntries(&buf, entries)

	output := buf.String()
	alphaIdx := strings.Index(output, "alpha")
	zetaIdx := strings.Index(output, "zeta")

	if alphaIdx > zetaIdx {
		t.Fatalf("expected entries sorted alphabetically, got %q", output)
	}
}

func TestCatalogEntryToServiceCurated(t *testing.T) {
	curated := service.Service{Name: "test", Description: "Test service"}
	entry := catalog.Entry{
		Source:  catalog.SourceCurated,
		Name:    "test",
		Curated: &curated,
	}

	svc, ok := catalogEntryToService(entry)
	if !ok {
		t.Fatal("expected curated entry to convert successfully")
	}

	if svc.Name != "test" {
		t.Fatalf("expected service name %q, got %q", "test", svc.Name)
	}
}

func TestCatalogEntryToServiceRegistry(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "test",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{Name: "test"},
		},
	}

	_, ok := catalogEntryToService(entry)
	if ok {
		t.Fatal("expected registry entry to return false")
	}
}

func TestValidateSourceAcceptsValidValues(t *testing.T) {
	for _, source := range []string{"curated", "registry", "all"} {
		if err := validateSource(source); err != nil {
			t.Fatalf("expected %q to be valid, got error: %v", source, err)
		}
	}
}

func TestValidateSourceRejectsInvalidValues(t *testing.T) {
	for _, source := range []string{"", "unknown", "local", "remote"} {
		err := validateSource(source)
		if err == nil {
			t.Fatalf("expected %q to be rejected", source)
		}

		if !strings.Contains(err.Error(), "invalid --source value") {
			t.Fatalf("expected validation error message, got %v", err)
		}
	}
}

func TestLoadCatalogEmptyRegistryCache(t *testing.T) {
	stubLoadServicesForCatalog(t)
	stubLoadRegistryCache(t, nil)

	cat, err := loadCatalog("all", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := cat.All()
	if len(entries) != 2 {
		t.Fatalf("expected 2 curated entries when cache is empty, got %d", len(entries))
	}
}
