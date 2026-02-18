package target

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/andreagrandi/mcp-wire/internal/service"
)

func TestGeminiCLITargetMetadata(t *testing.T) {
	target := NewGeminiCLITarget()

	if target.Name() != "Gemini CLI" {
		t.Fatalf("expected target name Gemini CLI, got %q", target.Name())
	}

	if target.Slug() != "gemini" {
		t.Fatalf("expected target slug gemini, got %q", target.Slug())
	}
}

func TestGeminiCLITargetIsInstalledTrueWhenBinaryFound(t *testing.T) {
	target := newTestGeminiCLITarget(t)
	target.lookPath = func(file string) (string, error) {
		if file != "gemini" {
			return "", errors.New("not found")
		}

		return "/usr/local/bin/gemini", nil
	}

	if !target.IsInstalled() {
		t.Fatal("expected target to be reported as installed")
	}
}

func TestGeminiCLITargetIsInstalledFalseWhenBinaryNotFound(t *testing.T) {
	target := newTestGeminiCLITarget(t)
	target.lookPath = func(_ string) (string, error) {
		return "", errors.New("not found")
	}

	if target.IsInstalled() {
		t.Fatal("expected target to be reported as not installed")
	}
}

func TestGeminiCLITargetInstallCreatesConfigAndAddsSSEService(t *testing.T) {
	target := newTestGeminiCLITarget(t)

	svc := service.Service{
		Name:      "demo-sse-service",
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

	config := readGeminiConfigFile(t, target.configPath)
	mcpServers := mustMapValue(t, config["mcpServers"], "mcpServers")
	serviceConfig := mustMapValue(t, mcpServers["demo-sse-service"], "mcpServers.demo-sse-service")

	if serviceConfig["url"] != "https://example.com/sse" {
		t.Fatalf("expected url to be set, got %#v", serviceConfig["url"])
	}

	if _, ok := serviceConfig["type"]; ok {
		t.Fatal("expected no type field in gemini config")
	}

	headers := mustMapValue(t, serviceConfig["headers"], "mcpServers.demo-sse-service.headers")
	if headers["DEMO_TOKEN"] != "token-value" {
		t.Fatalf("expected header DEMO_TOKEN token-value, got %#v", headers["DEMO_TOKEN"])
	}
}

func TestGeminiCLITargetInstallCreatesConfigAndAddsHTTPService(t *testing.T) {
	target := newTestGeminiCLITarget(t)

	svc := service.Service{
		Name:      "demo-http-service",
		Transport: "http",
		URL:       "https://example.com/mcp",
	}

	resolvedEnv := map[string]string{
		"API_KEY": "key-value",
	}

	err := target.Install(svc, resolvedEnv)
	if err != nil {
		t.Fatalf("expected install to succeed: %v", err)
	}

	config := readGeminiConfigFile(t, target.configPath)
	mcpServers := mustMapValue(t, config["mcpServers"], "mcpServers")
	serviceConfig := mustMapValue(t, mcpServers["demo-http-service"], "mcpServers.demo-http-service")

	if serviceConfig["httpUrl"] != "https://example.com/mcp" {
		t.Fatalf("expected httpUrl to be set, got %#v", serviceConfig["httpUrl"])
	}

	if _, ok := serviceConfig["url"]; ok {
		t.Fatal("expected no url field for http transport in gemini config")
	}

	if _, ok := serviceConfig["type"]; ok {
		t.Fatal("expected no type field in gemini config")
	}

	headers := mustMapValue(t, serviceConfig["headers"], "mcpServers.demo-http-service.headers")
	if headers["API_KEY"] != "key-value" {
		t.Fatalf("expected header API_KEY key-value, got %#v", headers["API_KEY"])
	}
}

func TestGeminiCLITargetInstallUsesExplicitHeaders(t *testing.T) {
	target := newTestGeminiCLITarget(t)

	svc := service.Service{
		Name:      "registry-service",
		Transport: "sse",
		URL:       "https://example.com/sse",
		Headers: map[string]string{
			"Authorization": "Bearer my-token",
			"X-Static":      "static-value",
		},
	}

	resolvedEnv := map[string]string{
		"tenant": "acme",
	}

	err := target.Install(svc, resolvedEnv)
	if err != nil {
		t.Fatalf("expected install to succeed: %v", err)
	}

	config := readGeminiConfigFile(t, target.configPath)
	mcpServers := mustMapValue(t, config["mcpServers"], "mcpServers")
	serviceConfig := mustMapValue(t, mcpServers["registry-service"], "mcpServers.registry-service")

	headers := mustMapValue(t, serviceConfig["headers"], "mcpServers.registry-service.headers")
	if headers["Authorization"] != "Bearer my-token" {
		t.Fatalf("expected Authorization header, got %#v", headers["Authorization"])
	}

	if headers["X-Static"] != "static-value" {
		t.Fatalf("expected X-Static header, got %#v", headers["X-Static"])
	}

	if _, ok := headers["tenant"]; ok {
		t.Fatal("expected tenant env var not to appear in headers when svc.Headers is set")
	}
}

func TestGeminiCLITargetInstallNoHeadersWhenBothEmpty(t *testing.T) {
	target := newTestGeminiCLITarget(t)

	svc := service.Service{
		Name:      "no-header-service",
		Transport: "sse",
		URL:       "https://example.com/sse",
	}

	err := target.Install(svc, nil)
	if err != nil {
		t.Fatalf("expected install to succeed: %v", err)
	}

	config := readGeminiConfigFile(t, target.configPath)
	mcpServers := mustMapValue(t, config["mcpServers"], "mcpServers")
	serviceConfig := mustMapValue(t, mcpServers["no-header-service"], "mcpServers.no-header-service")

	if _, ok := serviceConfig["headers"]; ok {
		t.Fatal("expected no headers key when both svc.Headers and resolvedEnv are empty")
	}
}

func TestGeminiCLITargetInstallWritesStdioServiceConfiguration(t *testing.T) {
	target := newTestGeminiCLITarget(t)

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

	config := readGeminiConfigFile(t, target.configPath)
	mcpServers := mustMapValue(t, config["mcpServers"], "mcpServers")
	serviceConfig := mustMapValue(t, mcpServers["filesystem-service"], "mcpServers.filesystem-service")

	if serviceConfig["command"] != "npx" {
		t.Fatalf("expected command to be npx, got %#v", serviceConfig["command"])
	}

	args, ok := serviceConfig["args"].([]any)
	if !ok {
		t.Fatalf("expected args to be an array, got %#v", serviceConfig["args"])
	}

	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(args))
	}

	if args[0] != "-y" || args[1] != "demo-command" || args[2] != "--flag" {
		t.Fatalf("expected args [-y demo-command --flag], got %#v", args)
	}

	env := mustMapValue(t, serviceConfig["env"], "mcpServers.filesystem-service.env")
	if env["TOKEN"] != "value" {
		t.Fatalf("expected env TOKEN value, got %#v", env["TOKEN"])
	}

	if _, ok := serviceConfig["type"]; ok {
		t.Fatal("expected no type field in gemini config")
	}
}

