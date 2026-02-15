package catalog

import (
	"testing"

	"github.com/andreagrandi/mcp-wire/internal/registry"
	"github.com/andreagrandi/mcp-wire/internal/service"
)

func sampleService(name, desc string) service.Service {
	return service.Service{
		Name:        name,
		Description: desc,
		Transport:   "sse",
		URL:         "https://example.com/sse",
		Env: []service.EnvVar{
			{Name: "TOKEN", Description: "API token", Required: true},
		},
	}
}

func sampleRegistryServer(name, title, desc string) registry.ServerResponse {
	return registry.ServerResponse{
		Server: registry.ServerJSON{
			Name:        name,
			Title:       title,
			Description: desc,
			Version:     "1.0.0",
			Remotes: []registry.Transport{
				{
					Type: "sse",
					URL:  "https://example.com/sse",
				},
			},
		},
	}
}

func TestSourceConstants(t *testing.T) {
	if SourceCurated != "curated" {
		t.Fatalf("expected SourceCurated=%q, got %q", "curated", SourceCurated)
	}
	if SourceRegistry != "registry" {
		t.Fatalf("expected SourceRegistry=%q, got %q", "registry", SourceRegistry)
	}
}

func TestFromCurated(t *testing.T) {
	svc := sampleService("sentry", "Error tracking")
	entry := FromCurated(svc)

	if entry.Source != SourceCurated {
		t.Fatalf("expected source=%q, got %q", SourceCurated, entry.Source)
	}
	if entry.Name != "sentry" {
		t.Fatalf("expected name=%q, got %q", "sentry", entry.Name)
	}
	if entry.Curated == nil {
		t.Fatal("expected Curated pointer to be set")
	}
	if entry.Registry != nil {
		t.Fatal("expected Registry pointer to be nil")
	}
}

func TestFromCuratedMap(t *testing.T) {
	services := map[string]service.Service{
		"sentry": sampleService("sentry", "Error tracking"),
		"jira":   sampleService("jira", "Project management"),
	}
	entries := FromCuratedMap(services)

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Source != SourceCurated {
			t.Fatalf("expected source=%q, got %q", SourceCurated, e.Source)
		}
		if e.Curated == nil {
			t.Fatal("expected Curated pointer to be set")
		}
	}
}

func TestFromRegistry(t *testing.T) {
	resp := sampleRegistryServer("ns/sentry", "Sentry", "Error tracking")
	entry := FromRegistry(resp)

	if entry.Source != SourceRegistry {
		t.Fatalf("expected source=%q, got %q", SourceRegistry, entry.Source)
	}
	if entry.Name != "ns/sentry" {
		t.Fatalf("expected name=%q, got %q", "ns/sentry", entry.Name)
	}
	if entry.Registry == nil {
		t.Fatal("expected Registry pointer to be set")
	}
	if entry.Curated != nil {
		t.Fatal("expected Curated pointer to be nil")
	}
}

func TestFromRegistrySlice(t *testing.T) {
	servers := []registry.ServerResponse{
		sampleRegistryServer("ns/a", "A", "Server A"),
		sampleRegistryServer("ns/b", "B", "Server B"),
	}
	entries := FromRegistrySlice(servers)

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Source != SourceRegistry {
			t.Fatalf("expected source=%q, got %q", SourceRegistry, e.Source)
		}
	}
}

func TestDisplayNameCurated(t *testing.T) {
	entry := FromCurated(sampleService("sentry", "Error tracking"))
	if entry.DisplayName() != "sentry" {
		t.Fatalf("expected display name=%q, got %q", "sentry", entry.DisplayName())
	}
}

func TestDisplayNameRegistryTitle(t *testing.T) {
	entry := FromRegistry(sampleRegistryServer("ns/sentry", "Sentry Integration", "Error tracking"))
	if entry.DisplayName() != "Sentry Integration" {
		t.Fatalf("expected display name=%q, got %q", "Sentry Integration", entry.DisplayName())
	}
}

func TestDisplayNameRegistryFallbackToName(t *testing.T) {
	resp := registry.ServerResponse{
		Server: registry.ServerJSON{
			Name:    "ns/sentry",
			Version: "1.0.0",
		},
	}
	entry := FromRegistry(resp)
	if entry.DisplayName() != "ns/sentry" {
		t.Fatalf("expected display name=%q, got %q", "ns/sentry", entry.DisplayName())
	}
}

