package cli

import (
	"bytes"
	"errors"
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
	printCatalogEntries(&buf, entries, false)

	output := buf.String()

	if !strings.Contains(output, "Available services:") {
		t.Fatalf("expected heading in output, got %q", output)
	}

	if !strings.Contains(output, "alpha") {
		t.Fatalf("expected alpha in output, got %q", output)
	}

	if !strings.Contains(output, "beta") {
		t.Fatalf("expected beta in output, got %q", output)
	}

	if strings.Contains(output, "* ") {
		t.Fatalf("expected no markers when showMarkers=false, got %q", output)
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
	printCatalogEntries(&buf, nil, false)

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
	printCatalogEntries(&buf, entries, false)

	output := buf.String()
	alphaIdx := strings.Index(output, "alpha")
	zetaIdx := strings.Index(output, "zeta")

	if alphaIdx > zetaIdx {
		t.Fatalf("expected entries sorted alphabetically, got %q", output)
	}
}

func TestPrintCatalogEntriesWithMarkers(t *testing.T) {
	entries := []catalog.Entry{
		{Source: catalog.SourceCurated, Name: "alpha", Curated: &service.Service{Name: "alpha", Description: "Alpha desc"}},
		{Source: catalog.SourceRegistry, Name: "beta", Registry: &registry.ServerResponse{Server: registry.ServerJSON{Name: "beta", Description: "Beta desc"}}},
	}

	var buf bytes.Buffer
	printCatalogEntries(&buf, entries, true)

	output := buf.String()

	if !strings.Contains(output, "(* = curated by mcp-wire)") {
		t.Fatalf("expected legend line in output, got %q", output)
	}

	if !strings.Contains(output, "* alpha") {
		t.Fatalf("expected curated entry with * prefix, got %q", output)
	}

	if strings.Contains(output, "* beta") {
		t.Fatalf("expected registry entry without * prefix, got %q", output)
	}

	if !strings.Contains(output, "  beta") {
		t.Fatalf("expected registry entry with space prefix, got %q", output)
	}
}

func TestPrintCatalogEntriesMarkersOnlyCurated(t *testing.T) {
	entries := []catalog.Entry{
		{Source: catalog.SourceCurated, Name: "alpha", Curated: &service.Service{Name: "alpha", Description: "Alpha desc"}},
		{Source: catalog.SourceCurated, Name: "beta", Curated: &service.Service{Name: "beta", Description: "Beta desc"}},
	}

	var buf bytes.Buffer
	printCatalogEntries(&buf, entries, true)

	output := buf.String()

	if !strings.Contains(output, "(* = curated by mcp-wire)") {
		t.Fatalf("expected legend line in output, got %q", output)
	}

	if !strings.Contains(output, "* alpha") {
		t.Fatalf("expected alpha with * prefix, got %q", output)
	}

	if !strings.Contains(output, "* beta") {
		t.Fatalf("expected beta with * prefix, got %q", output)
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

func TestCatalogEntryToServiceRegistryNoRemotes(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "test",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{Name: "test"},
		},
	}

	_, ok := catalogEntryToService(entry)
	if ok {
		t.Fatal("expected registry entry without remotes to return false")
	}
}

func TestCatalogEntryToServiceRegistryRemote(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "test",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:        "test",
				Description: "Test server",
				Remotes: []registry.Transport{
					{Type: "streamable-http", URL: "https://example.com/mcp"},
				},
			},
		},
	}

	svc, ok := catalogEntryToService(entry)
	if !ok {
		t.Fatal("expected registry entry with remote to convert successfully")
	}

	if svc.Name != "test" {
		t.Fatalf("expected service name %q, got %q", "test", svc.Name)
	}

	if svc.Transport != "http" {
		t.Fatalf("expected transport %q, got %q", "http", svc.Transport)
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

func TestPrintRegistryTrustSummaryShowsSource(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "test-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:        "test-server",
				Description: "A test server",
				Version:     "1.0.0",
				Remotes: []registry.Transport{
					{Type: "sse", URL: "https://example.com/sse"},
				},
			},
		},
	}

	var buf bytes.Buffer
	printRegistryTrustSummary(&buf, entry)
	output := buf.String()

	if !strings.Contains(output, "Registry Service Information:") {
		t.Fatalf("expected header in output, got %q", output)
	}
	if !strings.Contains(output, "registry (community, not vetted by mcp-wire)") {
		t.Fatalf("expected source line in output, got %q", output)
	}
}

