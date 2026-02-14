package target

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/andreagrandi/mcp-wire/internal/service"
)

func TestOpenCodeTargetMetadata(t *testing.T) {
	target := NewOpenCodeTarget()

	if target.Name() != "OpenCode" {
		t.Fatalf("expected target name OpenCode, got %q", target.Name())
	}

	if target.Slug() != "opencode" {
		t.Fatalf("expected target slug opencode, got %q", target.Slug())
	}
}

func TestOpenCodeTargetIsInstalledTrueWhenBinaryFound(t *testing.T) {
	target := newTestOpenCodeTarget(t)
	target.lookPath = func(file string) (string, error) {
		if file != "opencode" {
			return "", errors.New("not found")
		}

		return "/usr/local/bin/opencode", nil
	}

	if !target.IsInstalled() {
		t.Fatal("expected target to be reported as installed")
	}
}

func TestOpenCodeTargetIsInstalledTrueWhenFallbackBinaryExists(t *testing.T) {
	target := newTestOpenCodeTarget(t)
	target.lookPath = func(_ string) (string, error) {
		return "", errors.New("not found")
	}

	fallbackBinaryPath := filepath.Join(t.TempDir(), "opencode")
	err := os.WriteFile(fallbackBinaryPath, []byte("#!/bin/sh\nexit 0\n"), 0o755)
	if err != nil {
		t.Fatalf("failed to create fallback binary: %v", err)
	}

	target.fallbackBinaryPaths = []string{fallbackBinaryPath}
	target.statPath = os.Stat

	if !target.IsInstalled() {
		t.Fatal("expected target to be reported as installed via fallback binary")
	}
}

func TestOpenCodeTargetIsInstalledFalseWhenFallbackBinaryNotExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows treats existing fallback files as executable")
	}

	target := newTestOpenCodeTarget(t)
	target.lookPath = func(_ string) (string, error) {
		return "", errors.New("not found")
	}

	fallbackBinaryPath := filepath.Join(t.TempDir(), "opencode")
	err := os.WriteFile(fallbackBinaryPath, []byte("#!/bin/sh\nexit 0\n"), 0o600)
	if err != nil {
		t.Fatalf("failed to create fallback file: %v", err)
	}

	target.fallbackBinaryPaths = []string{fallbackBinaryPath}
	target.statPath = os.Stat

	if target.IsInstalled() {
		t.Fatal("expected target to be reported as not installed for non-executable fallback")
	}
}

func TestOpenCodeTargetInstallCreatesConfigAndAddsRemoteService(t *testing.T) {
	target := newTestOpenCodeTarget(t)

	svc := service.Service{
		Name:      "demo-service",
		Transport: "http",
		URL:       "https://example.com/mcp",
	}

	resolvedEnv := map[string]string{
		"DEMO_TOKEN": "token-value",
	}

	err := target.Install(svc, resolvedEnv)
	if err != nil {
		t.Fatalf("expected install to succeed: %v", err)
	}

	config := readOpenCodeConfigFile(t, target.configPath)
	mcpEntries := mustMapValue(t, config["mcp"], "mcp")
	serviceConfig := mustMapValue(t, mcpEntries["demo-service"], "mcp.demo-service")

	if serviceConfig["type"] != "remote" {
		t.Fatalf("expected type remote, got %#v", serviceConfig["type"])
	}

	if serviceConfig["url"] != "https://example.com/mcp" {
		t.Fatalf("expected URL to be set, got %#v", serviceConfig["url"])
	}

	if serviceConfig["enabled"] != true {
		t.Fatalf("expected enabled to be true, got %#v", serviceConfig["enabled"])
	}

	headers := mustMapValue(t, serviceConfig["headers"], "mcp.demo-service.headers")
	if headers["DEMO_TOKEN"] != "token-value" {
		t.Fatalf("expected header DEMO_TOKEN token-value, got %#v", headers["DEMO_TOKEN"])
	}
}

