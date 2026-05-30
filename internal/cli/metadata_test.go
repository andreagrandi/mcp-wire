package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andreagrandi/mcp-wire/internal/catalog"
	"github.com/andreagrandi/mcp-wire/internal/config"
	"github.com/andreagrandi/mcp-wire/internal/service"
	"github.com/andreagrandi/mcp-wire/internal/target"
)

type fakeMetadataTarget struct {
	name       string
	slug       string
	installed  bool
	configPath string
}

func (t fakeMetadataTarget) Name() string                                         { return t.name }
func (t fakeMetadataTarget) Slug() string                                         { return t.slug }
func (t fakeMetadataTarget) IsInstalled() bool                                    { return t.installed }
func (t fakeMetadataTarget) Install(_ service.Service, _ map[string]string) error { return nil }
func (t fakeMetadataTarget) Uninstall(_ string) error                             { return nil }
func (t fakeMetadataTarget) List() ([]string, error)                              { return nil, nil }
func (t fakeMetadataTarget) ConfigPath() string                                   { return t.configPath }

// fakeScopedMetadataTarget adds scope support but not OAuth.
type fakeScopedMetadataTarget struct {
	fakeMetadataTarget
	scopes []target.ConfigScope
}

func (t fakeScopedMetadataTarget) SupportedScopes() []target.ConfigScope { return t.scopes }
func (t fakeScopedMetadataTarget) InstallWithScope(_ service.Service, _ map[string]string, _ target.ConfigScope) error {
	return nil
}
func (t fakeScopedMetadataTarget) UninstallWithScope(_ string, _ target.ConfigScope) error {
	return nil
}
func (t fakeScopedMetadataTarget) ListWithScope(_ target.ConfigScope) ([]string, error) {
	return nil, nil
}

// fakeAuthMetadataTarget adds automatic OAuth support but not scopes.
type fakeAuthMetadataTarget struct {
	fakeMetadataTarget
}

func (t fakeAuthMetadataTarget) Authenticate(_ string, _ io.Reader, _ io.Writer, _ io.Writer) error {
	return nil
}

func sampleMetadataCatalog() *catalog.Catalog {
	services := map[string]service.Service{
		"sentry": {
			Name:        "sentry",
			Description: "Error tracking",
			Transport:   "sse",
			Auth:        "oauth",
			URL:         "https://example.com/sse",
		},
		"linear": {
			Name:        "linear",
			Description: "Issue tracking",
			Transport:   "http",
			URL:         "https://example.com/mcp",
			Env: []service.EnvVar{
				{
					Name:        "LINEAR_TOKEN",
					Description: "API token",
					Required:    true,
					SetupURL:    "https://linear.app/tokens",
					SetupHint:   "Create a token",
				},
			},
		},
		"playwright": {
			Name:        "playwright",
			Description: "Browser automation",
			Transport:   "stdio",
			Command:     "npx",
			Args:        []string{"-y", "@playwright/mcp@latest"},
		},
	}

	return catalog.Merge(catalog.FromCuratedMap(services), nil)
}

type capturedCatalogCall struct {
	source          string
	registryEnabled bool
	called          bool
}

func newTestMetadataDeps(t *testing.T, targets []target.Target, registryEnabled bool) (metadataDeps, *capturedCatalogCall) {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "config.json")
	if registryEnabled {
		cfg, err := config.LoadFrom(configPath)
		if err != nil {
			t.Fatalf("failed to load test config: %v", err)
		}
		if err := cfg.SetFeature("registry", true); err != nil {
			t.Fatalf("failed to enable registry: %v", err)
		}
	}

	captured := &capturedCatalogCall{}

	deps := metadataDeps{
		loadConfig: func() (*config.Config, error) { return config.LoadFrom(configPath) },
		allTargets: func() []target.Target { return targets },
		loadCatalog: func(source string, regEnabled bool) (*catalog.Catalog, error) {
			captured.source = source
			captured.registryEnabled = regEnabled
			captured.called = true
			return sampleMetadataCatalog(), nil
		},
		version: "test-version",
	}

	return deps, captured
}

func runMetadataToDocument(t *testing.T, deps metadataDeps, source string) metadataDocument {
	t.Helper()

	buf := new(bytes.Buffer)
	if err := runMetadata(buf, deps, source); err != nil {
		t.Fatalf("expected metadata to succeed: %v", err)
	}

	var doc metadataDocument
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("expected valid JSON output, got error %v for output:\n%s", err, buf.String())
	}

	return doc
}