func TestPrintRegistryTrustSummaryShowsInstallType(t *testing.T) {
	entry := catalog.Entry{
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

	var buf bytes.Buffer
	printRegistryTrustSummary(&buf, entry)
	output := buf.String()

	if !strings.Contains(output, "Install:   remote") {
		t.Fatalf("expected install type in output, got %q", output)
	}
}

func TestPrintRegistryTrustSummaryShowsTransport(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "test-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:    "test-server",
				Version: "1.0.0",
				Remotes: []registry.Transport{
					{Type: "streamable-http", URL: "https://example.com/mcp"},
				},
			},
		},
	}

	var buf bytes.Buffer
	printRegistryTrustSummary(&buf, entry)
	output := buf.String()

	if !strings.Contains(output, "Transport: streamable-http") {
		t.Fatalf("expected transport in output, got %q", output)
	}
}

func TestPrintRegistryTrustSummaryShowsSecrets(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "test-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:    "test-server",
				Version: "1.0.0",
				Remotes: []registry.Transport{
					{
						Type: "sse",
						URL:  "https://example.com/sse",
						Headers: []registry.KeyValueInput{
							{Name: "Authorization", IsSecret: true, IsRequired: true},
							{Name: "X-API-Key", IsSecret: true, IsRequired: true},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	printRegistryTrustSummary(&buf, entry)
	output := buf.String()

	if !strings.Contains(output, "Secrets:   Authorization, X-API-Key") {
		t.Fatalf("expected secrets in output, got %q", output)
	}
}

func TestPrintRegistryTrustSummaryShowsRepoURL(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "test-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:    "test-server",
				Version: "1.0.0",
				Repository: &registry.Repository{
					URL: "https://github.com/example/repo",
				},
				Remotes: []registry.Transport{
					{Type: "sse", URL: "https://example.com/sse"},
				},
			},
		},
	}

	var buf bytes.Buffer
	printRegistryTrustSummary(&buf, entry)
	output := buf.String()

	if !strings.Contains(output, "Repo:      https://github.com/example/repo") {
		t.Fatalf("expected repo URL in output, got %q", output)
	}
}

