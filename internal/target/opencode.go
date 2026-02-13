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
	"github.com/tidwall/jsonc"
)

const (
	openCodeBinaryName = "opencode"
	openCodeSlug       = "opencode"
)

// OpenCodeTarget manages MCP service configuration for OpenCode.
type OpenCodeTarget struct {
	configPath          string
	lookPath            func(file string) (string, error)
	statPath            func(name string) (os.FileInfo, error)
	binaryNames         []string
	fallbackBinaryPaths []string
}

// NewOpenCodeTarget returns a target instance for OpenCode.
func NewOpenCodeTarget() *OpenCodeTarget {
	return &OpenCodeTarget{
		configPath:          defaultOpenCodeConfigPath(),
		lookPath:            exec.LookPath,
		statPath:            os.Stat,
		binaryNames:         []string{openCodeBinaryName},
		fallbackBinaryPaths: defaultOpenCodeFallbackBinaryPaths(),
	}
}

// Name returns the target display name.
func (t *OpenCodeTarget) Name() string {
	return "OpenCode"
}

// Slug returns the target identifier used in CLI flags.
func (t *OpenCodeTarget) Slug() string {
	return openCodeSlug
}

// IsInstalled reports whether OpenCode is available via supported install methods.
func (t *OpenCodeTarget) IsInstalled() bool {
	binaryNames := t.binaryNames
	if len(binaryNames) == 0 {
		binaryNames = []string{openCodeBinaryName}
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
func (t *OpenCodeTarget) Install(svc service.Service, resolvedEnv map[string]string) error {
	serviceName := strings.TrimSpace(svc.Name)
	if serviceName == "" {
		return errors.New("service name is required")
	}

	config, _, err := t.readConfig()
	if err != nil {
		return err
	}

	mcpDefinitions, err := getOpenCodeMCPEntries(config, true)
	if err != nil {
		return err
	}

	serverConfig, err := buildOpenCodeServerConfig(svc, resolvedEnv)
	if err != nil {
		return err
	}

	mcpDefinitions[serviceName] = serverConfig

	return t.writeConfig(config)
}

// Uninstall removes a service from the target config.
func (t *OpenCodeTarget) Uninstall(serviceName string) error {
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

	mcpDefinitions, err := getOpenCodeMCPEntries(config, false)
	if err != nil {
		return err
	}

	if mcpDefinitions == nil {
		return nil
	}

	delete(mcpDefinitions, trimmedServiceName)

	return t.writeConfig(config)
}

// List returns configured service names from the target config.
func (t *OpenCodeTarget) List() ([]string, error) {
	config, exists, err := t.readConfig()
	if err != nil {
		return nil, err
	}

	if !exists {
		return []string{}, nil
	}

	mcpDefinitions, err := getOpenCodeMCPEntries(config, false)
	if err != nil {
		return nil, err
	}

	if mcpDefinitions == nil {
		return []string{}, nil
	}

	services := make([]string, 0, len(mcpDefinitions))
	for serviceName := range mcpDefinitions {
		services = append(services, serviceName)
	}

	sort.Strings(services)

	return services, nil
}

func (t *OpenCodeTarget) readConfig() (map[string]any, bool, error) {
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

	// OpenCode supports both JSON and JSONC-style content (comments/trailing commas)
	// in user config files, including files named with a .json extension.
	data = jsonc.ToJSON(data)

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, true, fmt.Errorf("parse config file %q: %w", t.configPath, err)
	}

	return config, true, nil
}

func (t *OpenCodeTarget) writeConfig(config map[string]any) error {
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

func defaultOpenCodeConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "opencode", "opencode.json")
	}

	configDir := filepath.Join(homeDir, ".config", "opencode")
	candidates := []string{
		filepath.Join(configDir, "opencode.json"),
		filepath.Join(configDir, "opencode.jsonc"),
		filepath.Join(configDir, "config.json"),
	}

	for _, candidatePath := range candidates {
		info, err := os.Stat(candidatePath)
		if err != nil {
			continue
		}

		if info.IsDir() {
			continue
		}

		return candidatePath
	}

	return candidates[0]
}

func defaultOpenCodeFallbackBinaryPaths() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	return []string{
		filepath.Join(homeDir, ".opencode", "bin", "opencode"),
		filepath.Join(homeDir, ".opencode", "bin", "opencode.exe"),
	}
}

func getOpenCodeMCPEntries(config map[string]any, createIfMissing bool) (map[string]any, error) {
	rawMCPEntries, exists := config["mcp"]
	if !exists || rawMCPEntries == nil {
		if !createIfMissing {
			return nil, nil
		}

		mcpEntries := map[string]any{}
		config["mcp"] = mcpEntries

		return mcpEntries, nil
	}

	mcpEntries, ok := rawMCPEntries.(map[string]any)
	if !ok {
		return nil, errors.New("invalid config: mcp must be an object")
	}

	return mcpEntries, nil
}

func buildOpenCodeServerConfig(svc service.Service, resolvedEnv map[string]string) (map[string]any, error) {
	transport := strings.ToLower(strings.TrimSpace(svc.Transport))
	if transport == "" {
		return nil, errors.New("service transport is required")
	}

	serverConfig := map[string]any{
		"enabled": true,
	}

	switch transport {
	case "sse":
		url := strings.TrimSpace(svc.URL)
		if url == "" {
			return nil, errors.New("sse service requires url")
		}

		serverConfig["type"] = "remote"
		serverConfig["url"] = url

		headers := normalizeResolvedEnv(resolvedEnv)
		if len(headers) > 0 {
			serverConfig["headers"] = headers
		}
	case "stdio":
		command := strings.TrimSpace(svc.Command)
		if command == "" {
			return nil, errors.New("stdio service requires command")
		}

		serverConfig["type"] = "local"

		commandParts := make([]string, 0, len(svc.Args)+1)
		commandParts = append(commandParts, command)
		commandParts = append(commandParts, svc.Args...)
		serverConfig["command"] = commandParts

		environment := normalizeResolvedEnv(resolvedEnv)
		if len(environment) > 0 {
			serverConfig["environment"] = environment
		}
	default:
		return nil, fmt.Errorf("unsupported transport %q", svc.Transport)
	}

	return serverConfig, nil
}
