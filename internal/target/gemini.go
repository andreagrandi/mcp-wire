package target

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/andreagrandi/mcp-wire/internal/service"
)

const (
	geminiBinaryName = "gemini"
	geminiSlug       = "gemini"
)

// GeminiCLITarget manages MCP service configuration for Gemini CLI.
type GeminiCLITarget struct {
	configPath          string
	lookPath            func(file string) (string, error)
	statPath            func(name string) (os.FileInfo, error)
	binaryNames         []string
	fallbackBinaryPaths []string
}

// NewGeminiCLITarget returns a target instance for Gemini CLI.
func NewGeminiCLITarget() *GeminiCLITarget {
	return &GeminiCLITarget{
		configPath:          defaultGeminiCLIConfigPath(),
		lookPath:            exec.LookPath,
		statPath:            os.Stat,
		binaryNames:         []string{geminiBinaryName},
		fallbackBinaryPaths: nil,
	}
}

// Name returns the target display name.
func (t *GeminiCLITarget) Name() string {
	return "Gemini CLI"
}

// Slug returns the target identifier used in CLI flags.
func (t *GeminiCLITarget) Slug() string {
	return geminiSlug
}

// IsInstalled reports whether Gemini CLI is available via supported install methods.
func (t *GeminiCLITarget) IsInstalled() bool {
	binaryNames := t.binaryNames
	if len(binaryNames) == 0 {
		binaryNames = []string{geminiBinaryName}
	}

	for _, binaryName := range binaryNames {
		if strings.TrimSpace(binaryName) == "" {
			continue
		}

		if _, err := t.lookPath(binaryName); err == nil {
			return true
		}
	}

	for _, fallbackPath := range t.fallbackBinaryPaths {
		if isExecutableFilePath(fallbackPath, t.statPath) {
			return true
		}
	}

	return false
}

// Install writes or updates the service configuration in the target config.
func (t *GeminiCLITarget) Install(svc service.Service, resolvedEnv map[string]string) error {
	serviceName := strings.TrimSpace(svc.Name)
	if serviceName == "" {
		return errors.New("service name is required")
	}

	config, _, err := t.readConfig()
	if err != nil {
		return err
	}

	mcpServers, err := getGeminiMCPServers(config, true)
	if err != nil {
		return err
	}

	serverConfig, err := buildGeminiCLIServerConfig(svc, resolvedEnv)
	if err != nil {
		return err
	}

	mcpServers[serviceName] = serverConfig

	return t.writeConfig(config)
}

// Uninstall removes a service from the target config.
func (t *GeminiCLITarget) Uninstall(serviceName string) error {
	trimmedServiceName := strings.TrimSpace(serviceName)
	if trimmedServiceName == "" {
		return errors.New("service name is required")
	}

	config, exists, err := t.readConfig()
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	mcpServers, err := getGeminiMCPServers(config, false)
	if err != nil {
		return err
	}

	if mcpServers == nil {
		return nil
	}

	delete(mcpServers, trimmedServiceName)

	return t.writeConfig(config)
}

// List returns configured service names from the target config.
func (t *GeminiCLITarget) List() ([]string, error) {
	config, exists, err := t.readConfig()
	if err != nil {
		return nil, err
	}

	if !exists {
		return []string{}, nil
	}

	mcpServers, err := getGeminiMCPServers(config, false)
	if err != nil {
		return nil, err
	}

	if mcpServers == nil {
		return []string{}, nil
	}

	services := make([]string, 0, len(mcpServers))
	for serviceName := range mcpServers {
		services = append(services, serviceName)
	}

	sort.Strings(services)

	return services, nil
}

func (t *GeminiCLITarget) readConfig() (map[string]any, bool, error) {
	data, err := os.ReadFile(t.configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]any{}, false, nil
		}

		return nil, false, fmt.Errorf("read config file %q: %w", t.configPath, err)
	}

	config := map[string]any{}
	if len(bytes.TrimSpace(data)) == 0 {
		return config, true, nil
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, true, fmt.Errorf("parse config file %q: %w", t.configPath, err)
	}

	return config, true, nil
}

func (t *GeminiCLITarget) writeConfig(config map[string]any) error {
	configDir := filepath.Dir(t.configPath)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("create config directory %q: %w", configDir, err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize config file %q: %w", t.configPath, err)
	}

	data = append(data, '\n')

	if err := os.WriteFile(t.configPath, data, 0o600); err != nil {
		return fmt.Errorf("write config file %q: %w", t.configPath, err)
	}

	return nil
}

func defaultGeminiCLIConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".gemini", "settings.json")
	}

	return filepath.Join(homeDir, ".gemini", "settings.json")
}

func getGeminiMCPServers(config map[string]any, createIfMissing bool) (map[string]any, error) {
	rawMCPServers, exists := config["mcpServers"]
	if !exists || rawMCPServers == nil {
		if !createIfMissing {
			return nil, nil
		}

		mcpServers := map[string]any{}
		config["mcpServers"] = mcpServers

		return mcpServers, nil
	}

	mcpServers, ok := rawMCPServers.(map[string]any)
	if !ok {
		return nil, errors.New("invalid config: mcpServers must be an object")
	}

	return mcpServers, nil
}

func buildGeminiCLIServerConfig(svc service.Service, resolvedEnv map[string]string) (map[string]any, error) {
	transport := strings.ToLower(strings.TrimSpace(svc.Transport))
	if transport == "" {
		return nil, errors.New("service transport is required")
	}

	serverConfig := map[string]any{}

	switch transport {
	case "sse":
		url := strings.TrimSpace(svc.URL)
		if url == "" {
			return nil, errors.New("sse service requires url")
		}

		serverConfig["url"] = url

		headers := geminiHeaders(svc, resolvedEnv)
		if len(headers) > 0 {
			serverConfig["headers"] = headers
		}
	case "http":
		url := strings.TrimSpace(svc.URL)
		if url == "" {
			return nil, errors.New("http service requires url")
		}

		serverConfig["httpUrl"] = url

		headers := geminiHeaders(svc, resolvedEnv)
		if len(headers) > 0 {
			serverConfig["headers"] = headers
		}
	case "stdio":
		command := strings.TrimSpace(svc.Command)
		if command == "" {
			return nil, errors.New("stdio service requires command")
		}

		serverConfig["command"] = command
		if len(svc.Args) > 0 {
			serverConfig["args"] = svc.Args
		}

		env := normalizeResolvedEnv(resolvedEnv)
		if len(env) > 0 {
			serverConfig["env"] = env
		}
	default:
		return nil, fmt.Errorf("unsupported transport %q", svc.Transport)
	}

	return serverConfig, nil
}

func geminiHeaders(svc service.Service, resolvedEnv map[string]string) map[string]string {
	if svc.Headers != nil {
		return svc.Headers
	}

	return normalizeResolvedEnv(resolvedEnv)
}