func TestPrintRegistryTrustSummaryExcludesOptionalSecrets(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "test-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:    "test-server",
				Version: "1.0.0",
				Packages: []registry.Package{
					{
						RegistryType: "npm",
						Identifier:   "@example/server",
						Transport:    registry.Transport{Type: "stdio"},
						EnvironmentVariables: []registry.KeyValueInput{
							{Name: "REQ_TOKEN", IsRequired: true},
							{Name: "OPT_TOKEN", IsRequired: false},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	printRegistryTrustSummary(&buf, entry)
	output := buf.String()

	if !strings.Contains(output, "Secrets:   REQ_TOKEN") {
		t.Fatalf("expected required secret in output, got %q", output)
	}
	if strings.Contains(output, "OPT_TOKEN") {
		t.Fatalf("expected optional secret excluded, got %q", output)
	}
}

func TestPrintRegistryTrustSummaryOmitsSecretsWhenAllOptional(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "test-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:    "test-server",
				Version: "1.0.0",
				Packages: []registry.Package{
					{
						RegistryType: "npm",
						Identifier:   "@example/server",
						Transport:    registry.Transport{Type: "stdio"},
						EnvironmentVariables: []registry.KeyValueInput{
							{Name: "OPT_A", IsRequired: false},
							{Name: "OPT_B", IsRequired: false},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	printRegistryTrustSummary(&buf, entry)
	output := buf.String()

	if strings.Contains(output, "Secrets:") {
		t.Fatalf("expected no secrets line when all are optional, got %q", output)
	}
}

func TestPrintRegistryTrustSummaryOmitsEmptyFields(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "test-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:    "test-server",
				Version: "1.0.0",
			},
		},
	}

	var buf bytes.Buffer
	printRegistryTrustSummary(&buf, entry)
	output := buf.String()

	if strings.Contains(output, "Install:") {
		t.Fatalf("expected no install line for empty install type, got %q", output)
	}
	if strings.Contains(output, "Transport:") {
		t.Fatalf("expected no transport line for empty transport, got %q", output)
	}
	if strings.Contains(output, "Secrets:") {
		t.Fatalf("expected no secrets line for no env vars, got %q", output)
	}
	if strings.Contains(output, "Repo:") {
		t.Fatalf("expected no repo line for no repo URL, got %q", output)
	}
}

func TestPrintRegistryTrustSummaryPackageInstallType(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "test-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:    "test-server",
				Version: "1.0.0",
				Packages: []registry.Package{
					{
						RegistryType: "npm",
						Identifier:   "@example/server",
						Transport:    registry.Transport{Type: "stdio"},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	printRegistryTrustSummary(&buf, entry)
	output := buf.String()

	if !strings.Contains(output, "Install:   package") {
		t.Fatalf("expected package install type, got %q", output)
	}
	if !strings.Contains(output, "Transport: stdio") {
		t.Fatalf("expected stdio transport, got %q", output)
	}
}

func TestPrintRegistryTrustSummaryShowsPackageInfo(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "test-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:    "test-server",
				Version: "1.0.0",
				Packages: []registry.Package{
					{
						RegistryType: "npm",
						Identifier:   "@example/server",
						Version:      "2.0.0",
						Transport:    registry.Transport{Type: "stdio"},
						RuntimeHint:  "requires Node.js 18+",
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	printRegistryTrustSummary(&buf, entry)
	output := buf.String()

	if !strings.Contains(output, "Package:   npm (@example/server@2.0.0)") {
		t.Fatalf("expected package info in output, got %q", output)
	}

	if !strings.Contains(output, "Runtime:   requires Node.js 18+") {
		t.Fatalf("expected runtime hint in output, got %q", output)
	}
}

func TestPrintRegistryTrustSummaryOmitsRuntimeHintWhenEmpty(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "test-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:    "test-server",
				Version: "1.0.0",
				Packages: []registry.Package{
					{
						RegistryType: "npm",
						Identifier:   "@example/server",
						Transport:    registry.Transport{Type: "stdio"},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	printRegistryTrustSummary(&buf, entry)
	output := buf.String()

	if strings.Contains(output, "Runtime:") {
		t.Fatalf("expected no runtime line when hint is empty, got %q", output)
	}
}

func TestRegistryRemoteToServiceStreamableHTTP(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "test-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:        "test-server",
				Description: "A test server",
				Remotes: []registry.Transport{
					{Type: "streamable-http", URL: "https://example.com/mcp"},
				},
			},
		},
	}

	svc, ok := registryRemoteToService(entry)
	if !ok {
		t.Fatal("expected streamable-http remote to convert successfully")
	}

	if svc.Transport != "http" {
		t.Fatalf("expected transport %q, got %q", "http", svc.Transport)
	}

	if svc.URL != "https://example.com/mcp" {
		t.Fatalf("expected URL %q, got %q", "https://example.com/mcp", svc.URL)
	}

	if svc.Name != "test-server" {
		t.Fatalf("expected name %q, got %q", "test-server", svc.Name)
	}
}

func TestRegistryRemoteToServiceSSE(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "sse-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:        "sse-server",
				Description: "An SSE server",
				Remotes: []registry.Transport{
					{Type: "sse", URL: "https://example.com/sse"},
				},
			},
		},
	}

	svc, ok := registryRemoteToService(entry)
	if !ok {
		t.Fatal("expected sse remote to convert successfully")
	}

	if svc.Transport != "sse" {
		t.Fatalf("expected transport %q, got %q", "sse", svc.Transport)
	}

	if svc.URL != "https://example.com/sse" {
		t.Fatalf("expected URL %q, got %q", "https://example.com/sse", svc.URL)
	}
}

func TestRegistryRemoteToServiceNoRemotes(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "pkg-only",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "pkg-only",
				Packages: []registry.Package{
					{RegistryType: "npm", Identifier: "@example/server", Transport: registry.Transport{Type: "stdio"}},
				},
			},
		},
	}

	_, ok := registryRemoteToService(entry)
	if ok {
		t.Fatal("expected package-only entry to return false")
	}
}

func TestRegistryRemoteToServiceUnsupportedTransport(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "grpc-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "grpc-server",
				Remotes: []registry.Transport{
					{Type: "grpc", URL: "grpc://example.com:9090"},
				},
			},
		},
	}

	_, ok := registryRemoteToService(entry)
	if ok {
		t.Fatal("expected unsupported transport to return false")
	}
}

func TestRegistryRemoteToServiceURLVariables(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "tenant-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "tenant-server",
				Remotes: []registry.Transport{
					{
						Type: "streamable-http",
						URL:  "https://{tenant}.example.com/mcp",
						Variables: map[string]registry.InputField{
							"tenant": {Description: "Your tenant ID", IsRequired: true, Default: "default-tenant"},
						},
					},
				},
			},
		},
	}

	svc, ok := registryRemoteToService(entry)
	if !ok {
		t.Fatal("expected entry with URL variables to convert successfully")
	}

	if len(svc.Env) != 1 {
		t.Fatalf("expected 1 env var, got %d", len(svc.Env))
	}

	if svc.Env[0].Name != "tenant" {
		t.Fatalf("expected env var name %q, got %q", "tenant", svc.Env[0].Name)
	}

	if svc.Env[0].Default != "default-tenant" {
		t.Fatalf("expected default %q, got %q", "default-tenant", svc.Env[0].Default)
	}

	if !svc.Env[0].Required {
		t.Fatal("expected env var to be required")
	}
}