func TestDescriptionCurated(t *testing.T) {
	entry := FromCurated(sampleService("sentry", "Error tracking"))
	if entry.Description() != "Error tracking" {
		t.Fatalf("expected description=%q, got %q", "Error tracking", entry.Description())
	}
}

func TestDescriptionRegistry(t *testing.T) {
	entry := FromRegistry(sampleRegistryServer("ns/sentry", "Sentry", "Error tracking"))
	if entry.Description() != "Error tracking" {
		t.Fatalf("expected description=%q, got %q", "Error tracking", entry.Description())
	}
}

func TestEnvVarsCurated(t *testing.T) {
	svc := sampleService("sentry", "Error tracking")
	entry := FromCurated(svc)
	vars := entry.EnvVars()

	if len(vars) != 1 {
		t.Fatalf("expected 1 env var, got %d", len(vars))
	}
	if vars[0].Name != "TOKEN" {
		t.Fatalf("expected env var name=%q, got %q", "TOKEN", vars[0].Name)
	}
}

func TestEnvVarsRegistryPackages(t *testing.T) {
	resp := registry.ServerResponse{
		Server: registry.ServerJSON{
			Name:    "ns/sentry",
			Version: "1.0.0",
			Packages: []registry.Package{
				{
					EnvironmentVariables: []registry.KeyValueInput{
						{Name: "API_KEY", Description: "API key", IsRequired: true},
						{Name: "ORG_ID", Description: "Organization", IsRequired: false},
					},
				},
			},
		},
	}
	entry := FromRegistry(resp)
	vars := entry.EnvVars()

	if len(vars) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(vars))
	}
	if vars[0].Name != "API_KEY" {
		t.Fatalf("expected first env var=%q, got %q", "API_KEY", vars[0].Name)
	}
	if !vars[0].Required {
		t.Fatal("expected API_KEY to be required")
	}
	if vars[1].Name != "ORG_ID" {
		t.Fatalf("expected second env var=%q, got %q", "ORG_ID", vars[1].Name)
	}
}

func TestEnvVarsRegistryRemoteSecretHeaders(t *testing.T) {
	resp := registry.ServerResponse{
		Server: registry.ServerJSON{
			Name:    "ns/sentry",
			Version: "1.0.0",
			Remotes: []registry.Transport{
				{
					Type: "sse",
					URL:  "https://example.com/sse",
					Headers: []registry.KeyValueInput{
						{Name: "Authorization", Description: "Auth header", IsSecret: true, IsRequired: true},
						{Name: "X-Custom", Description: "Non-secret header", IsSecret: false},
					},
				},
			},
		},
	}
	entry := FromRegistry(resp)
	vars := entry.EnvVars()

	if len(vars) != 1 {
		t.Fatalf("expected 1 env var (only secret headers), got %d", len(vars))
	}
	if vars[0].Name != "Authorization" {
		t.Fatalf("expected env var=%q, got %q", "Authorization", vars[0].Name)
	}
}

func TestEnvVarsRegistryDeduplication(t *testing.T) {
	resp := registry.ServerResponse{
		Server: registry.ServerJSON{
			Name:    "ns/sentry",
			Version: "1.0.0",
			Packages: []registry.Package{
				{
					EnvironmentVariables: []registry.KeyValueInput{
						{Name: "API_KEY", Description: "From package 1"},
					},
				},
				{
					EnvironmentVariables: []registry.KeyValueInput{
						{Name: "API_KEY", Description: "From package 2"},
					},
				},
			},
		},
	}
	entry := FromRegistry(resp)
	vars := entry.EnvVars()

	if len(vars) != 1 {
		t.Fatalf("expected 1 env var (deduplicated), got %d", len(vars))
	}
}

func TestEnvVarsRegistryDeduplicationMergesRequired(t *testing.T) {
	resp := registry.ServerResponse{
		Server: registry.ServerJSON{
			Name:    "ns/sentry",
			Version: "1.0.0",
			Packages: []registry.Package{
				{
					EnvironmentVariables: []registry.KeyValueInput{
						{Name: "TOKEN", Description: "", IsRequired: false},
					},
				},
			},
			Remotes: []registry.Transport{
				{
					Type: "sse",
					URL:  "https://example.com/sse",
					Headers: []registry.KeyValueInput{
						{Name: "TOKEN", Description: "Auth token", IsSecret: true, IsRequired: true},
					},
				},
			},
		},
	}
	entry := FromRegistry(resp)
	vars := entry.EnvVars()

	if len(vars) != 1 {
		t.Fatalf("expected 1 env var (deduplicated), got %d", len(vars))
	}
	if !vars[0].Required {
		t.Fatal("expected Required=true after merging optional+required duplicate")
	}
	if vars[0].Description != "Auth token" {
		t.Fatalf("expected description backfilled from remote, got %q", vars[0].Description)
	}
}