func TestMetadataTopLevelCapabilities(t *testing.T) {
	deps, _ := newTestMetadataDeps(t, nil, false)

	doc := runMetadataToDocument(t, deps, "curated")

	if doc.SchemaVersion != metadataSchemaVersion {
		t.Fatalf("expected schema_version %d, got %d", metadataSchemaVersion, doc.SchemaVersion)
	}

	if doc.MCPWireVersion != "test-version" {
		t.Fatalf("expected version %q, got %q", "test-version", doc.MCPWireVersion)
	}

	if strings.Join(doc.Transports, ",") != "http,sse,stdio" {
		t.Fatalf("expected transports [http sse stdio], got %v", doc.Transports)
	}

	if strings.Join(doc.Scopes, ",") != "user,project" {
		t.Fatalf("expected scopes [user project], got %v", doc.Scopes)
	}
}

func TestMetadataTargetsSortedWithCapabilities(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "claude.json")
	targets := []target.Target{
		fakeAuthMetadataTarget{
			fakeMetadataTarget: fakeMetadataTarget{name: "Codex CLI", slug: "codex", installed: false},
		},
		fakeScopedMetadataTarget{
			fakeMetadataTarget: fakeMetadataTarget{
				name:       "Claude Code",
				slug:       "claude",
				installed:  true,
				configPath: configPath,
			},
			scopes: []target.ConfigScope{target.ConfigScopeUser, target.ConfigScopeProject, target.ConfigScopeEffective},
		},
		fakeMetadataTarget{name: "OpenCode", slug: "opencode", installed: true},
	}

	deps, _ := newTestMetadataDeps(t, targets, false)
	doc := runMetadataToDocument(t, deps, "curated")

	if len(doc.Targets) != 3 {
		t.Fatalf("expected 3 targets, got %d", len(doc.Targets))
	}

	slugs := []string{doc.Targets[0].Slug, doc.Targets[1].Slug, doc.Targets[2].Slug}
	if strings.Join(slugs, ",") != "claude,codex,opencode" {
		t.Fatalf("expected targets sorted by slug, got %v", slugs)
	}

	claude := doc.Targets[0]
	if !claude.Installed {
		t.Fatal("expected claude installed=true")
	}
	if claude.ConfigPath != configPath {
		t.Fatalf("expected claude config_path %q, got %q", configPath, claude.ConfigPath)
	}
	if strings.Join(claude.Scopes, ",") != "user,project,effective" {
		t.Fatalf("expected claude scopes [user project effective], got %v", claude.Scopes)
	}
	if claude.SupportsOAuth {
		t.Fatal("expected claude supports_oauth=false")
	}

	codex := doc.Targets[1]
	if codex.Installed {
		t.Fatal("expected codex installed=false")
	}
	if !codex.SupportsOAuth {
		t.Fatal("expected codex supports_oauth=true")
	}
	if strings.Join(codex.Scopes, ",") != "user" {
		t.Fatalf("expected codex scopes [user], got %v", codex.Scopes)
	}

	opencode := doc.Targets[2]
	if strings.Join(opencode.Scopes, ",") != "user" {
		t.Fatalf("expected opencode default scopes [user], got %v", opencode.Scopes)
	}
	if opencode.SupportsOAuth {
		t.Fatal("expected opencode supports_oauth=false")
	}
}

func findMetadataService(doc metadataDocument, name string) (metadataService, bool) {
	for _, svc := range doc.Services {
		if svc.Name == name {
			return svc, true
		}
	}
	return metadataService{}, false
}

func TestMetadataServicesIncludeAuthAndEnv(t *testing.T) {
	deps, _ := newTestMetadataDeps(t, nil, false)
	doc := runMetadataToDocument(t, deps, "curated")

	sentry, ok := findMetadataService(doc, "sentry")
	if !ok {
		t.Fatal("expected sentry service in metadata")
	}
	if sentry.Source != "curated" {
		t.Fatalf("expected sentry source=curated, got %q", sentry.Source)
	}
	if sentry.Transport != "sse" {
		t.Fatalf("expected sentry transport=sse, got %q", sentry.Transport)
	}
	if sentry.Auth != "oauth" {
		t.Fatalf("expected sentry auth=oauth, got %q", sentry.Auth)
	}
	if sentry.InstallMethod != "remote" {
		t.Fatalf("expected sentry install_method=remote, got %q", sentry.InstallMethod)
	}

	linear, ok := findMetadataService(doc, "linear")
	if !ok {
		t.Fatal("expected linear service in metadata")
	}
	if linear.Auth != "api_key" {
		t.Fatalf("expected linear auth=api_key, got %q", linear.Auth)
	}
	if len(linear.Env) != 1 {
		t.Fatalf("expected linear to have 1 env var, got %d", len(linear.Env))
	}
	env := linear.Env[0]
	if env.Name != "LINEAR_TOKEN" || !env.Required {
		t.Fatalf("expected required LINEAR_TOKEN env var, got %+v", env)
	}
	if env.SetupURL != "https://linear.app/tokens" || env.SetupHint != "Create a token" {
		t.Fatalf("expected setup metadata on env var, got %+v", env)
	}

	playwright, ok := findMetadataService(doc, "playwright")
	if !ok {
		t.Fatal("expected playwright service in metadata")
	}
	if playwright.Transport != "stdio" {
		t.Fatalf("expected playwright transport=stdio, got %q", playwright.Transport)
	}
	if playwright.Auth != "none" {
		t.Fatalf("expected playwright auth=none, got %q", playwright.Auth)
	}
	if playwright.InstallMethod != "local" {
		t.Fatalf("expected playwright install_method=local, got %q", playwright.InstallMethod)
	}
}

