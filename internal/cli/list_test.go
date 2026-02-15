package cli

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andreagrandi/mcp-wire/internal/config"
	"github.com/andreagrandi/mcp-wire/internal/registry"
	"github.com/andreagrandi/mcp-wire/internal/service"
	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
)

type fakeListTarget struct {
	name      string
	slug      string
	installed bool
}

func (t fakeListTarget) Name() string {
	return t.name
}

func (t fakeListTarget) Slug() string {
	return t.slug
}

func (t fakeListTarget) IsInstalled() bool {
	return t.installed
}

func (t fakeListTarget) Install(_ service.Service, _ map[string]string) error {
	return nil
}

func (t fakeListTarget) Uninstall(_ string) error {
	return nil
}

func (t fakeListTarget) List() ([]string, error) {
	return nil, nil
}

func TestListServicesCommandPrintsSortedServices(t *testing.T) {
	originalLoadServices := loadServices
	t.Cleanup(func() {
		loadServices = originalLoadServices
	})

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"zeta": {
				Name:        "zeta",
				Description: "Last service",
			},
			"alpha": {
				Name:        "alpha",
				Description: "First service",
			},
		}, nil
	}

	stubConfigWithRegistry(t, false)

	output, err := executeRootCommand(t, "list", "services", "--source", "curated")
	if err != nil {
		t.Fatalf("expected list services command to succeed: %v", err)
	}

	if !strings.Contains(output, "Available services:") {
		t.Fatalf("expected heading in output, got %q", output)
	}

	alphaIndex := strings.Index(output, "alpha")
	zetaIndex := strings.Index(output, "zeta")
	if alphaIndex == -1 || zetaIndex == -1 {
		t.Fatalf("expected both services in output, got %q", output)
	}

	if alphaIndex > zetaIndex {
		t.Fatalf("expected services sorted alphabetically, got %q", output)
	}
}

func TestListServicesCommandPrintsEmptyState(t *testing.T) {
	originalLoadServices := loadServices
	t.Cleanup(func() {
		loadServices = originalLoadServices
	})

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{}, nil
	}

	stubConfigWithRegistry(t, false)

	output, err := executeRootCommand(t, "list", "services", "--source", "curated")
	if err != nil {
		t.Fatalf("expected list services command to succeed: %v", err)
	}

	if !strings.Contains(output, "(none)") {
		t.Fatalf("expected empty state marker, got %q", output)
	}
}

func TestListServicesCommandReturnsLoaderError(t *testing.T) {
	originalLoadServices := loadServices
	t.Cleanup(func() {
		loadServices = originalLoadServices
	})

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return nil, errors.New("loader failed")
	}

	stubConfigWithRegistry(t, false)

	_, err := executeRootCommand(t, "list", "services", "--source", "curated")
	if err == nil {
		t.Fatal("expected list services command to fail")
	}

	if !strings.Contains(err.Error(), "load services") {
		t.Fatalf("expected wrapped loader error, got %v", err)
	}
}

func TestPrintServicesListPrintsServiceWithoutDescription(t *testing.T) {
	services := map[string]service.Service{
		"demo": {
			Name: "demo",
		},
	}

	var output bytes.Buffer
	printServicesList(&output, services)

	if !strings.Contains(output.String(), "  demo\n") {
		t.Fatalf("expected service name line without description, got %q", output.String())
	}
}

func TestListTargetsCommandPrintsSortedTargetsWithStatus(t *testing.T) {
	originalAllTargets := allTargets
	t.Cleanup(func() {
		allTargets = originalAllTargets
	})

	allTargets = func() []targetpkg.Target {
		return []targetpkg.Target{
			fakeListTarget{name: "Zeta CLI", slug: "zeta-cli", installed: false},
			fakeListTarget{name: "Alpha CLI", slug: "alpha-cli", installed: true},
		}
	}

	output, err := executeRootCommand(t, "list", "targets")
	if err != nil {
		t.Fatalf("expected list targets command to succeed: %v", err)
	}

	if !strings.Contains(output, "Targets:") {
		t.Fatalf("expected heading in output, got %q", output)
	}

	alphaIndex := strings.Index(output, "alpha-cli")
	zetaIndex := strings.Index(output, "zeta-cli")
	if alphaIndex == -1 || zetaIndex == -1 {
		t.Fatalf("expected both targets in output, got %q", output)
	}

	if alphaIndex > zetaIndex {
		t.Fatalf("expected targets sorted by slug, got %q", output)
	}

	if !strings.Contains(output, "installed") {
		t.Fatalf("expected installed status in output, got %q", output)
	}

	if !strings.Contains(output, "not found") {
		t.Fatalf("expected not found status in output, got %q", output)
	}
}