func TestEnvVarsRegistrySkipsEmptyName(t *testing.T) {
	resp := registry.ServerResponse{
		Server: registry.ServerJSON{
			Name:    "ns/sentry",
			Version: "1.0.0",
			Packages: []registry.Package{
				{
					EnvironmentVariables: []registry.KeyValueInput{
						{Name: "", Description: "No name"},
						{Name: "VALID", Description: "Has name"},
					},
				},
			},
		},
	}
	entry := FromRegistry(resp)
	vars := entry.EnvVars()

	if len(vars) != 1 {
		t.Fatalf("expected 1 env var (skipped empty), got %d", len(vars))
	}
	if vars[0].Name != "VALID" {
		t.Fatalf("expected env var=%q, got %q", "VALID", vars[0].Name)
	}
}

func TestTransportCurated(t *testing.T) {
	svc := sampleService("sentry", "Error tracking")
	entry := FromCurated(svc)
	if entry.Transport() != "sse" {
		t.Fatalf("expected transport=%q, got %q", "sse", entry.Transport())
	}
}

func TestTransportRegistryRemote(t *testing.T) {
	entry := FromRegistry(sampleRegistryServer("ns/sentry", "Sentry", "Error tracking"))
	if entry.Transport() != "sse" {
		t.Fatalf("expected transport=%q, got %q", "sse", entry.Transport())
	}
}

func TestTransportRegistryPackageFallback(t *testing.T) {
	resp := registry.ServerResponse{
		Server: registry.ServerJSON{
			Name:    "ns/sentry",
			Version: "1.0.0",
			Packages: []registry.Package{
				{
					Transport: registry.Transport{Type: "stdio"},
				},
			},
		},
	}
	entry := FromRegistry(resp)
	if entry.Transport() != "stdio" {
		t.Fatalf("expected transport=%q, got %q", "stdio", entry.Transport())
	}
}

func TestTransportEmpty(t *testing.T) {
	resp := registry.ServerResponse{
		Server: registry.ServerJSON{
			Name:    "ns/empty",
			Version: "1.0.0",
		},
	}
	entry := FromRegistry(resp)
	if entry.Transport() != "" {
		t.Fatalf("expected empty transport, got %q", entry.Transport())
	}
}

func TestHasRemotesCuratedSSE(t *testing.T) {
	svc := service.Service{Name: "test", Transport: "sse"}
	entry := FromCurated(svc)
	if !entry.HasRemotes() {
		t.Fatal("expected HasRemotes=true for curated sse")
	}
}

func TestHasRemotesCuratedHTTP(t *testing.T) {
	svc := service.Service{Name: "test", Transport: "http"}
	entry := FromCurated(svc)
	if !entry.HasRemotes() {
		t.Fatal("expected HasRemotes=true for curated http")
	}
}

func TestHasRemotesCuratedStdio(t *testing.T) {
	svc := service.Service{Name: "test", Transport: "stdio"}
	entry := FromCurated(svc)
	if entry.HasRemotes() {
		t.Fatal("expected HasRemotes=false for curated stdio")
	}
}

func TestHasRemotesRegistry(t *testing.T) {
	entry := FromRegistry(sampleRegistryServer("ns/a", "A", "A"))
	if !entry.HasRemotes() {
		t.Fatal("expected HasRemotes=true for registry with remotes")
	}
}

func TestHasRemotesRegistryNoRemotes(t *testing.T) {
	resp := registry.ServerResponse{
		Server: registry.ServerJSON{Name: "ns/a", Version: "1.0.0"},
	}
	entry := FromRegistry(resp)
	if entry.HasRemotes() {
		t.Fatal("expected HasRemotes=false for registry without remotes")
	}
}

func TestHasPackages(t *testing.T) {
	resp := registry.ServerResponse{
		Server: registry.ServerJSON{
			Name:    "ns/a",
			Version: "1.0.0",
			Packages: []registry.Package{
				{RegistryType: "npm", Identifier: "@example/server"},
			},
		},
	}
	entry := FromRegistry(resp)
	if !entry.HasPackages() {
		t.Fatal("expected HasPackages=true")
	}
}

