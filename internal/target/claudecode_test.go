package target

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/andreagrandi/mcp-wire/internal/service"
)

func TestClaudeCodeTargetMetadata(t *testing.T) {
	target := NewClaudeCodeTarget()

	if target.Name() != "Claude Code" {
		t.Fatalf("expected target name Claude Code, got %q", target.Name())
	}

	if target.Slug() != "claudecode" {
		t.Fatalf("expected target slug claudecode, got %q", target.Slug())
	}
}

func TestClaudeCodeTargetIsInstalledTrueWhenBinaryFound(t *testing.T) {
	target := newTestClaudeCodeTarget(t)
	target.lookPath = func(file string) (string, error) {
		if file != "claude" {
			t.Fatalf("expected lookup for claude binary, got %q", file)
		}

		return "/usr/local/bin/claude", nil
	}

	if !target.IsInstalled() {
		t.Fatal("expected target to be reported as installed")
	}
}

func TestClaudeCodeTargetIsInstalledFalseWhenBinaryMissing(t *testing.T) {
	target := newTestClaudeCodeTarget(t)
	target.lookPath = func(_ string) (string, error) {
		return "", errors.New("not found")
	}

	if target.IsInstalled() {
		t.Fatal("expected target to be reported as not installed")
	}
}

func TestClaudeCodeTargetInstallCreatesConfigAndAddsSSEService(t *testing.T) {
	target := newTestClaudeCodeTarget(t)

	svc := service.Service{
		Name:      "demo-service",
		Transport: "sse",
		URL:       "https://example.com/sse",
	}

	resolvedEnv := map[string]string{
		"DEMO_TOKEN": "token-value",
	}

	err := target.Install(svc, resolvedEnv)
	if err != nil {
		t.Fatalf("expected install to succeed: %v", err)
	}

	config := readTargetConfigFile(t, target.configPath)
	mcpServers := mustMapValue(t, config["mcpServers"], "mcpServers")
	serviceConfig := mustMapValue(t, mcpServers["demo-service"], "mcpServers.demo-service")

	if serviceConfig["type"] != "sse" {
		t.Fatalf("expected service type sse, got %#v", serviceConfig["type"])
	}

	if serviceConfig["url"] != "https://example.com/sse" {
		t.Fatalf("expected service url to be set, got %#v", serviceConfig["url"])
	}

	envConfig := mustMapValue(t, serviceConfig["env"], "mcpServers.demo-service.env")
	if envConfig["DEMO_TOKEN"] != "token-value" {
		t.Fatalf("expected DEMO_TOKEN env value, got %#v", envConfig["DEMO_TOKEN"])
	}
}

func TestClaudeCodeTargetInstallPreservesUnknownTopLevelKeys(t *testing.T) {
	target := newTestClaudeCodeTarget(t)

	initialConfig := map[string]any{
		"custom": map[string]any{
			"enabled": true,
		},
		"mcpServers": map[string]any{
			"existing-service": map[string]any{
				"type": "sse",
				"url":  "https://existing.example.com/sse",
			},
		},
	}

	writeTargetConfigFile(t, target.configPath, initialConfig)

	svc := service.Service{
		Name:      "new-service",
		Transport: "sse",
		URL:       "https://example.com/sse",
	}

	err := target.Install(svc, nil)
	if err != nil {
		t.Fatalf("expected install to succeed: %v", err)
	}

	updatedConfig := readTargetConfigFile(t, target.configPath)
	if _, ok := updatedConfig["custom"]; !ok {
		t.Fatal("expected unknown top-level key to be preserved")
	}

	mcpServers := mustMapValue(t, updatedConfig["mcpServers"], "mcpServers")
	if _, ok := mcpServers["existing-service"]; !ok {
		t.Fatal("expected existing service entry to be preserved")
	}

	if _, ok := mcpServers["new-service"]; !ok {
		t.Fatal("expected new service entry to be added")
	}
}

func TestClaudeCodeTargetInstallWritesStdioServiceConfiguration(t *testing.T) {
	target := newTestClaudeCodeTarget(t)

	svc := service.Service{
		Name:      "filesystem-service",
		Transport: "stdio",
		Command:   "npx",
		Args:      []string{"-y", "demo-command"},
	}

	err := target.Install(svc, nil)
	if err != nil {
		t.Fatalf("expected install to succeed: %v", err)
	}

	config := readTargetConfigFile(t, target.configPath)
	mcpServers := mustMapValue(t, config["mcpServers"], "mcpServers")
	serviceConfig := mustMapValue(t, mcpServers["filesystem-service"], "mcpServers.filesystem-service")

	if serviceConfig["type"] != "stdio" {
		t.Fatalf("expected service type stdio, got %#v", serviceConfig["type"])
	}

	if serviceConfig["command"] != "npx" {
		t.Fatalf("expected service command npx, got %#v", serviceConfig["command"])
	}

	args, ok := serviceConfig["args"].([]any)
	if !ok {
		t.Fatalf("expected args to be an array, got %#v", serviceConfig["args"])
	}

	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
}