func TestRegistryRemoteToServiceHeaderVariables(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "header-var-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "header-var-server",
				Remotes: []registry.Transport{
					{
						Type: "sse",
						URL:  "https://example.com/sse",
						Headers: []registry.KeyValueInput{
							{
								Name:  "Authorization",
								Value: "Bearer {api_key}",
								Variables: map[string]registry.InputField{
									"api_key": {Description: "API key", IsRequired: true},
								},
							},
						},
					},
				},
			},
		},
	}

	svc, ok := registryRemoteToService(entry)
	if !ok {
		t.Fatal("expected entry with header variables to convert successfully")
	}

	if len(svc.Env) != 1 {
		t.Fatalf("expected 1 env var, got %d", len(svc.Env))
	}

	if svc.Env[0].Name != "api_key" {
		t.Fatalf("expected env var name %q, got %q", "api_key", svc.Env[0].Name)
	}

	if svc.Headers["Authorization"] != "Bearer {api_key}" {
		t.Fatalf("expected header template %q, got %q", "Bearer {api_key}", svc.Headers["Authorization"])
	}
}

func TestRegistryRemoteToServiceSecretHeaderNoTemplate(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "secret-header-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "secret-header-server",
				Remotes: []registry.Transport{
					{
						Type: "streamable-http",
						URL:  "https://example.com/mcp",
						Headers: []registry.KeyValueInput{
							{
								Name:        "X-API-Key",
								Description: "API key header",
								IsSecret:    true,
								IsRequired:  true,
							},
						},
					},
				},
			},
		},
	}

	svc, ok := registryRemoteToService(entry)
	if !ok {
		t.Fatal("expected entry with secret header to convert successfully")
	}

	if len(svc.Env) != 1 {
		t.Fatalf("expected 1 env var, got %d", len(svc.Env))
	}

	if svc.Env[0].Name != "X-API-Key" {
		t.Fatalf("expected env var name %q, got %q", "X-API-Key", svc.Env[0].Name)
	}

	if svc.Headers["X-API-Key"] != "{X-API-Key}" {
		t.Fatalf("expected header placeholder %q, got %q", "{X-API-Key}", svc.Headers["X-API-Key"])
	}
}

func TestRegistryPackageToServiceNpm(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "npm-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:        "npm-server",
				Description: "An npm MCP server",
				Packages: []registry.Package{
					{
						RegistryType: "npm",
						Identifier:   "@example/mcp-server",
						Version:      "1.2.3",
						Transport:    registry.Transport{Type: "stdio"},
					},
				},
			},
		},
	}

	svc, ok := registryPackageToService(entry)
	if !ok {
		t.Fatal("expected npm package to convert successfully")
	}

	if svc.Transport != "stdio" {
		t.Fatalf("expected transport %q, got %q", "stdio", svc.Transport)
	}

	if svc.Command != "npx" {
		t.Fatalf("expected command %q, got %q", "npx", svc.Command)
	}

	expectedArgs := []string{"-y", "@example/mcp-server@1.2.3"}
	if len(svc.Args) != len(expectedArgs) {
		t.Fatalf("expected args %v, got %v", expectedArgs, svc.Args)
	}

	for i, expected := range expectedArgs {
		if svc.Args[i] != expected {
			t.Fatalf("expected args[%d] = %q, got %q", i, expected, svc.Args[i])
		}
	}

	if svc.Name != "npm-server" {
		t.Fatalf("expected name %q, got %q", "npm-server", svc.Name)
	}
}

func TestRegistryPackageToServiceNpmNoVersion(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "npm-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "npm-server",
				Packages: []registry.Package{
					{
						RegistryType: "npm",
						Identifier:   "@example/mcp-server",
						Transport:    registry.Transport{Type: "stdio"},
					},
				},
			},
		},
	}

	svc, ok := registryPackageToService(entry)
	if !ok {
		t.Fatal("expected npm package without version to convert successfully")
	}

	expectedArgs := []string{"-y", "@example/mcp-server"}
	if len(svc.Args) != len(expectedArgs) {
		t.Fatalf("expected args %v, got %v", expectedArgs, svc.Args)
	}

	for i, expected := range expectedArgs {
		if svc.Args[i] != expected {
			t.Fatalf("expected args[%d] = %q, got %q", i, expected, svc.Args[i])
		}
	}
}

func TestRegistryPackageToServicePypi(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "pypi-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "pypi-server",
				Packages: []registry.Package{
					{
						RegistryType: "pypi",
						Identifier:   "mcp-server-example",
						Version:      "0.5.0",
						Transport:    registry.Transport{Type: "stdio"},
					},
				},
			},
		},
	}

	svc, ok := registryPackageToService(entry)
	if !ok {
		t.Fatal("expected pypi package to convert successfully")
	}

	if svc.Command != "uvx" {
		t.Fatalf("expected command %q, got %q", "uvx", svc.Command)
	}

	expectedArgs := []string{"mcp-server-example@0.5.0"}
	if len(svc.Args) != len(expectedArgs) {
		t.Fatalf("expected args %v, got %v", expectedArgs, svc.Args)
	}

	if svc.Args[0] != expectedArgs[0] {
		t.Fatalf("expected args[0] = %q, got %q", expectedArgs[0], svc.Args[0])
	}
}