func TestHasPackagesCurated(t *testing.T) {
	entry := FromCurated(sampleService("test", "test"))
	if entry.HasPackages() {
		t.Fatal("expected HasPackages=false for curated entry")
	}
}

func TestRepositoryURL(t *testing.T) {
	resp := registry.ServerResponse{
		Server: registry.ServerJSON{
			Name:    "ns/a",
			Version: "1.0.0",
			Repository: &registry.Repository{
				URL: "https://github.com/example/repo",
			},
		},
	}
	entry := FromRegistry(resp)
	if entry.RepositoryURL() != "https://github.com/example/repo" {
		t.Fatalf("expected repo URL, got %q", entry.RepositoryURL())
	}
}

func TestRepositoryURLEmpty(t *testing.T) {
	entry := FromCurated(sampleService("test", "test"))
	if entry.RepositoryURL() != "" {
		t.Fatalf("expected empty repo URL, got %q", entry.RepositoryURL())
	}
}

func TestWebsiteURL(t *testing.T) {
	resp := registry.ServerResponse{
		Server: registry.ServerJSON{
			Name:       "ns/a",
			Version:    "1.0.0",
			WebsiteURL: "https://example.com",
		},
	}
	entry := FromRegistry(resp)
	if entry.WebsiteURL() != "https://example.com" {
		t.Fatalf("expected website URL, got %q", entry.WebsiteURL())
	}
}

func TestWebsiteURLEmpty(t *testing.T) {
	entry := FromCurated(sampleService("test", "test"))
	if entry.WebsiteURL() != "" {
		t.Fatalf("expected empty website URL, got %q", entry.WebsiteURL())
	}
}

func TestMergeCuratedOnly(t *testing.T) {
	curated := []Entry{FromCurated(sampleService("sentry", "Error tracking"))}
	cat := Merge(curated, nil)

	if cat.Count() != 1 {
		t.Fatalf("expected 1 entry, got %d", cat.Count())
	}
	if cat.All()[0].Source != SourceCurated {
		t.Fatalf("expected source=%q, got %q", SourceCurated, cat.All()[0].Source)
	}
}

func TestMergeRegistryOnly(t *testing.T) {
	reg := []Entry{FromRegistry(sampleRegistryServer("ns/sentry", "Sentry", "Error tracking"))}
	cat := Merge(nil, reg)

	if cat.Count() != 1 {
		t.Fatalf("expected 1 entry, got %d", cat.Count())
	}
	if cat.All()[0].Source != SourceRegistry {
		t.Fatalf("expected source=%q, got %q", SourceRegistry, cat.All()[0].Source)
	}
}

func TestMergeCuratedPrecedence(t *testing.T) {
	curated := []Entry{FromCurated(sampleService("sentry", "Curated desc"))}
	reg := []Entry{FromRegistry(sampleRegistryServer("sentry", "Sentry", "Registry desc"))}
	cat := Merge(curated, reg)

	if cat.Count() != 1 {
		t.Fatalf("expected 1 entry (curated wins), got %d", cat.Count())
	}
	if cat.All()[0].Source != SourceCurated {
		t.Fatalf("expected curated to take precedence")
	}
}

func TestMergeCaseInsensitiveCollision(t *testing.T) {
	curated := []Entry{FromCurated(sampleService("Sentry", "Curated"))}
	reg := []Entry{FromRegistry(sampleRegistryServer("sentry", "Sentry", "Registry"))}
	cat := Merge(curated, reg)

	if cat.Count() != 1 {
		t.Fatalf("expected 1 entry (case-insensitive collision), got %d", cat.Count())
	}
}

func TestMergeBothEmpty(t *testing.T) {
	cat := Merge(nil, nil)
	if cat.Count() != 0 {
		t.Fatalf("expected 0 entries, got %d", cat.Count())
	}
}

func TestAllSorted(t *testing.T) {
	curated := []Entry{
		FromCurated(sampleService("zebra", "Last")),
		FromCurated(sampleService("alpha", "First")),
	}
	cat := Merge(curated, nil)
	all := cat.All()

	if all[0].Name != "alpha" {
		t.Fatalf("expected first entry=%q, got %q", "alpha", all[0].Name)
	}
	if all[1].Name != "zebra" {
		t.Fatalf("expected second entry=%q, got %q", "zebra", all[1].Name)
	}
}

