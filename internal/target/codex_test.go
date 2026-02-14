package target

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/andreagrandi/mcp-wire/internal/service"
	toml "github.com/pelletier/go-toml/v2"
)

func TestCodexTargetMetadata(t *testing.T) {
	target := NewCodexTarget()

	if target.Name() != "Codex CLI" {
		t.Fatalf("expected target name Codex CLI, got %q", target.Name())
	}

	if target.Slug() != "codex" {
		t.Fatalf("expected target slug codex, got %q", target.Slug())
	}
}

func TestCodexTargetIsInstalledTrueWhenBinaryFound(t *testing.T) {
	target := newTestCodexTarget(t)
	target.lookPath = func(file string) (string, error) {
		if file != "codex" {
			t.Fatalf("expected lookup for codex binary, got %q", file)
		}

		return "/usr/local/bin/codex", nil
	}

	if !target.IsInstalled() {
		t.Fatal("expected target to be reported as installed")
	}
}

func TestCodexTargetIsInstalledFalseWhenBinaryMissing(t *testing.T) {
	target := newTestCodexTarget(t)
	target.lookPath = func(_ string) (string, error) {
		return "", errors.New("not found")
	}

	if target.IsInstalled() {
		t.Fatal("expected target to be reported as not installed")
	}
}

func TestCodexTargetInstallCreatesConfigAndAddsStdioService(t *testing.T) {
	target := newTestCodexTarget(t)

	svc := service.Service{
		Name:      "demo-service",
		Transport: "stdio",
		Command:   "npx",
		Args:      []string{"-y", "demo-command"},
	}

	resolvedEnv := map[string]string{
		"DEMO_TOKEN": "token-value",
	}

	err := target.Install(svc, resolvedEnv)
	if err != nil {
		t.Fatalf("expected install to succeed: %v", err)
	}

	config := readCodexConfigFile(t, target.configPath)
	mcpServers := mustMapValue(t, config["mcp_servers"], "mcp_servers")
	serviceConfig := mustMapValue(t, mcpServers["demo-service"], "mcp_servers.demo-service")

	if serviceConfig["command"] != "npx" {
		t.Fatalf("expected command npx, got %#v", serviceConfig["command"])
	}

	args, ok := serviceConfig["args"].([]any)
	if !ok {
		t.Fatalf("expected args to be an array, got %#v", serviceConfig["args"])
	}

	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}

	envConfig := mustMapValue(t, serviceConfig["env"], "mcp_servers.demo-service.env")
	if envConfig["DEMO_TOKEN"] != "token-value" {
		t.Fatalf("expected DEMO_TOKEN env value, got %#v", envConfig["DEMO_TOKEN"])
	}
}

func TestCodexTargetInstallCreatesHTTPServiceUsingBearerTokenEnvVar(t *testing.T) {
	target := newTestCodexTarget(t)

	svc := service.Service{
		Name:      "docs-service",
		Transport: "http",
		URL:       "https://example.com/mcp",
		Env: []service.EnvVar{
			{Name: "DOCS_TOKEN", Required: true},
		},
	}

	err := target.Install(svc, map[string]string{"DOCS_TOKEN": "secret"})
	if err != nil {
		t.Fatalf("expected install to succeed: %v", err)
	}

	config := readCodexConfigFile(t, target.configPath)
	mcpServers := mustMapValue(t, config["mcp_servers"], "mcp_servers")
	serviceConfig := mustMapValue(t, mcpServers["docs-service"], "mcp_servers.docs-service")

	if serviceConfig["url"] != "https://example.com/mcp" {
		t.Fatalf("expected URL to be set, got %#v", serviceConfig["url"])
	}

	if serviceConfig["bearer_token_env_var"] != "DOCS_TOKEN" {
		t.Fatalf("expected bearer_token_env_var DOCS_TOKEN, got %#v", serviceConfig["bearer_token_env_var"])
	}
}

func TestCodexTargetInstallPreservesUnknownTopLevelKeys(t *testing.T) {
	target := newTestCodexTarget(t)

	initialConfig := map[string]any{
		"model": "gpt-demo",
		"mcp_servers": map[string]any{
			"existing-service": map[string]any{
				"url": "https://existing.example.com/mcp",
			},
		},
	}

	writeCodexConfigFile(t, target.configPath, initialConfig)

	svc := service.Service{
		Name:      "new-service",
		Transport: "sse",
		URL:       "https://example.com/mcp",
	}

	err := target.Install(svc, nil)
	if err != nil {
		t.Fatalf("expected install to succeed: %v", err)
	}

	updatedConfig := readCodexConfigFile(t, target.configPath)
	if _, ok := updatedConfig["model"]; !ok {
		t.Fatal("expected unknown top-level key to be preserved")
	}

	mcpServers := mustMapValue(t, updatedConfig["mcp_servers"], "mcp_servers")
	if _, ok := mcpServers["existing-service"]; !ok {
		t.Fatal("expected existing service entry to be preserved")
	}

	if _, ok := mcpServers["new-service"]; !ok {
		t.Fatal("expected new service entry to be added")
	}
}

