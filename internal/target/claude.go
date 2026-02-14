package target

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/andreagrandi/mcp-wire/internal/service"
)

const (
	claudeCodeBinaryName = "claude"
	claudeCodeSlug       = "claude"
)

// ClaudeCodeTarget manages MCP service configuration for Claude Code.
type ClaudeCodeTarget struct {
	configPath          string
	lookPath            func(file string) (string, error)
	statPath            func(name string) (os.FileInfo, error)
	binaryNames         []string
	fallbackBinaryPaths []string
}

// NewClaudeCodeTarget returns a target instance for Claude Code.
func NewClaudeCodeTarget() *ClaudeCodeTarget {
	return &ClaudeCodeTarget{
		configPath:          defaultClaudeCodeConfigPath(),
		lookPath:            exec.LookPath,
		statPath:            os.Stat,
		binaryNames:         []string{claudeCodeBinaryName, "claude-code"},
		fallbackBinaryPaths: defaultClaudeCodeFallbackBinaryPaths(),
	}
}

// Name returns the target display name.
func (t *ClaudeCodeTarget) Name() string {
	return "Claude Code"
}

// Slug returns the target identifier used in CLI flags.
func (t *ClaudeCodeTarget) Slug() string {
	return claudeCodeSlug
}

// IsInstalled reports whether Claude Code is available via supported install methods.
func (t *ClaudeCodeTarget) IsInstalled() bool {
	binaryNames := t.binaryNames
	if len(binaryNames) == 0 {
		binaryNames = []string{claudeCodeBinaryName}
	}

	for _, binaryName := range binaryNames {
		if strings.TrimSpace(binaryName) == "" {
			continue
		}

		_, err := t.lookPath(binaryName)
		if err == nil {
			return true
		}
	}

	for _, fallbackPath := range t.fallbackBinaryPaths {
		if !isExecutableFilePath(fallbackPath, t.statPath) {
			continue
		}

		return true
	}

	return false
}

// Install writes or updates the service configuration in the target config.
func (t *ClaudeCodeTarget) Install(svc service.Service, resolvedEnv map[string]string) error {
	return t.InstallWithScope(svc, resolvedEnv, ConfigScopeUser)
}

// SupportedScopes returns the scopes supported by Claude Code target operations.
func (t *ClaudeCodeTarget) SupportedScopes() []ConfigScope {
	return []ConfigScope{ConfigScopeUser, ConfigScopeProject, ConfigScopeEffective}
}

// InstallWithScope writes or updates the service configuration in the requested scope.
func (t *ClaudeCodeTarget) InstallWithScope(svc service.Service, resolvedEnv map[string]string, scope ConfigScope) error {
	serviceName := strings.TrimSpace(svc.Name)
	if serviceName == "" {
		return errors.New("service name is required")
	}

	config, _, err := t.readConfig()
	if err != nil {
		return err
	}

	mcpServers, err := getMCPServers(config, scope, true)
	if err != nil {
		return err
	}

	serverConfig, err := buildClaudeCodeServerConfig(svc, resolvedEnv)
	if err != nil {
		return err
	}

	mcpServers[serviceName] = serverConfig

	return t.writeConfig(config)
}

// Uninstall removes a service from the target config.
func (t *ClaudeCodeTarget) Uninstall(serviceName string) error {
	return t.UninstallWithScope(serviceName, ConfigScopeUser)
}

