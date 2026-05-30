package target

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/andreagrandi/mcp-wire/internal/service"
)

// bundledServicesDir resolves the repository's services/ directory, which holds
// the curated definitions embedded into the binary at build time.
func bundledServicesDir(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve current file path")
	}

	return filepath.Join(filepath.Dir(currentFile), "..", "..", "services")
}

func TestClaudeCodeTargetInstallsBundledOAuthRemoteServices(t *testing.T) {
	services, err := service.LoadServices(bundledServicesDir(t))
	if err != nil {
		t.Fatalf("failed to load bundled services: %v", err)
	}

	testCases := []struct {
		name string
		url  string
	}{
		{name: "github", url: "https://api.githubcopilot.com/mcp/"},
		{name: "notion", url: "https://mcp.notion.com/mcp"},
		{name: "linear", url: "https://mcp.linear.app/mcp"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			svc, ok := services[testCase.name]
			if !ok {
				t.Fatalf("expected bundled service %q to be available", testCase.name)
			}

			target := newTestClaudeCodeTarget(t)
			if err := target.Install(svc, nil); err != nil {
				t.Fatalf("expected install to succeed: %v", err)
			}

			config := readTargetConfigFile(t, target.configPath)
			mcpServers := mustMapValue(t, config["mcpServers"], "mcpServers")
			serviceConfig := mustMapValue(t, mcpServers[testCase.name], "mcpServers."+testCase.name)

			if serviceConfig["type"] != "http" {
				t.Fatalf("expected service type http, got %#v", serviceConfig["type"])
			}

			if serviceConfig["url"] != testCase.url {
				t.Fatalf("expected service url %q, got %#v", testCase.url, serviceConfig["url"])
			}
		})
	}
}