func TestCodexTargetInstallReturnsErrorForInvalidTransport(t *testing.T) {
	target := newTestCodexTarget(t)

	err := target.Install(service.Service{Name: "demo-service", Transport: "grpc"}, nil)
	if err == nil {
		t.Fatal("expected install to fail for unsupported transport")
	}
}

func TestCodexTargetUninstallRemovesService(t *testing.T) {
	target := newTestCodexTarget(t)

	initialConfig := map[string]any{
		"mcp_servers": map[string]any{
			"service-a": map[string]any{"url": "https://a.example.com/mcp"},
			"service-b": map[string]any{"url": "https://b.example.com/mcp"},
		},
	}

	writeCodexConfigFile(t, target.configPath, initialConfig)

	err := target.Uninstall("service-a")
	if err != nil {
		t.Fatalf("expected uninstall to succeed: %v", err)
	}

	updatedConfig := readCodexConfigFile(t, target.configPath)
	mcpServers := mustMapValue(t, updatedConfig["mcp_servers"], "mcp_servers")

	if _, ok := mcpServers["service-a"]; ok {
		t.Fatal("expected service-a to be removed")
	}

	if _, ok := mcpServers["service-b"]; !ok {
		t.Fatal("expected service-b to remain")
	}
}

func TestCodexTargetUninstallIgnoresMissingConfigFile(t *testing.T) {
	target := newTestCodexTarget(t)

	err := target.Uninstall("service-a")
	if err != nil {
		t.Fatalf("expected uninstall to succeed on missing file: %v", err)
	}
}

func TestCodexTargetListReturnsSortedServiceNames(t *testing.T) {
	target := newTestCodexTarget(t)

	initialConfig := map[string]any{
		"mcp_servers": map[string]any{
			"service-b": map[string]any{"url": "https://b.example.com/mcp"},
			"service-a": map[string]any{"url": "https://a.example.com/mcp"},
		},
	}

	writeCodexConfigFile(t, target.configPath, initialConfig)

	services, err := target.List()
	if err != nil {
		t.Fatalf("expected list to succeed: %v", err)
	}

	if len(services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(services))
	}

	if services[0] != "service-a" || services[1] != "service-b" {
		t.Fatalf("expected sorted service names, got %#v", services)
	}
}

func TestCodexTargetListReturnsEmptyWhenConfigMissing(t *testing.T) {
	target := newTestCodexTarget(t)

	services, err := target.List()
	if err != nil {
		t.Fatalf("expected list to succeed: %v", err)
	}

	if len(services) != 0 {
		t.Fatalf("expected no services, got %d", len(services))
	}
}

func TestCodexTargetMethodsErrorWhenMCPServersIsNotTable(t *testing.T) {
	target := newTestCodexTarget(t)

	writeCodexConfigFile(t, target.configPath, map[string]any{"mcp_servers": "invalid"})

	_, listErr := target.List()
	if listErr == nil {
		t.Fatal("expected list to fail on invalid mcp_servers type")
	}

	uninstallErr := target.Uninstall("service-a")
	if uninstallErr == nil {
		t.Fatal("expected uninstall to fail on invalid mcp_servers type")
	}

	installErr := target.Install(service.Service{Name: "service-a", Transport: "sse", URL: "https://example.com/mcp"}, nil)
	if installErr == nil {
		t.Fatal("expected install to fail on invalid mcp_servers type")
	}
}

func TestPickBearerEnvVarPrefersServiceDefinitionOrder(t *testing.T) {
	svc := service.Service{
		Env: []service.EnvVar{
			{Name: "SECOND_TOKEN"},
			{Name: "FIRST_TOKEN"},
		},
	}

	resolved := map[string]string{
		"FIRST_TOKEN":  "one",
		"SECOND_TOKEN": "two",
	}

	bearerEnvVar := pickBearerEnvVar(svc, resolved)
	if bearerEnvVar != "SECOND_TOKEN" {
		t.Fatalf("expected SECOND_TOKEN from service order, got %q", bearerEnvVar)
	}
}

func newTestCodexTarget(t *testing.T) *CodexTarget {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), ".codex", "config.toml")
	target := NewCodexTarget()
	target.configPath = configPath
	target.lookPath = func(_ string) (string, error) {
		return "", errors.New("not found")
	}

	return target
}

func writeCodexConfigFile(t *testing.T, configPath string, config map[string]any) {
	t.Helper()

	configDir := filepath.Dir(configPath)
	err := os.MkdirAll(configDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	data, err := toml.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	err = os.WriteFile(configPath, data, 0o600)
	if err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
}

func readCodexConfigFile(t *testing.T, configPath string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	config := map[string]any{}
	err = toml.Unmarshal(data, &config)
	if err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	return config
}