func TestListTargetsCommandPrintsEmptyState(t *testing.T) {
	originalAllTargets := allTargets
	t.Cleanup(func() {
		allTargets = originalAllTargets
	})

	allTargets = func() []targetpkg.Target {
		return []targetpkg.Target{}
	}

	output, err := executeRootCommand(t, "list", "targets")
	if err != nil {
		t.Fatalf("expected list targets command to succeed: %v", err)
	}

	if !strings.Contains(output, "(none)") {
		t.Fatalf("expected empty state marker, got %q", output)
	}
}

func stubConfigWithRegistry(t *testing.T, enabled bool) {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "config.json")

	originalConfig := loadConfig
	t.Cleanup(func() { loadConfig = originalConfig })

	loadConfig = func() (*config.Config, error) {
		cfg, err := config.LoadFrom(configPath)
		if err != nil {
			return nil, err
		}

		if enabled {
			if err := cfg.SetFeature("registry", true); err != nil {
				return nil, err
			}
		}

		return cfg, nil
	}
}

func TestListServicesSourceAllWithRegistryEnabled(t *testing.T) {
	originalLoadServices := loadServices
	t.Cleanup(func() { loadServices = originalLoadServices })

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"alpha": {Name: "alpha", Description: "Alpha service"},
		}, nil
	}

	originalLoadRegistryCache := loadRegistryCache
	t.Cleanup(func() { loadRegistryCache = originalLoadRegistryCache })

	loadRegistryCache = func() []registry.ServerResponse {
		return []registry.ServerResponse{
			{Server: registry.ServerJSON{Name: "beta", Description: "Beta from registry"}},
		}
	}

	stubConfigWithRegistry(t, true)

	output, err := executeRootCommand(t, "list", "services", "--source", "all")
	if err != nil {
		t.Fatalf("expected command to succeed: %v", err)
	}

	if !strings.Contains(output, "* alpha") {
		t.Fatalf("expected curated service with * marker, got %q", output)
	}

	if !strings.Contains(output, "(* = curated by mcp-wire)") {
		t.Fatalf("expected legend line in output, got %q", output)
	}

	if strings.Contains(output, "* beta") {
		t.Fatalf("expected registry service without * marker, got %q", output)
	}

	if !strings.Contains(output, "beta") {
		t.Fatalf("expected registry service in output, got %q", output)
	}
}

func TestListServicesSourceRegistryWithRegistryEnabled(t *testing.T) {
	originalLoadServices := loadServices
	t.Cleanup(func() { loadServices = originalLoadServices })

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"alpha": {Name: "alpha", Description: "Alpha service"},
		}, nil
	}

	originalLoadRegistryCache := loadRegistryCache
	t.Cleanup(func() { loadRegistryCache = originalLoadRegistryCache })

	loadRegistryCache = func() []registry.ServerResponse {
		return []registry.ServerResponse{
			{Server: registry.ServerJSON{Name: "beta", Description: "Beta from registry"}},
		}
	}

	stubConfigWithRegistry(t, true)

	output, err := executeRootCommand(t, "list", "services", "--source", "registry")
	if err != nil {
		t.Fatalf("expected command to succeed: %v", err)
	}

	if strings.Contains(output, "alpha") {
		t.Fatalf("expected no curated services in registry-only output, got %q", output)
	}

	if !strings.Contains(output, "beta") {
		t.Fatalf("expected registry service in output, got %q", output)
	}
}

func TestListServicesDefaultSourceUnchanged(t *testing.T) {
	originalLoadServices := loadServices
	t.Cleanup(func() { loadServices = originalLoadServices })

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"alpha": {Name: "alpha", Description: "Alpha service"},
		}, nil
	}

	stubConfigWithRegistry(t, false)

	output, err := executeRootCommand(t, "list", "services", "--source", "curated")
	if err != nil {
		t.Fatalf("expected command to succeed: %v", err)
	}

	if !strings.Contains(output, "alpha") {
		t.Fatalf("expected curated service in default output, got %q", output)
	}

	if !strings.Contains(output, "Available services:") {
		t.Fatalf("expected heading in output, got %q", output)
	}
}

func TestListServicesInvalidSourceReturnsError(t *testing.T) {
	stubConfigWithRegistry(t, true)

	_, err := executeRootCommand(t, "list", "services", "--source", "bogus")
	if err == nil {
		t.Fatal("expected error for invalid --source value")
	}

	if !strings.Contains(err.Error(), "invalid --source value") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestListServicesSourceAllWithRegistryDisabledReturnsError(t *testing.T) {
	stubConfigWithRegistry(t, false)

	_, err := executeRootCommand(t, "list", "services", "--source", "all")
	if err == nil {
		t.Fatal("expected error when using --source with registry disabled")
	}

	if !strings.Contains(err.Error(), "--source requires the registry feature") {
		t.Fatalf("expected feature flag error, got %v", err)
	}
}

func executeRootCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer

	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs(args)
	t.Cleanup(func() {
		rootCmd.SetArgs([]string{})
	})

	err := rootCmd.Execute()
	output := stdout.String() + stderr.String()

	return output, err
}