func TestAllReturnsCopy(t *testing.T) {
	curated := []Entry{FromCurated(sampleService("sentry", "Error tracking"))}
	cat := Merge(curated, nil)

	all := cat.All()
	all[0].Name = "mutated"

	if cat.entries[0].Name == "mutated" {
		t.Fatal("All() should return a copy, not a reference")
	}
}

func TestBySource(t *testing.T) {
	curated := []Entry{FromCurated(sampleService("sentry", "Curated"))}
	reg := []Entry{FromRegistry(sampleRegistryServer("ns/jira", "Jira", "Registry"))}
	cat := Merge(curated, reg)

	curatedOnly := cat.BySource(SourceCurated)
	if len(curatedOnly) != 1 {
		t.Fatalf("expected 1 curated entry, got %d", len(curatedOnly))
	}
	if curatedOnly[0].Source != SourceCurated {
		t.Fatalf("expected source=%q, got %q", SourceCurated, curatedOnly[0].Source)
	}

	registryOnly := cat.BySource(SourceRegistry)
	if len(registryOnly) != 1 {
		t.Fatalf("expected 1 registry entry, got %d", len(registryOnly))
	}
}

func TestSearchMatchesName(t *testing.T) {
	curated := []Entry{
		FromCurated(sampleService("sentry", "Error tracking")),
		FromCurated(sampleService("jira", "Project management")),
	}
	cat := Merge(curated, nil)

	results := cat.Search("sentry")
	if len(results) != 1 {
		t.Fatalf("expected 1 match, got %d", len(results))
	}
	if results[0].Name != "sentry" {
		t.Fatalf("expected match=%q, got %q", "sentry", results[0].Name)
	}
}

func TestSearchMatchesDisplayName(t *testing.T) {
	reg := []Entry{
		FromRegistry(sampleRegistryServer("ns/a", "Sentry Integration", "MCP server")),
		FromRegistry(sampleRegistryServer("ns/b", "Jira Tools", "Another server")),
	}
	cat := Merge(nil, reg)

	results := cat.Search("Sentry")
	if len(results) != 1 {
		t.Fatalf("expected 1 match, got %d", len(results))
	}
}