func TestRegistryPackageToServiceDocker(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "docker-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "docker-server",
				Packages: []registry.Package{
					{
						RegistryType: "docker",
						Identifier:   "ghcr.io/example/mcp-server",
						Version:      "latest",
						Transport:    registry.Transport{Type: "stdio"},
					},
				},
			},
		},
	}

	svc, ok := registryPackageToService(entry)
	if !ok {
		t.Fatal("expected docker package to convert successfully")
	}

	if svc.Command != "docker" {
		t.Fatalf("expected command %q, got %q", "docker", svc.Command)
	}

	expectedArgs := []string{"run", "-i", "--rm", "ghcr.io/example/mcp-server:latest"}
	if len(svc.Args) != len(expectedArgs) {
		t.Fatalf("expected args %v, got %v", expectedArgs, svc.Args)
	}

	for i, expected := range expectedArgs {
		if svc.Args[i] != expected {
			t.Fatalf("expected args[%d] = %q, got %q", i, expected, svc.Args[i])
		}
	}
}

func TestRegistryPackageToServiceOCI(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "oci-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "oci-server",
				Packages: []registry.Package{
					{
						RegistryType: "oci",
						Identifier:   "ghcr.io/example/mcp-server",
						Transport:    registry.Transport{Type: "stdio"},
					},
				},
			},
		},
	}

	svc, ok := registryPackageToService(entry)
	if !ok {
		t.Fatal("expected oci package to convert successfully")
	}

	if svc.Command != "docker" {
		t.Fatalf("expected command %q, got %q", "docker", svc.Command)
	}

	if svc.Args[0] != "run" {
		t.Fatalf("expected args[0] = %q, got %q", "run", svc.Args[0])
	}
}

func TestRegistryPackageToServiceNuget(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "nuget-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "nuget-server",
				Packages: []registry.Package{
					{
						RegistryType: "nuget",
						Identifier:   "Example.McpServer",
						Transport:    registry.Transport{Type: "stdio"},
					},
				},
			},
		},
	}

	svc, ok := registryPackageToService(entry)
	if !ok {
		t.Fatal("expected nuget package to convert successfully")
	}

	if svc.Command != "dotnet" {
		t.Fatalf("expected command %q, got %q", "dotnet", svc.Command)
	}

	expectedArgs := []string{"tool", "run", "Example.McpServer"}
	if len(svc.Args) != len(expectedArgs) {
		t.Fatalf("expected args %v, got %v", expectedArgs, svc.Args)
	}

	for i, expected := range expectedArgs {
		if svc.Args[i] != expected {
			t.Fatalf("expected args[%d] = %q, got %q", i, expected, svc.Args[i])
		}
	}
}

func TestRegistryPackageToServiceMcpb(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "mcpb-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "mcpb-server",
				Packages: []registry.Package{
					{
						RegistryType: "mcpb",
						Identifier:   "example-server",
						Transport:    registry.Transport{Type: "stdio"},
					},
				},
			},
		},
	}

	svc, ok := registryPackageToService(entry)
	if !ok {
		t.Fatal("expected mcpb package to convert successfully")
	}

	if svc.Command != "mcpb" {
		t.Fatalf("expected command %q, got %q", "mcpb", svc.Command)
	}

	expectedArgs := []string{"run", "example-server"}
	if len(svc.Args) != len(expectedArgs) {
		t.Fatalf("expected args %v, got %v", expectedArgs, svc.Args)
	}

	for i, expected := range expectedArgs {
		if svc.Args[i] != expected {
			t.Fatalf("expected args[%d] = %q, got %q", i, expected, svc.Args[i])
		}
	}
}

func TestRegistryPackageToServiceNoPackages(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "empty",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{Name: "empty"},
		},
	}

	_, ok := registryPackageToService(entry)
	if ok {
		t.Fatal("expected entry without packages to return false")
	}
}

func TestRegistryPackageToServiceEmptyIdentifier(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "bad-pkg",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "bad-pkg",
				Packages: []registry.Package{
					{
						RegistryType: "npm",
						Identifier:   "",
						Transport:    registry.Transport{Type: "stdio"},
					},
				},
			},
		},
	}

	_, ok := registryPackageToService(entry)
	if ok {
		t.Fatal("expected empty identifier to return false")
	}
}