func TestClaudeCodeTargetInstallReturnsErrorForInvalidTransport(t *testing.T) {
	target := newTestClaudeCodeTarget(t)

	err := target.Install(service.Service{Name: "demo-service", Transport: "http"}, nil)
	if err == nil {
		t.Fatal("expected install to fail for unsupported transport")
	}
}

func TestClaudeCodeTargetUninstallRemovesService(t *testing.T) {
	target := newTestClaudeCodeTarget(t)

	initialConfig := map[string]any{
		"mcpServers": map[string]any{
			"service-a": map[string]any{"type": "sse", "url": "https://a.example.com"},
			"service-b": map[string]any{"type": "sse", "url": "https://b.example.com"},
		},
	}

	writeTargetConfigFile(t, target.configPath, initialConfig)

	err := target.Uninstall("service-a")
	if err != nil {
		t.Fatalf("expected uninstall to succeed: %v", err)
	}

	updatedConfig := readTargetConfigFile(t, target.configPath)
	mcpServers := mustMapValue(t, updatedConfig["mcpServers"], "mcpServers")

	if _, ok := mcpServers["service-a"]; ok {
		t.Fatal("expected service-a to be removed")
	}

	if _, ok := mcpServers["service-b"]; !ok {
		t.Fatal("expected service-b to remain")
	}
}

func TestClaudeCodeTargetUninstallIgnoresMissingConfigFile(t *testing.T) {
	target := newTestClaudeCodeTarget(t)

	err := target.Uninstall("service-a")
	if err != nil {
		t.Fatalf("expected uninstall to succeed on missing file: %v", err)
	}
}

func TestClaudeCodeTargetListReturnsSortedServiceNames(t *testing.T) {
	target := newTestClaudeCodeTarget(t)

	initialConfig := map[string]any{
		"mcpServers": map[string]any{
			"service-b": map[string]any{"type": "sse", "url": "https://b.example.com"},
			"service-a": map[string]any{"type": "sse", "url": "https://a.example.com"},
		},
	}

	writeTargetConfigFile(t, target.configPath, initialConfig)

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

func TestClaudeCodeTargetListReturnsEmptyWhenConfigMissing(t *testing.T) {
	target := newTestClaudeCodeTarget(t)

	services, err := target.List()
	if err != nil {
		t.Fatalf("expected list to succeed: %v", err)
	}

	if len(services) != 0 {
		t.Fatalf("expected no services, got %d", len(services))
	}
}

func TestClaudeCodeTargetMethodsErrorWhenMCPServersIsNotObject(t *testing.T) {
	target := newTestClaudeCodeTarget(t)

	writeTargetConfigFile(t, target.configPath, map[string]any{"mcpServers": "invalid"})

	_, listErr := target.List()
	if listErr == nil {
		t.Fatal("expected list to fail on invalid mcpServers type")
	}

	uninstallErr := target.Uninstall("service-a")
	if uninstallErr == nil {
		t.Fatal("expected uninstall to fail on invalid mcpServers type")
	}

	installErr := target.Install(service.Service{Name: "service-a", Transport: "sse", URL: "https://example.com/sse"}, nil)
	if installErr == nil {
		t.Fatal("expected install to fail on invalid mcpServers type")
	}
}

func newTestClaudeCodeTarget(t *testing.T) *ClaudeCodeTarget {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), ".claude", "settings.json")
	target := NewClaudeCodeTarget()
	target.configPath = configPath
	target.lookPath = func(_ string) (string, error) {
		return "", errors.New("not found")
	}

	return target
}

func writeTargetConfigFile(t *testing.T, configPath string, config map[string]any) {
	t.Helper()

	configDir := filepath.Dir(configPath)
	err := os.MkdirAll(configDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	err = os.WriteFile(configPath, data, 0o600)
	if err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
}

func readTargetConfigFile(t *testing.T, configPath string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	config := map[string]any{}
	err = json.Unmarshal(data, &config)
	if err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	return config
}

func mustMapValue(t *testing.T, value any, path string) map[string]any {
	t.Helper()

	mapValue, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected %s to be an object, got %#v", path, value)
	}

	return mapValue
}