func TestMetadataReportsFeatures(t *testing.T) {
	deps, _ := newTestMetadataDeps(t, nil, true)
	doc := runMetadataToDocument(t, deps, "curated")

	var registry *metadataFeature
	for i := range doc.Features {
		if doc.Features[i].Name == "registry" {
			registry = &doc.Features[i]
			break
		}
	}

	if registry == nil {
		t.Fatalf("expected registry feature in metadata, got %+v", doc.Features)
	}
	if !registry.Enabled {
		t.Fatal("expected registry feature enabled=true")
	}
	if registry.Description == "" {
		t.Fatal("expected registry feature to have a description")
	}
}

func TestMetadataPassesSourceAndRegistryFlag(t *testing.T) {
	deps, captured := newTestMetadataDeps(t, nil, true)

	if _, err := buildMetadataDocument(deps, "all"); err != nil {
		t.Fatalf("expected buildMetadataDocument to succeed: %v", err)
	}

	if !captured.called {
		t.Fatal("expected loadCatalog to be called")
	}
	if captured.source != "all" {
		t.Fatalf("expected source %q passed to loadCatalog, got %q", "all", captured.source)
	}
	if !captured.registryEnabled {
		t.Fatal("expected registryEnabled=true passed to loadCatalog when feature is on")
	}
}

func TestMetadataOutputNeverContainsNull(t *testing.T) {
	deps, _ := newTestMetadataDeps(t, []target.Target{
		fakeMetadataTarget{name: "OpenCode", slug: "opencode", installed: true},
	}, false)

	buf := new(bytes.Buffer)
	if err := runMetadata(buf, deps, "curated"); err != nil {
		t.Fatalf("expected metadata to succeed: %v", err)
	}

	if strings.Contains(buf.String(), "null") {
		t.Fatalf("expected no null values in metadata output, got:\n%s", buf.String())
	}
}

func TestMetadataConfigLoadFailure(t *testing.T) {
	deps, _ := newTestMetadataDeps(t, nil, false)
	deps.loadConfig = func() (*config.Config, error) { return nil, errors.New("boom") }

	buf := new(bytes.Buffer)
	err := runMetadata(buf, deps, "curated")
	if err == nil {
		t.Fatal("expected error when config load fails")
	}
	if !strings.Contains(err.Error(), "load config") {
		t.Fatalf("expected wrapped config error, got %v", err)
	}
}

func TestMetadataInvalidSource(t *testing.T) {
	output, err := executeRootCommand(t, "metadata", "--source", "bogus")
	if err == nil {
		t.Fatal("expected error for invalid source")
	}
	if !strings.Contains(output, "invalid --source") {
		t.Fatalf("expected invalid source message, got %q", output)
	}
}

func TestMetadataCommandRegistered(t *testing.T) {
	output, err := executeRootCommand(t, "metadata", "--help")
	if err != nil {
		t.Fatalf("expected metadata --help to succeed: %v", err)
	}

	if !strings.Contains(output, "JSON") {
		t.Fatalf("expected metadata help to mention JSON, got %q", output)
	}
	if !strings.Contains(output, "schema_version") {
		t.Fatalf("expected metadata help to mention schema_version, got %q", output)
	}
}

func TestNormalizeAuthLabel(t *testing.T) {
	cases := map[string]string{
		"OAuth":   "oauth",
		"API key": "api_key",
		"none":    "none",
		"":        "none",
	}

	for input, want := range cases {
		if got := normalizeAuthLabel(input); got != want {
			t.Fatalf("normalizeAuthLabel(%q) = %q, want %q", input, got, want)
		}
	}
}