func TestRegistryPackageToServiceUnsupportedType(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "unknown-pkg",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "unknown-pkg",
				Packages: []registry.Package{
					{
						RegistryType: "brew",
						Identifier:   "some-package",
						Transport:    registry.Transport{Type: "stdio"},
					},
				},
			},
		},
	}

	_, ok := registryPackageToService(entry)
	if ok {
		t.Fatal("expected unsupported registry type to return false")
	}
}

func TestRegistryPackageToServiceWithEnvVars(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "env-pkg",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "env-pkg",
				Packages: []registry.Package{
					{
						RegistryType: "npm",
						Identifier:   "@example/server",
						Transport:    registry.Transport{Type: "stdio"},
						EnvironmentVariables: []registry.KeyValueInput{
							{Name: "API_TOKEN", Description: "API token", IsRequired: true},
							{Name: "OPTIONAL_VAR", Description: "Optional var", IsRequired: false},
						},
					},
				},
			},
		},
	}

	svc, ok := registryPackageToService(entry)
	if !ok {
		t.Fatal("expected package with env vars to convert successfully")
	}

	if len(svc.Env) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(svc.Env))
	}

	if svc.Env[0].Name != "API_TOKEN" {
		t.Fatalf("expected first env var %q, got %q", "API_TOKEN", svc.Env[0].Name)
	}

	if !svc.Env[0].Required {
		t.Fatal("expected API_TOKEN to be required")
	}

	if svc.Env[1].Name != "OPTIONAL_VAR" {
		t.Fatalf("expected second env var %q, got %q", "OPTIONAL_VAR", svc.Env[1].Name)
	}

	if svc.Env[1].Required {
		t.Fatal("expected OPTIONAL_VAR to not be required")
	}
}

func TestRegistryPackageToServiceWithRuntimeArgs(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "args-pkg",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "args-pkg",
				Packages: []registry.Package{
					{
						RegistryType: "npm",
						Identifier:   "@example/server",
						Transport:    registry.Transport{Type: "stdio"},
						RuntimeArguments: []registry.Argument{
							{Value: "--stdio"},
							{Value: "--verbose"},
						},
					},
				},
			},
		},
	}

	svc, ok := registryPackageToService(entry)
	if !ok {
		t.Fatal("expected package with runtime args to convert successfully")
	}

	expectedArgs := []string{"-y", "@example/server", "--stdio", "--verbose"}
	if len(svc.Args) != len(expectedArgs) {
		t.Fatalf("expected args %v, got %v", expectedArgs, svc.Args)
	}

	for i, expected := range expectedArgs {
		if svc.Args[i] != expected {
			t.Fatalf("expected args[%d] = %q, got %q", i, expected, svc.Args[i])
		}
	}
}

func TestRegistryPackageToServiceRuntimeArgInputCreatesEnvVar(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "input-args-pkg",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "input-args-pkg",
				Packages: []registry.Package{
					{
						RegistryType: "npm",
						Identifier:   "@example/server",
						Transport:    registry.Transport{Type: "stdio"},
						RuntimeArguments: []registry.Argument{
							{Name: "directory", Description: "Directory to serve", IsRequired: true},
						},
					},
				},
			},
		},
	}

	svc, ok := registryPackageToService(entry)
	if !ok {
		t.Fatal("expected package with input runtime args to convert successfully")
	}

	if len(svc.Env) != 1 {
		t.Fatalf("expected 1 env var from runtime arg, got %d", len(svc.Env))
	}

	if svc.Env[0].Name != "directory" {
		t.Fatalf("expected env var name %q, got %q", "directory", svc.Env[0].Name)
	}

	// Check that the placeholder is in the args
	found := false
	for _, arg := range svc.Args {
		if arg == "{directory}" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected {directory} placeholder in args, got %v", svc.Args)
	}
}

func TestCatalogEntryToServiceRegistryPackageFallback(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "pkg-only",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:        "pkg-only",
				Description: "Package-only server",
				Packages: []registry.Package{
					{
						RegistryType: "npm",
						Identifier:   "@example/server",
						Transport:    registry.Transport{Type: "stdio"},
					},
				},
			},
		},
	}

	svc, ok := catalogEntryToService(entry)
	if !ok {
		t.Fatal("expected package-only entry to convert via fallback")
	}

	if svc.Transport != "stdio" {
		t.Fatalf("expected transport %q, got %q", "stdio", svc.Transport)
	}

	if svc.Command != "npx" {
		t.Fatalf("expected command %q, got %q", "npx", svc.Command)
	}
}

