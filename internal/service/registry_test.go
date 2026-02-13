package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateServiceRejectsMissingName(t *testing.T) {
	service := Service{
		Transport: "sse",
		URL:       "https://example.com/sse",
	}

	err := ValidateService(service)
	if err == nil {
		t.Fatal("expected validation error for missing name")
	}
}

func TestValidateServiceRejectsUnsupportedTransport(t *testing.T) {
	service := Service{
		Name:      "demo-service",
		Transport: "http",
	}

	err := ValidateService(service)
	if err == nil {
		t.Fatal("expected validation error for unsupported transport")
	}
}

func TestValidateServiceRequiresURLForSSE(t *testing.T) {
	service := Service{
		Name:      "demo-service",
		Transport: "sse",
	}

	err := ValidateService(service)
	if err == nil {
		t.Fatal("expected validation error for missing sse url")
	}
}

func TestValidateServiceRequiresCommandForStdio(t *testing.T) {
	service := Service{
		Name:      "filesystem",
		Transport: "stdio",
	}

	err := ValidateService(service)
	if err == nil {
		t.Fatal("expected validation error for missing stdio command")
	}
}

func TestLoadServicesLoadsDefinitionsFromMultiplePaths(t *testing.T) {
	bundledDir := t.TempDir()
	userDir := t.TempDir()

	bundledService := `name: demo-service
description: "Bundled demo service"
transport: sse
url: "https://bundled.example.com/sse"
env: []
`

	userService := `name: demo-service
description: "User demo service"
transport: sse
url: "https://user.example.com/sse"
env: []
`

	filesystem := `name: filesystem
description: "Filesystem"
transport: stdio
command: npx
args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
env: []
`

	writeTestFile(t, filepath.Join(bundledDir, "demo-service.yaml"), bundledService)
	writeTestFile(t, filepath.Join(userDir, "demo-service.yaml"), userService)
	writeTestFile(t, filepath.Join(userDir, "filesystem.yaml"), filesystem)
	writeTestFile(t, filepath.Join(userDir, "notes.txt"), "not yaml")

	services, err := LoadServices(bundledDir, userDir)
	if err != nil {
		t.Fatalf("expected services to load: %v", err)
	}

	if len(services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(services))
	}

	demoService, ok := services["demo-service"]
	if !ok {
		t.Fatal("expected demo-service to be present")
	}

	if demoService.Description != "User demo service" {
		t.Fatalf("expected user definition to override bundled definition, got %q", demoService.Description)
	}

	if _, ok := services["filesystem"]; !ok {
		t.Fatal("expected filesystem service to be present")
	}
}

func TestLoadServicesReturnsErrorForInvalidServiceDefinition(t *testing.T) {
	servicesDir := t.TempDir()

	invalidService := `name: broken
description: "Broken"
transport: sse
env: []
`

	writeTestFile(t, filepath.Join(servicesDir, "broken.yaml"), invalidService)

	_, err := LoadServices(servicesDir)
	if err == nil {
		t.Fatal("expected error for invalid service definition")
	}
}

func TestLoadServicesSkipsMissingDirectories(t *testing.T) {
	servicesDir := t.TempDir()

	serviceDefinition := `name: docs-service
description: "Documentation service"
transport: sse
url: "https://docs.example.com/mcp"
env: []
`

	writeTestFile(t, filepath.Join(servicesDir, "docs-service.yaml"), serviceDefinition)

	missingDir := filepath.Join(servicesDir, "missing")
	services, err := LoadServices(missingDir, servicesDir)
	if err != nil {
		t.Fatalf("expected missing directory to be ignored: %v", err)
	}

	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}

	if _, ok := services["docs-service"]; !ok {
		t.Fatal("expected docs-service to be present")
	}
}

func TestResolveServicePathsWithoutHomeDirectory(t *testing.T) {
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		if originalHome == "" {
			if err := os.Unsetenv("HOME"); err != nil {
				t.Fatalf("failed to unset HOME: %v", err)
			}
			return
		}

		if err := os.Setenv("HOME", originalHome); err != nil {
			t.Fatalf("failed to restore HOME: %v", err)
		}
	})

	if err := os.Unsetenv("HOME"); err != nil {
		t.Fatalf("failed to unset HOME: %v", err)
	}

	paths, err := resolveServicePaths()
	if err != nil {
		t.Fatalf("expected resolveServicePaths to succeed, got: %v", err)
	}

	if len(paths) == 0 {
		t.Fatal("expected at least bundled services path")
	}
}

func TestLoadServicesLoadsEmbeddedDefaultsWhenPathsAreMissing(t *testing.T) {
	originalWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	isolatedDirectory := t.TempDir()
	if err := os.Chdir(isolatedDirectory); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalWorkingDirectory); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})

	t.Setenv("HOME", isolatedDirectory)
	t.Setenv("USERPROFILE", isolatedDirectory)

	services, err := LoadServices()
	if err != nil {
		t.Fatalf("expected embedded services to load: %v", err)
	}

	for _, name := range []string{"jira", "sentry", "context7", "playwright"} {
		if _, ok := services[name]; !ok {
			t.Fatalf("expected service %q to be available from embedded defaults", name)
		}
	}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()

	err := os.WriteFile(path, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("failed to write test file %q: %v", path, err)
	}
}