func TestOpenCodeTargetInstallWritesLocalServiceConfiguration(t *testing.T) {
	target := newTestOpenCodeTarget(t)

	svc := service.Service{
		Name:      "filesystem-service",
		Transport: "stdio",
		Command:   "npx",
		Args:      []string{"-y", "demo-command", "--flag"},
	}

	err := target.Install(svc, map[string]string{"TOKEN": "value"})
	if err != nil {
		t.Fatalf("expected install to succeed: %v", err)
	}

	config := readOpenCodeConfigFile(t, target.configPath)
	mcpEntries := mustMapValue(t, config["mcp"], "mcp")
	serviceConfig := mustMapValue(t, mcpEntries["filesystem-service"], "mcp.filesystem-service")

	if serviceConfig["type"] != "local" {
		t.Fatalf("expected type local, got %#v", serviceConfig["type"])
	}

	commandParts, ok := serviceConfig["command"].([]any)
	if !ok {
		t.Fatalf("expected command to be an array, got %#v", serviceConfig["command"])
	}

	if len(commandParts) != 4 {
		t.Fatalf("expected 4 command parts, got %d", len(commandParts))
	}

	if commandParts[0] != "npx" {
		t.Fatalf("expected command[0] to be npx, got %#v", commandParts[0])
	}

	environment := mustMapValue(t, serviceConfig["environment"], "mcp.filesystem-service.environment")
	if environment["TOKEN"] != "value" {
		t.Fatalf("expected environment TOKEN value, got %#v", environment["TOKEN"])
	}
}

func TestOpenCodeTargetInstallPreservesUnknownTopLevelKeys(t *testing.T) {
	target := newTestOpenCodeTarget(t)

	initialConfig := map[string]any{
		"theme": "custom-theme",
		"mcp": map[string]any{
			"existing-service": map[string]any{
				"type": "remote",
				"url":  "https://existing.example.com/mcp",
			},
		},
	}

	writeOpenCodeConfigFile(t, target.configPath, initialConfig)

	svc := service.Service{
		Name:      "new-service",
		Transport: "sse",
		URL:       "https://example.com/mcp",
	}

	err := target.Install(svc, nil)
	if err != nil {
		t.Fatalf("expected install to succeed: %v", err)
	}

	updatedConfig := readOpenCodeConfigFile(t, target.configPath)
	if _, ok := updatedConfig["theme"]; !ok {
		t.Fatal("expected unknown top-level key to be preserved")
	}

	mcpEntries := mustMapValue(t, updatedConfig["mcp"], "mcp")
	if _, ok := mcpEntries["existing-service"]; !ok {
		t.Fatal("expected existing service entry to be preserved")
	}

	if _, ok := mcpEntries["new-service"]; !ok {
		t.Fatal("expected new service entry to be added")
	}
}

func TestOpenCodeTargetInstallReturnsErrorForInvalidTransport(t *testing.T) {
	target := newTestOpenCodeTarget(t)

	err := target.Install(service.Service{Name: "demo-service", Transport: "grpc"}, nil)
	if err == nil {
		t.Fatal("expected install to fail for unsupported transport")
	}
}

func TestOpenCodeTargetUninstallRemovesService(t *testing.T) {
	target := newTestOpenCodeTarget(t)

	initialConfig := map[string]any{
		"mcp": map[string]any{
			"service-a": map[string]any{"type": "remote", "url": "https://a.example.com/mcp"},
			"service-b": map[string]any{"type": "remote", "url": "https://b.example.com/mcp"},
		},
	}

	writeOpenCodeConfigFile(t, target.configPath, initialConfig)

	err := target.Uninstall("service-a")
	if err != nil {
		t.Fatalf("expected uninstall to succeed: %v", err)
	}

	updatedConfig := readOpenCodeConfigFile(t, target.configPath)
	mcpEntries := mustMapValue(t, updatedConfig["mcp"], "mcp")

	if _, ok := mcpEntries["service-a"]; ok {
		t.Fatal("expected service-a to be removed")
	}

	if _, ok := mcpEntries["service-b"]; !ok {
		t.Fatal("expected service-b to remain")
	}
}

func TestOpenCodeTargetUninstallIgnoresMissingConfigFile(t *testing.T) {
	target := newTestOpenCodeTarget(t)

	err := target.Uninstall("service-a")
	if err != nil {
		t.Fatalf("expected uninstall to succeed on missing file: %v", err)
	}
}