func TestCatalogEntryToServicePrefersRemoteOverPackage(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "dual-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "dual-server",
				Remotes: []registry.Transport{
					{Type: "sse", URL: "https://example.com/sse"},
				},
				Packages: []registry.Package{
					{
						RegistryType: "npm",
						Identifier:   "@example/server",
						Transport:    registry.Transport{Type: "stdio"},
					},
				},
			},
		},
	}

	svc, ok := catalogEntryToService(entry)
	if !ok {
		t.Fatal("expected dual-method entry to convert successfully")
	}

	if svc.Transport != "sse" {
		t.Fatalf("expected remote transport %q to take precedence, got %q", "sse", svc.Transport)
	}
}

func TestResolvePackageArgumentsLiterals(t *testing.T) {
	args := []registry.Argument{
		{Value: "--stdio"},
		{Value: "--port"},
		{Value: "3000"},
	}

	result := resolvePackageArguments(args, nil)

	expected := []string{"--stdio", "--port", "3000"}
	if len(result) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}

	for i, exp := range expected {
		if result[i] != exp {
			t.Fatalf("expected result[%d] = %q, got %q", i, exp, result[i])
		}
	}
}

func TestResolvePackageArgumentsInputCreatesPlaceholder(t *testing.T) {
	var envVars []service.EnvVar
	seen := map[string]int{}

	addVar := func(name, desc, defaultVal string, required bool) {
		if _, exists := seen[name]; exists {
			return
		}
		seen[name] = len(envVars)
		envVars = append(envVars, service.EnvVar{
			Name:        name,
			Description: desc,
			Required:    required,
			Default:     defaultVal,
		})
	}

	args := []registry.Argument{
		{Value: "--stdio"},
		{Name: "path", Description: "Directory path", IsRequired: true, Default: "/tmp"},
	}

	result := resolvePackageArguments(args, addVar)

	if len(result) != 2 {
		t.Fatalf("expected 2 args, got %v", result)
	}

	if result[0] != "--stdio" {
		t.Fatalf("expected first arg %q, got %q", "--stdio", result[0])
	}

	if result[1] != "{path}" {
		t.Fatalf("expected second arg %q, got %q", "{path}", result[1])
	}

	if len(envVars) != 1 {
		t.Fatalf("expected 1 env var, got %d", len(envVars))
	}

	if envVars[0].Name != "path" {
		t.Fatalf("expected env var name %q, got %q", "path", envVars[0].Name)
	}

	if envVars[0].Default != "/tmp" {
		t.Fatalf("expected env var default %q, got %q", "/tmp", envVars[0].Default)
	}
}

func TestResolvePackageArgumentsNilAddVarSkipsInputArgs(t *testing.T) {
	args := []registry.Argument{
		{Value: "--stdio"},
		{Name: "path", Description: "Directory path", IsRequired: true},
	}

	result := resolvePackageArguments(args, nil)

	if len(result) != 1 {
		t.Fatalf("expected 1 arg (input arg skipped with nil addVar), got %v", result)
	}

	if result[0] != "--stdio" {
		t.Fatalf("expected %q, got %q", "--stdio", result[0])
	}
}

func TestRegistryPackageToServiceSkipsUnsupportedFirstPackage(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "multi-pkg",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "multi-pkg",
				Packages: []registry.Package{
					{
						RegistryType: "brew",
						Identifier:   "unsupported-package",
						Transport:    registry.Transport{Type: "stdio"},
					},
					{
						RegistryType: "npm",
						Identifier:   "@example/server",
						Version:      "1.0.0",
						Transport:    registry.Transport{Type: "stdio"},
					},
				},
			},
		},
	}

	svc, ok := registryPackageToService(entry)
	if !ok {
		t.Fatal("expected fallback to second (npm) package")
	}

	if svc.Command != "npx" {
		t.Fatalf("expected command %q, got %q", "npx", svc.Command)
	}

	if len(svc.Args) < 2 || svc.Args[1] != "@example/server@1.0.0" {
		t.Fatalf("expected npm identifier in args, got %v", svc.Args)
	}
}

func TestRegistryPackageToServiceAllPackagesUnsupported(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "unsupported-all",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "unsupported-all",
				Packages: []registry.Package{
					{RegistryType: "brew", Identifier: "pkg1", Transport: registry.Transport{Type: "stdio"}},
					{RegistryType: "snap", Identifier: "pkg2", Transport: registry.Transport{Type: "stdio"}},
				},
			},
		},
	}

	_, ok := registryPackageToService(entry)
	if ok {
		t.Fatal("expected all-unsupported packages to return false")
	}
}