func TestGeminiCLITargetInstallPreservesUnknownTopLevelKeys(t *testing.T) {
	target := newTestGeminiCLITarget(t)

	initialConfig := map[string]any{
		"theme": "custom-theme",
		"mcpServers": map[string]any{
			"existing-service": map[string]any{
				"url": "https://existing.example.com/sse",
			},
		},
	}

	writeGeminiConfigFile(t, target.configPath, initialConfig)

	svc := service.Service{
		Name:      "new-service",
		Transport: "sse",
		URL:       "https://example.com/sse",
	}

	err := target.Install(svc, nil)
	if err != nil {
		t.Fatalf("expected install to succeed: %v", err)
	}

	updatedConfig := readGeminiConfigFile(t, target.configPath)
	if _, ok := updatedConfig["theme"]; !ok {
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

func TestGeminiCLITargetInstallReturnsErrorForEmptyServiceName(t *testing.T) {
	target := newTestGeminiCLITarget(t)

	err := target.Install(service.Service{Name: "", Transport: "sse", URL: "https://example.com/sse"}, nil)
	if err == nil {
		t.Fatal("expected install to fail for empty service name")
	}
}

func TestGeminiCLITargetInstallReturnsErrorForMissingURL(t *testing.T) {
	target := newTestGeminiCLITarget(t)

	err := target.Install(service.Service{Name: "demo-service", Transport: "sse"}, nil)
	if err == nil {
		t.Fatal("expected install to fail for missing url")
	}
}

func TestGeminiCLITargetInstallReturnsErrorForMissingCommand(t *testing.T) {
	target := newTestGeminiCLITarget(t)

	err := target.Install(service.Service{Name: "demo-service", Transport: "stdio"}, nil)
	if err == nil {
		t.Fatal("expected install to fail for missing command")
	}
}

func TestGeminiCLITargetInstallReturnsErrorForInvalidTransport(t *testing.T) {
	target := newTestGeminiCLITarget(t)

	err := target.Install(service.Service{Name: "demo-service", Transport: "grpc"}, nil)
	if err == nil {
		t.Fatal("expected install to fail for unsupported transport")
	}
}

func TestGeminiCLITargetUninstallRemovesService(t *testing.T) {
	target := newTestGeminiCLITarget(t)

	initialConfig := map[string]any{
		"mcpServers": map[string]any{
			"service-a": map[string]any{"url": "https://a.example.com/sse"},
			"service-b": map[string]any{"url": "https://b.example.com/sse"},
		},
	}

	writeGeminiConfigFile(t, target.configPath, initialConfig)

	err := target.Uninstall("service-a")
	if err != nil {
		t.Fatalf("expected uninstall to succeed: %v", err)
	}

	updatedConfig := readGeminiConfigFile(t, target.configPath)
	mcpServers := mustMapValue(t, updatedConfig["mcpServers"], "mcpServers")

	if _, ok := mcpServers["service-a"]; ok {
		t.Fatal("expected service-a to be removed")
	}

	if _, ok := mcpServers["service-b"]; !ok {
		t.Fatal("expected service-b to remain")
	}
}

func TestGeminiCLITargetUninstallIgnoresMissingConfigFile(t *testing.T) {
	target := newTestGeminiCLITarget(t)

	err := target.Uninstall("service-a")
	if err != nil {
		t.Fatalf("expected uninstall to succeed on missing file: %v", err)
	}
}

func TestGeminiCLITargetUninstallReturnsErrorForEmptyServiceName(t *testing.T) {
	target := newTestGeminiCLITarget(t)

	err := target.Uninstall("")
	if err == nil {
		t.Fatal("expected uninstall to fail for empty service name")
	}
}

func TestGeminiCLITargetListReturnsSortedServiceNames(t *testing.T) {
	target := newTestGeminiCLITarget(t)

	initialConfig := map[string]any{
		"mcpServers": map[string]any{
			"service-b": map[string]any{"url": "https://b.example.com/sse"},
			"service-a": map[string]any{"url": "https://a.example.com/sse"},
		},
	}

	writeGeminiConfigFile(t, target.configPath, initialConfig)

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

func TestGeminiCLITargetListReturnsEmptyWhenConfigMissing(t *testing.T) {
	target := newTestGeminiCLITarget(t)

	services, err := target.List()
	if err != nil {
		t.Fatalf("expected list to succeed: %v", err)
	}

	if len(services) != 0 {
		t.Fatalf("expected no services, got %d", len(services))
	}
}

func TestGeminiCLITargetMethodsErrorWhenMCPServersIsNotObject(t *testing.T) {
	target := newTestGeminiCLITarget(t)

	writeGeminiConfigFile(t, target.configPath, map[string]any{"mcpServers": "invalid"})

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

func newTestGeminiCLITarget(t *testing.T) *GeminiCLITarget {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), ".gemini", "settings.json")
	target := NewGeminiCLITarget()
	target.configPath = configPath
	target.binaryNames = []string{"gemini"}
	target.fallbackBinaryPaths = nil
	target.statPath = os.Stat
	target.lookPath = func(_ string) (string, error) {
		return "", errors.New("not found")
	}

	return target
}

func writeGeminiConfigFile(t *testing.T, configPath string, config map[string]any) {
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

func readGeminiConfigFile(t *testing.T, configPath string) map[string]any {
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
