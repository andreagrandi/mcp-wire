package target

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/andreagrandi/mcp-wire/internal/service"
	toml "github.com/pelletier/go-toml/v2"
)

const (
	codexBinaryName = "codex"
	codexSlug       = "codex"
)

// CodexTarget manages MCP service configuration for Codex CLI.
type CodexTarget struct {
	configPath string
	lookPath   func(file string) (string, error)
}

// NewCodexTarget returns a target instance for Codex CLI.
func NewCodexTarget() *CodexTarget {
	return &CodexTarget{
		configPath: defaultCodexConfigPath(),
		lookPath:   exec.LookPath,
	}
}

// Name returns the target display name.
func (t *CodexTarget) Name() string {
	return "Codex CLI"
}

// Slug returns the target identifier used in CLI flags.
func (t *CodexTarget) Slug() string {
	return codexSlug
}

// IsInstalled reports whether Codex CLI is available in PATH.
func (t *CodexTarget) IsInstalled() bool {
	_, err := t.lookPath(codexBinaryName)
	return err == nil
}

// Install writes or updates the service configuration in the target config.
func (t *CodexTarget) Install(svc service.Service, resolvedEnv map[string]string) error {
	serviceName := strings.TrimSpace(svc.Name)
	if serviceName == "" {
		return errors.New("service name is required")
	}

	config, _, err := t.readConfig()
	if err != nil {
		return err
	}

	mcpServers, err := getCodexMCPServers(config, true)
	if err != nil {
		return err
	}

	serverConfig, err := buildCodexServerConfig(svc, resolvedEnv)
	if err != nil {
		return err
	}

	mcpServers[serviceName] = serverConfig

	return t.writeConfig(config)
}

// Uninstall removes a service from the target config.
func (t *CodexTarget) Uninstall(serviceName string) error {
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

	mcpServers, err := getCodexMCPServers(config, false)
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
func (t *CodexTarget) List() ([]string, error) {
	config, exists, err := t.readConfig()
	if err != nil {
		return nil, err
	}

	if !exists {
		return []string{}, nil
	}

	mcpServers, err := getCodexMCPServers(config, false)
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

func (t *CodexTarget) readConfig() (map[string]any, bool, error) {
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

	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, true, fmt.Errorf("parse config file %q: %w", t.configPath, err)
	}

	return config, true, nil
}

func (t *CodexTarget) writeConfig(config map[string]any) error {
	configDir := filepath.Dir(t.configPath)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("create config directory %q: %w", configDir, err)
	}

	data, err := toml.Marshal(config)
	if err != nil {
		return fmt.Errorf("serialize config file %q: %w", t.configPath, err)
	}

	data = append(data, '\n')

	if err := os.WriteFile(t.configPath, data, 0o600); err != nil {
		return fmt.Errorf("write config file %q: %w", t.configPath, err)
	}

	return nil
}

func defaultCodexConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".codex", "config.toml")
	}

	return filepath.Join(homeDir, ".codex", "config.toml")
}

func getCodexMCPServers(config map[string]any, createIfMissing bool) (map[string]any, error) {
	rawMCPServers, exists := config["mcp_servers"]
	if !exists || rawMCPServers == nil {
		if !createIfMissing {
			return nil, nil
		}

		mcpServers := map[string]any{}
		config["mcp_servers"] = mcpServers

		return mcpServers, nil
	}

	mcpServers, ok := rawMCPServers.(map[string]any)
	if !ok {
		return nil, errors.New("invalid config: mcp_servers must be a table")
	}

	return mcpServers, nil
}

func buildCodexServerConfig(svc service.Service, resolvedEnv map[string]string) (map[string]any, error) {
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

		bearerEnvVar := pickBearerEnvVar(svc, resolvedEnv)
		if bearerEnvVar != "" {
			serverConfig["bearer_token_env_var"] = bearerEnvVar
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

func pickBearerEnvVar(svc service.Service, resolvedEnv map[string]string) string {
	for _, envVar := range svc.Env {
		name := strings.TrimSpace(envVar.Name)
		if name == "" {
			continue
		}

		if _, exists := resolvedEnv[name]; exists {
			return name
		}
	}

	envNames := make([]string, 0, len(resolvedEnv))
	for name := range resolvedEnv {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" {
			continue
		}

		envNames = append(envNames, trimmedName)
	}

	if len(envNames) == 0 {
		return ""
	}

	sort.Strings(envNames)

	return envNames[0]
}