func TestRegistryPackageToServicePackageArgInputCreatesPlaceholder(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "pkg-arg-input",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "pkg-arg-input",
				Packages: []registry.Package{
					{
						RegistryType: "npm",
						Identifier:   "@example/server",
						Transport:    registry.Transport{Type: "stdio"},
						PackageArguments: []registry.Argument{
							{Value: "--registry"},
							{Name: "registry_url", Description: "Custom registry URL", IsRequired: true},
						},
					},
				},
			},
		},
	}

	svc, ok := registryPackageToService(entry)
	if !ok {
		t.Fatal("expected package with input package args to convert")
	}

	// Expect: npx -y --registry {registry_url} @example/server
	foundRegistry := false
	foundPlaceholder := false
	for _, arg := range svc.Args {
		if arg == "--registry" {
			foundRegistry = true
		}
		if arg == "{registry_url}" {
			foundPlaceholder = true
		}
	}

	if !foundRegistry {
		t.Fatalf("expected --registry in args, got %v", svc.Args)
	}

	if !foundPlaceholder {
		t.Fatalf("expected {registry_url} placeholder in args, got %v", svc.Args)
	}

	// Verify the env var was created
	envFound := false
	for _, ev := range svc.Env {
		if ev.Name == "registry_url" {
			envFound = true
			if !ev.Required {
				t.Fatal("expected registry_url env var to be required")
			}
		}
	}

	if !envFound {
		t.Fatal("expected registry_url env var to be registered")
	}
}

func TestRefreshRegistryEntryUpdatesOnSuccess(t *testing.T) {
	original := fetchServerLatest
	t.Cleanup(func() { fetchServerLatest = original })

	fetchServerLatest = func(serverName string) (*registry.ServerResponse, error) {
		return &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:        serverName,
				Description: "Updated description",
				Packages: []registry.Package{
					{
						RegistryType: "npm",
						Identifier:   "@example/updated",
						Transport:    registry.Transport{Type: "stdio"},
					},
				},
			},
		}, nil
	}

	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "test-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:        "test-server",
				Description: "Cached description",
			},
		},
	}

	refreshed := refreshRegistryEntry(entry)

	if refreshed.Registry.Server.Description != "Updated description" {
		t.Fatalf("expected refreshed description, got %q", refreshed.Registry.Server.Description)
	}

	if len(refreshed.Registry.Server.Packages) != 1 {
		t.Fatal("expected refreshed entry to have packages")
	}
}

func TestRefreshRegistryEntryFallsBackOnError(t *testing.T) {
	original := fetchServerLatest
	t.Cleanup(func() { fetchServerLatest = original })

	fetchServerLatest = func(_ string) (*registry.ServerResponse, error) {
		return nil, errors.New("network error")
	}

	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "test-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name:        "test-server",
				Description: "Cached description",
			},
		},
	}

	refreshed := refreshRegistryEntry(entry)

	if refreshed.Registry.Server.Description != "Cached description" {
		t.Fatalf("expected original entry on error, got %q", refreshed.Registry.Server.Description)
	}
}

func TestRefreshRegistryEntrySkipsCurated(t *testing.T) {
	original := fetchServerLatest
	t.Cleanup(func() { fetchServerLatest = original })

	called := false
	fetchServerLatest = func(_ string) (*registry.ServerResponse, error) {
		called = true
		return nil, nil
	}

	curated := service.Service{Name: "test", Description: "curated"}
	entry := catalog.Entry{
		Source:  catalog.SourceCurated,
		Name:    "test",
		Curated: &curated,
	}

	refreshed := refreshRegistryEntry(entry)

	if called {
		t.Fatal("expected refresh to skip curated entries")
	}

	if refreshed.Source != catalog.SourceCurated {
		t.Fatal("expected curated entry returned unchanged")
	}
}

func TestRegistryRemoteToServiceMergesRequiredOnDuplicate(t *testing.T) {
	entry := catalog.Entry{
		Source: catalog.SourceRegistry,
		Name:   "merge-server",
		Registry: &registry.ServerResponse{
			Server: registry.ServerJSON{
				Name: "merge-server",
				Remotes: []registry.Transport{
					{
						Type: "streamable-http",
						URL:  "https://{api_key}.example.com/mcp",
						Variables: map[string]registry.InputField{
							"api_key": {Description: "URL key", IsRequired: false},
						},
						Headers: []registry.KeyValueInput{
							{
								Name:  "Authorization",
								Value: "Bearer {api_key}",
								Variables: map[string]registry.InputField{
									"api_key": {Description: "Auth key", IsRequired: true},
								},
							},
						},
					},
				},
			},
		},
	}

	svc, ok := registryRemoteToService(entry)
	if !ok {
		t.Fatal("expected entry to convert successfully")
	}

	if len(svc.Env) != 1 {
		t.Fatalf("expected 1 deduplicated env var, got %d", len(svc.Env))
	}

	if !svc.Env[0].Required {
		t.Fatal("expected required to be merged with OR (true wins)")
	}

	if svc.Env[0].Name != "api_key" {
		t.Fatalf("expected env var name %q, got %q", "api_key", svc.Env[0].Name)
	}
}
