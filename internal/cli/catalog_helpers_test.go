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
