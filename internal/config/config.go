package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	configFileName = "config.json"
	configDirName  = "mcp-wire"
)

// FeatureRegistry defines all known feature flags and their defaults.
var FeatureRegistry = map[string]FeatureDefinition{
	"registry": {
		Name:        "registry",
		Description: "Official MCP Registry integration",
		Default:     false,
	},
	"tui": {
		Name:        "tui",
		Description: "Full-screen Bubble Tea terminal UI",
		Default:     false,
	},
}

// FeatureDefinition describes a feature flag.
type FeatureDefinition struct {
	Name        string
	Description string
	Default     bool
}

// Config holds mcp-wire local settings.
type Config struct {
	path     string
	raw      map[string]json.RawMessage
	features map[string]bool
}

// Load reads the config from the default path.
func Load() (*Config, error) {
	return LoadFrom("")
}

// LoadFrom reads the config from the given path.
//
// If path is empty, it defaults to ~/.config/mcp-wire/config.json.
// If the file does not exist, a Config with default values is returned.
func LoadFrom(path string) (*Config, error) {
	resolved := strings.TrimSpace(path)
	if resolved == "" {
		resolved = defaultConfigPath()
	}

	cfg := &Config{
		path:     resolved,
		raw:      make(map[string]json.RawMessage),
		features: make(map[string]bool),
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}

		return nil, fmt.Errorf("read config file %q: %w", resolved, err)
	}

	if err := json.Unmarshal(data, &cfg.raw); err != nil {
		return nil, fmt.Errorf("parse config file %q: %w", resolved, err)
	}

	featuresRaw, ok := cfg.raw["features"]
	if ok {
		var featMap map[string]bool
		if err := json.Unmarshal(featuresRaw, &featMap); err != nil {
			return nil, fmt.Errorf("parse features in config file %q: %w", resolved, err)
		}

		for k, v := range featMap {
			cfg.features[k] = v
		}
	}

	return cfg, nil
}

// IsFeatureEnabled returns whether a feature flag is enabled.
//
// If the feature has not been explicitly set, the registry default is used.
// Unknown feature names always return false.
func (c *Config) IsFeatureEnabled(name string) bool {
	if c == nil {
		return false
	}

	trimmed := strings.TrimSpace(name)

	if val, ok := c.features[trimmed]; ok {
		return val
	}

	if def, ok := FeatureRegistry[trimmed]; ok {
		return def.Default
	}

	return false
}

// SetFeature sets a feature flag value and persists the config.
func (c *Config) SetFeature(name string, enabled bool) error {
	if c == nil {
		return errors.New("config is nil")
	}

	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errors.New("feature name is required")
	}

	if _, ok := FeatureRegistry[trimmed]; !ok {
		return fmt.Errorf("unknown feature %q", trimmed)
	}

	c.features[trimmed] = enabled

	return c.save()
}

// Features returns a sorted list of all known features with their status.
func (c *Config) Features() []FeatureStatus {
	result := make([]FeatureStatus, 0, len(FeatureRegistry))

	for _, def := range FeatureRegistry {
		result = append(result, FeatureStatus{
			Name:        def.Name,
			Description: def.Description,
			Enabled:     c.IsFeatureEnabled(def.Name),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// FeatureStatus describes the current state of a feature flag.
type FeatureStatus struct {
	Name        string
	Description string
	Enabled     bool
}

func (c *Config) save() error {
	configDir := filepath.Dir(c.path)
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return fmt.Errorf("create config directory %q: %w", configDir, err)
	}

	featuresJSON, err := json.Marshal(c.features)
	if err != nil {
		return fmt.Errorf("marshal features: %w", err)
	}

	c.raw["features"] = featuresJSON

	data, err := json.MarshalIndent(c.raw, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	data = append(data, '\n')

	if err := os.WriteFile(c.path, data, 0o644); err != nil {
		return fmt.Errorf("write config file %q: %w", c.path, err)
	}

	return nil
}

func defaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", configDirName, configFileName)
	}

	return filepath.Join(homeDir, ".config", configDirName, configFileName)
}