// UninstallWithScope removes a service from the requested scope.
func (t *ClaudeCodeTarget) UninstallWithScope(serviceName string, scope ConfigScope) error {
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

	mcpServers, err := getMCPServers(config, scope, false)
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
func (t *ClaudeCodeTarget) List() ([]string, error) {
	return t.ListWithScope(ConfigScopeEffective)
}

// ListWithScope returns configured service names from the requested scope.
func (t *ClaudeCodeTarget) ListWithScope(scope ConfigScope) ([]string, error) {
	config, exists, err := t.readConfig()
	if err != nil {
		return nil, err
	}

	if !exists {
		return []string{}, nil
	}

	serviceNames := make(map[string]struct{})

	switch scope {
	case ConfigScopeUser:
		if err := collectClaudeMCPServerNamesFromScope(config, serviceNames, "mcpServers"); err != nil {
			return nil, err
		}
	case ConfigScopeProject:
		projectMCPServers, err := getClaudeProjectMCPServers(config, false)
		if err != nil {
			return nil, err
		}

		if err := collectClaudeMCPServerNamesFromMCPServers(projectMCPServers, serviceNames); err != nil {
			return nil, err
		}
	case ConfigScopeEffective:
		if err := collectClaudeMCPServerNamesFromScope(config, serviceNames, "mcpServers"); err != nil {
			return nil, err
		}

		projectMCPServers, err := getClaudeProjectMCPServers(config, false)
		if err != nil {
			return nil, err
		}

		if err := collectClaudeMCPServerNamesFromMCPServers(projectMCPServers, serviceNames); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported scope %q", scope)
	}

	services := make([]string, 0, len(serviceNames))
	for serviceName := range serviceNames {
		services = append(services, serviceName)
	}

	sort.Strings(services)

	return services, nil
}

func (t *ClaudeCodeTarget) readConfig() (map[string]any, bool, error) {
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

func (t *ClaudeCodeTarget) writeConfig(config map[string]any) error {
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

func defaultClaudeCodeConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".claude.json"
	}

	candidates := []string{
		filepath.Join(homeDir, ".claude.json"),
		filepath.Join(homeDir, ".claude", "settings.json"),
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

func defaultClaudeCodeFallbackBinaryPaths() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	return []string{
		filepath.Join(homeDir, ".claude", "local", "claude"),
		filepath.Join(homeDir, ".claude", "local", "claude.cmd"),
		filepath.Join(homeDir, ".claude", "local", "node_modules", ".bin", "claude"),
		filepath.Join(homeDir, ".claude", "local", "node_modules", ".bin", "claude.cmd"),
	}
}

func isExecutableFilePath(path string, statPath func(name string) (os.FileInfo, error)) bool {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return false
	}

	stat := statPath
	if stat == nil {
		stat = os.Stat
	}

	info, err := stat(trimmedPath)
	if err != nil {
		return false
	}

	if info.IsDir() {
		return false
	}

	if runtime.GOOS == "windows" {
		return true
	}

	return info.Mode().Perm()&0o111 != 0
}

func getMCPServers(config map[string]any, scope ConfigScope, createIfMissing bool) (map[string]any, error) {
	switch scope {
	case ConfigScopeUser:
		return getClaudeUserMCPServers(config, createIfMissing)
	case ConfigScopeProject:
		return getClaudeProjectMCPServers(config, createIfMissing)
	default:
		return nil, fmt.Errorf("unsupported scope %q", scope)
	}
}

func getClaudeUserMCPServers(config map[string]any, createIfMissing bool) (map[string]any, error) {
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

func collectClaudeMCPServerNamesFromScope(scope map[string]any, serviceNames map[string]struct{}, label string) error {
	rawMCPServers, exists := scope["mcpServers"]
	if !exists || rawMCPServers == nil {
		return nil
	}

	mcpServers, ok := rawMCPServers.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid config: %s must be an object", label)
	}

	for serviceName := range mcpServers {
		trimmedName := strings.TrimSpace(serviceName)
		if trimmedName == "" {
			continue
		}

		serviceNames[trimmedName] = struct{}{}
	}

	return nil
}

func collectClaudeMCPServerNamesFromMCPServers(mcpServers map[string]any, serviceNames map[string]struct{}) error {
	if mcpServers == nil {
		return nil
	}

	for serviceName := range mcpServers {
		trimmedName := strings.TrimSpace(serviceName)
		if trimmedName == "" {
			continue
		}

		serviceNames[trimmedName] = struct{}{}
	}

	return nil
}

func getClaudeProjectMCPServers(config map[string]any, createIfMissing bool) (map[string]any, error) {
	rawProjects, hasProjects := config["projects"]
	if !hasProjects || rawProjects == nil {
		return nil, nil
	}

	projects, ok := rawProjects.(map[string]any)
	if !ok {
		return nil, errors.New("invalid config: projects must be an object")
	}

	projectKey, err := resolveClaudeProjectKey(projects, createIfMissing)
	if err != nil {
		return nil, err
	}

	if projectKey == "" {
		return nil, nil
	}

	rawProjectConfig, exists := projects[projectKey]
	if !exists || rawProjectConfig == nil {
		if !createIfMissing {
			return nil, nil
		}

		projectConfig := map[string]any{}
		projects[projectKey] = projectConfig
		rawProjectConfig = projectConfig
	}

	projectConfig, ok := rawProjectConfig.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid config: projects[%q] must be an object", projectKey)
	}

	rawMCPServers, exists := projectConfig["mcpServers"]
	if !exists || rawMCPServers == nil {
		if !createIfMissing {
			return nil, nil
		}

		mcpServers := map[string]any{}
		projectConfig["mcpServers"] = mcpServers
		return mcpServers, nil
	}

	mcpServers, ok := rawMCPServers.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid config: projects[%q].mcpServers must be an object", projectKey)
	}

	return mcpServers, nil
}