func TestSearchMatchesDescription(t *testing.T) {
	curated := []Entry{
		FromCurated(sampleService("tool-a", "Error tracking tool")),
		FromCurated(sampleService("tool-b", "File management")),
	}
	cat := Merge(curated, nil)

	results := cat.Search("error")
	if len(results) != 1 {
		t.Fatalf("expected 1 match, got %d", len(results))
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	curated := []Entry{FromCurated(sampleService("Sentry", "Error Tracking"))}
	cat := Merge(curated, nil)

	results := cat.Search("SENTRY")
	if len(results) != 1 {
		t.Fatalf("expected 1 match, got %d", len(results))
	}
}

func TestSearchEmptyQueryReturnsAll(t *testing.T) {
	curated := []Entry{
		FromCurated(sampleService("a", "A")),
		FromCurated(sampleService("b", "B")),
	}
	cat := Merge(curated, nil)

	results := cat.Search("")
	if len(results) != 2 {
		t.Fatalf("expected 2 results for empty query, got %d", len(results))
	}
}

func TestFindExactMatch(t *testing.T) {
	curated := []Entry{
		FromCurated(sampleService("sentry", "Error tracking")),
		FromCurated(sampleService("jira", "Project management")),
	}
	cat := Merge(curated, nil)

	entry, ok := cat.Find("sentry")
	if !ok {
		t.Fatal("expected Find to return true")
	}
	if entry.Name != "sentry" {
		t.Fatalf("expected name=%q, got %q", "sentry", entry.Name)
	}
}

func TestFindCaseInsensitive(t *testing.T) {
	curated := []Entry{FromCurated(sampleService("Sentry", "Error tracking"))}
	cat := Merge(curated, nil)

	_, ok := cat.Find("sentry")
	if !ok {
		t.Fatal("expected case-insensitive Find to match")
	}
}

func TestFindNotFound(t *testing.T) {
	curated := []Entry{FromCurated(sampleService("sentry", "Error tracking"))}
	cat := Merge(curated, nil)

	_, ok := cat.Find("nonexistent")
	if ok {
		t.Fatal("expected Find to return false for missing entry")
	}
}

func TestCount(t *testing.T) {
	curated := []Entry{
		FromCurated(sampleService("a", "A")),
		FromCurated(sampleService("b", "B")),
	}
	cat := Merge(curated, nil)

	if cat.Count() != 2 {
		t.Fatalf("expected count=2, got %d", cat.Count())
	}
}

func TestInstallTypeRemote(t *testing.T) {
	entry := FromRegistry(sampleRegistryServer("ns/a", "A", "Server A"))
	if entry.InstallType() != "remote" {
		t.Fatalf("expected install type=%q, got %q", "remote", entry.InstallType())
	}
}

func TestInstallTypePackage(t *testing.T) {
	resp := registry.ServerResponse{
		Server: registry.ServerJSON{
			Name:    "ns/a",
			Version: "1.0.0",
			Packages: []registry.Package{
				{RegistryType: "npm", Identifier: "@example/server", Transport: registry.Transport{Type: "stdio"}},
			},
		},
	}
	entry := FromRegistry(resp)
	if entry.InstallType() != "package" {
		t.Fatalf("expected install type=%q, got %q", "package", entry.InstallType())
	}
}

func TestInstallTypeRemoteAndPackage(t *testing.T) {
	resp := registry.ServerResponse{
		Server: registry.ServerJSON{
			Name:    "ns/a",
			Version: "1.0.0",
			Remotes: []registry.Transport{
				{Type: "sse", URL: "https://example.com/sse"},
			},
			Packages: []registry.Package{
				{RegistryType: "npm", Identifier: "@example/server", Transport: registry.Transport{Type: "stdio"}},
			},
		},
	}
	entry := FromRegistry(resp)
	if entry.InstallType() != "remote/package" {
		t.Fatalf("expected install type=%q, got %q", "remote/package", entry.InstallType())
	}
}

func TestInstallTypeEmpty(t *testing.T) {
	resp := registry.ServerResponse{
		Server: registry.ServerJSON{
			Name:    "ns/empty",
			Version: "1.0.0",
		},
	}
	entry := FromRegistry(resp)
	if entry.InstallType() != "" {
		t.Fatalf("expected empty install type, got %q", entry.InstallType())
	}
}

func TestInstallTypeCurated(t *testing.T) {
	svc := service.Service{Name: "test", Transport: "sse"}
	entry := FromCurated(svc)
	if entry.InstallType() != "remote" {
		t.Fatalf("expected install type=%q for curated sse, got %q", "remote", entry.InstallType())
	}
}

func TestPackageTypesRegistry(t *testing.T) {
	resp := registry.ServerResponse{
		Server: registry.ServerJSON{
			Name:    "ns/a",
			Version: "1.0.0",
			Packages: []registry.Package{
				{RegistryType: "npm", Identifier: "@example/server-a"},
				{RegistryType: "pypi", Identifier: "example-server"},
			},
		},
	}
	entry := FromRegistry(resp)
	types := entry.PackageTypes()

	if len(types) != 2 {
		t.Fatalf("expected 2 package types, got %d", len(types))
	}
	if types[0] != "npm" {
		t.Fatalf("expected first type=%q, got %q", "npm", types[0])
	}
	if types[1] != "pypi" {
		t.Fatalf("expected second type=%q, got %q", "pypi", types[1])
	}
}

func TestPackageTypesDeduplication(t *testing.T) {
	resp := registry.ServerResponse{
		Server: registry.ServerJSON{
			Name:    "ns/a",
			Version: "1.0.0",
			Packages: []registry.Package{
				{RegistryType: "npm", Identifier: "@example/server-a"},
				{RegistryType: "npm", Identifier: "@example/server-b"},
			},
		},
	}
	entry := FromRegistry(resp)
	types := entry.PackageTypes()

	if len(types) != 1 {
		t.Fatalf("expected 1 deduplicated package type, got %d", len(types))
	}
}

func TestPackageTypesCurated(t *testing.T) {
	entry := FromCurated(sampleService("test", "test"))
	types := entry.PackageTypes()

	if types != nil {
		t.Fatalf("expected nil package types for curated entry, got %v", types)
	}
}

func TestPackageTypesNoPackages(t *testing.T) {
	entry := FromRegistry(sampleRegistryServer("ns/a", "A", "Server A"))
	types := entry.PackageTypes()

	if types != nil {
		t.Fatalf("expected nil package types for entry without packages, got %v", types)
	}
}