func TestOpenCodeTargetListReturnsSortedServiceNames(t *testing.T) {
	target := newTestOpenCodeTarget(t)

	initialConfig := map[string]any{
		"mcp": map[string]any{
			"service-b": map[string]any{"type": "remote", "url": "https://b.example.com/mcp"},
			"service-a": map[string]any{"type": "remote", "url": "https://a.example.com/mcp"},
		},
	}

	writeOpenCodeConfigFile(t, target.configPath, initialConfig)

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

func TestOpenCodeTargetListReturnsEmptyWhenConfigMissing(t *testing.T) {
	target := newTestOpenCodeTarget(t)

	services, err := target.List()
	if err != nil {
		t.Fatalf("expected list to succeed: %v", err)
	}

	if len(services) != 0 {
		t.Fatalf("expected no services, got %d", len(services))
	}
}

func TestOpenCodeTargetMethodsErrorWhenMCPIsNotObject(t *testing.T) {
	target := newTestOpenCodeTarget(t)

	writeOpenCodeConfigFile(t, target.configPath, map[string]any{"mcp": "invalid"})

	_, listErr := target.List()
	if listErr == nil {
		t.Fatal("expected list to fail on invalid mcp type")
	}

	uninstallErr := target.Uninstall("service-a")
	if uninstallErr == nil {
		t.Fatal("expected uninstall to fail on invalid mcp type")
	}

	installErr := target.Install(service.Service{Name: "service-a", Transport: "sse", URL: "https://example.com/mcp"}, nil)
	if installErr == nil {
		t.Fatal("expected install to fail on invalid mcp type")
	}
}

func TestOpenCodeTargetCanReadJSONCConfig(t *testing.T) {
	target := newTestOpenCodeTarget(t)
	target.configPath = filepath.Join(t.TempDir(), ".config", "opencode", "opencode.jsonc")

	configDir := filepath.Dir(target.configPath)
	err := os.MkdirAll(configDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	jsoncContent := `{
  // top-level comment
  "theme": "my-theme",
  "mcp": {
    "service-a": {
      "type": "remote",
      "url": "https://example.com/mcp",
    },
  },
}`

	err = os.WriteFile(target.configPath, []byte(jsoncContent), 0o600)
	if err != nil {
		t.Fatalf("failed to write jsonc config: %v", err)
	}

	services, err := target.List()
	if err != nil {
		t.Fatalf("expected list to succeed for jsonc config: %v", err)
	}

	if len(services) != 1 || services[0] != "service-a" {
		t.Fatalf("expected service-a from jsonc config, got %#v", services)
	}
}

func TestOpenCodeTargetCanReadJSONStyleConfigWithJSONCFeatures(t *testing.T) {
	target := newTestOpenCodeTarget(t)
	target.configPath = filepath.Join(t.TempDir(), ".config", "opencode", "opencode.json")

	configDir := filepath.Dir(target.configPath)
	err := os.MkdirAll(configDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	jsonWithCommentsAndTrailingCommas := `{
  // OpenCode allows comments and trailing commas
  "mcp": {
    "service-a": {
      "type": "remote",
      "url": "https://example.com/mcp",
    },
  },
}`

	err = os.WriteFile(target.configPath, []byte(jsonWithCommentsAndTrailingCommas), 0o600)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	services, err := target.List()
	if err != nil {
		t.Fatalf("expected list to succeed for json file with JSONC features: %v", err)
	}

	if len(services) != 1 || services[0] != "service-a" {
		t.Fatalf("expected service-a from json file with JSONC features, got %#v", services)
	}
}

func TestDefaultOpenCodeConfigPathPrefersExistingConfigFile(t *testing.T) {
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		if originalHome == "" {
			_ = os.Unsetenv("HOME")
			return
		}

		_ = os.Setenv("HOME", originalHome)
	})

	tempHome := t.TempDir()
	err := os.Setenv("HOME", tempHome)
	if err != nil {
		t.Fatalf("failed to set HOME: %v", err)
	}

	configDir := filepath.Join(tempHome, ".config", "opencode")
	err = os.MkdirAll(configDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	expectedPath := filepath.Join(configDir, "opencode.jsonc")
	err = os.WriteFile(expectedPath, []byte("{}"), 0o600)
	if err != nil {
		t.Fatalf("failed to create existing config file: %v", err)
	}

	resolvedPath := defaultOpenCodeConfigPath()
	if !strings.HasSuffix(resolvedPath, filepath.Join(".config", "opencode", "opencode.jsonc")) {
		t.Fatalf("expected default path to use existing opencode.jsonc, got %q", resolvedPath)
	}
}

func newTestOpenCodeTarget(t *testing.T) *OpenCodeTarget {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), ".config", "opencode", "opencode.json")
	target := NewOpenCodeTarget()
	target.configPath = configPath
	target.binaryNames = []string{"opencode"}
	target.fallbackBinaryPaths = nil
	target.statPath = os.Stat
	target.lookPath = func(_ string) (string, error) {
		return "", errors.New("not found")
	}

	return target
}

func writeOpenCodeConfigFile(t *testing.T, configPath string, config map[string]any) {
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
		t.Fatalf("failed to write config file: %v", err)
	}
}

func readOpenCodeConfigFile(t *testing.T, configPath string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	config := map[string]any{}
	err = json.Unmarshal(data, &config)
	if err != nil {
		t.Fatalf("failed to unmarshal config file: %v", err)
	}

	return config
}