func resolveClaudeProjectKey(projects map[string]any, createIfMissing bool) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		if createIfMissing {
			return "", fmt.Errorf("resolve current working directory: %w", err)
		}

		return "", nil
	}

	currentDirectory := normalizePathForMatch(cwd)

	bestKey := ""
	bestLength := -1

	for key := range projects {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}

		projectPath := normalizePathForMatch(trimmedKey)
		if projectPath == "" {
			continue
		}

		if currentDirectory == projectPath {
			return key, nil
		}

		if !isPathAtOrWithin(currentDirectory, projectPath) {
			continue
		}

		if len(projectPath) <= bestLength {
			continue
		}

		bestKey = key
		bestLength = len(projectPath)
	}

	if bestKey != "" {
		return bestKey, nil
	}

	if !createIfMissing {
		return "", nil
	}

	cwdPath, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve current working directory: %w", err)
	}

	currentWorkingDirectory := filepath.Clean(cwdPath)
	if _, exists := projects[currentWorkingDirectory]; !exists {
		projects[currentWorkingDirectory] = map[string]any{}
	}

	return currentWorkingDirectory, nil
}

func normalizePathForMatch(path string) string {
	cleanPath := filepath.Clean(path)
	if cleanPath == "" {
		return cleanPath
	}

	absPath, err := filepath.Abs(cleanPath)
	if err == nil {
		cleanPath = absPath
	}

	resolvedPath, err := filepath.EvalSymlinks(cleanPath)
	if err == nil {
		cleanPath = resolvedPath
	}

	return filepath.Clean(cleanPath)
}

func isPathAtOrWithin(path string, parent string) bool {
	relativePath, err := filepath.Rel(parent, path)
	if err != nil {
		return false
	}

	if relativePath == "." {
		return true
	}

	if relativePath == ".." {
		return false
	}

	return !strings.HasPrefix(relativePath, ".."+string(os.PathSeparator))
}

func buildClaudeCodeServerConfig(svc service.Service, resolvedEnv map[string]string) (map[string]any, error) {
	transport := strings.ToLower(strings.TrimSpace(svc.Transport))
	if transport == "" {
		return nil, errors.New("service transport is required")
	}

	serverConfig := map[string]any{
		"type": transport,
	}

	switch transport {
	case "sse":
		url := strings.TrimSpace(svc.URL)
		if url == "" {
			return nil, errors.New("sse service requires url")
		}

		serverConfig["url"] = url
	case "stdio":
		command := strings.TrimSpace(svc.Command)
		if command == "" {
			return nil, errors.New("stdio service requires command")
		}

		serverConfig["command"] = command
		if len(svc.Args) > 0 {
			serverConfig["args"] = svc.Args
		}
	default:
		return nil, fmt.Errorf("unsupported transport %q", svc.Transport)
	}

	env := normalizeResolvedEnv(resolvedEnv)
	if len(env) > 0 {
		serverConfig["env"] = env
	}

	return serverConfig, nil
}

func normalizeResolvedEnv(resolvedEnv map[string]string) map[string]string {
	env := make(map[string]string, len(resolvedEnv))

	for name, value := range resolvedEnv {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" {
			continue
		}

		env[trimmedName] = value
	}

	return env
}
