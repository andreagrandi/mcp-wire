package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromReturnsDefaultsWhenFileMissing(t *testing.T) {
	cfg, err := LoadFrom(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatalf("expected load to succeed: %v", err)
	}

	if cfg.IsFeatureEnabled("registry") {
		t.Fatal("expected registry feature to be disabled by default")
	}
}

func TestLoadFromReadsExistingConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	content := `{"features":{"registry":true}}`

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadFrom(configPath)
	if err != nil {
		t.Fatalf("expected load to succeed: %v", err)
	}

	if !cfg.IsFeatureEnabled("registry") {
		t.Fatal("expected registry feature to be enabled")
	}
}

func TestLoadFromReturnsErrorOnInvalidJSON(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")

	if err := os.WriteFile(configPath, []byte("{not json}"), 0o644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := LoadFrom(configPath)
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestLoadFromReturnsErrorOnInvalidFeaturesType(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	content := `{"features":"not-a-map"}`

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := LoadFrom(configPath)
	if err == nil {
		t.Fatal("expected error on invalid features type")
	}
}

func TestSetFeatureEnableAndDisable(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg, err := LoadFrom(configPath)
	if err != nil {
		t.Fatalf("expected load to succeed: %v", err)
	}

	if err := cfg.SetFeature("registry", true); err != nil {
		t.Fatalf("expected enable to succeed: %v", err)
	}

	if !cfg.IsFeatureEnabled("registry") {
		t.Fatal("expected registry to be enabled after SetFeature")
	}

	if err := cfg.SetFeature("registry", false); err != nil {
		t.Fatalf("expected disable to succeed: %v", err)
	}

	if cfg.IsFeatureEnabled("registry") {
		t.Fatal("expected registry to be disabled after SetFeature(false)")
	}
}

func TestSetFeaturePersistsToDisk(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg, err := LoadFrom(configPath)
	if err != nil {
		t.Fatalf("expected load to succeed: %v", err)
	}

	if err := cfg.SetFeature("registry", true); err != nil {
		t.Fatalf("expected set to succeed: %v", err)
	}

	reloaded, err := LoadFrom(configPath)
	if err != nil {
		t.Fatalf("expected reload to succeed: %v", err)
	}

	if !reloaded.IsFeatureEnabled("registry") {
		t.Fatal("expected registry to remain enabled after reload")
	}
}

func TestSetFeatureCreatesDirectoryAndFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "nested", "dir", "config.json")
	cfg, err := LoadFrom(configPath)
	if err != nil {
		t.Fatalf("expected load to succeed: %v", err)
	}

	if err := cfg.SetFeature("registry", true); err != nil {
		t.Fatalf("expected set to succeed: %v", err)
	}

	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config file to be created: %v", err)
	}
}

func TestSetFeatureRejectsUnknownFeature(t *testing.T) {
	cfg, err := LoadFrom(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatalf("expected load to succeed: %v", err)
	}

	err = cfg.SetFeature("nonexistent", true)
	if err == nil {
		t.Fatal("expected error for unknown feature")
	}
}

func TestSetFeatureRejectsEmptyName(t *testing.T) {
	cfg, err := LoadFrom(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatalf("expected load to succeed: %v", err)
	}

	err = cfg.SetFeature("  ", true)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestSetFeatureRejectsNilConfig(t *testing.T) {
	var cfg *Config

	err := cfg.SetFeature("registry", true)
	if err == nil {
		t.Fatal("expected error on nil config")
	}
}

func TestIsFeatureEnabledReturnsFalseOnNilConfig(t *testing.T) {
	var cfg *Config

	if cfg.IsFeatureEnabled("registry") {
		t.Fatal("expected nil config to return false")
	}
}

func TestIsFeatureEnabledReturnsFalseForUnknownFeature(t *testing.T) {
	cfg, err := LoadFrom(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatalf("expected load to succeed: %v", err)
	}

	if cfg.IsFeatureEnabled("nonexistent") {
		t.Fatal("expected unknown feature to return false")
	}
}

func TestFeaturesReturnsSortedList(t *testing.T) {
	cfg, err := LoadFrom(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatalf("expected load to succeed: %v", err)
	}

	features := cfg.Features()
	if len(features) == 0 {
		t.Fatal("expected at least one feature")
	}

	found := false
	for _, f := range features {
		if f.Name == "registry" {
			found = true
			if f.Enabled {
				t.Fatal("expected registry to be disabled by default")
			}

			if f.Description == "" {
				t.Fatal("expected registry to have a description")
			}
		}
	}

	if !found {
		t.Fatal("expected registry feature in list")
	}
}

func TestFeaturesReflectsSetState(t *testing.T) {
	cfg, err := LoadFrom(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatalf("expected load to succeed: %v", err)
	}

	if err := cfg.SetFeature("registry", true); err != nil {
		t.Fatalf("expected set to succeed: %v", err)
	}

	features := cfg.Features()
	for _, f := range features {
		if f.Name == "registry" && !f.Enabled {
			t.Fatal("expected registry to show as enabled")
		}
	}
}

func TestConfigPreservesValidJSON(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg, err := LoadFrom(configPath)
	if err != nil {
		t.Fatalf("expected load to succeed: %v", err)
	}

	if err := cfg.SetFeature("registry", true); err != nil {
		t.Fatalf("expected set to succeed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config file to be readable: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("expected valid JSON on disk: %v", err)
	}

	features, ok := parsed["features"].(map[string]any)
	if !ok {
		t.Fatal("expected features key in JSON")
	}

	registry, ok := features["registry"].(bool)
	if !ok || !registry {
		t.Fatal("expected registry=true in JSON")
	}
}

func TestLoadUsesDefaultPath(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load() to succeed: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected non-nil config from Load()")
	}
}

func TestSetFeaturePreservesUnknownTopLevelKeys(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	content := `{"custom_setting":"keep-me","features":{"registry":false}}`

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadFrom(configPath)
	if err != nil {
		t.Fatalf("expected load to succeed: %v", err)
	}

	if err := cfg.SetFeature("registry", true); err != nil {
		t.Fatalf("expected set to succeed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config file to be readable: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("expected valid JSON on disk: %v", err)
	}

	customVal, ok := parsed["custom_setting"].(string)
	if !ok || customVal != "keep-me" {
		t.Fatalf("expected custom_setting to be preserved, got %v", parsed["custom_setting"])
	}

	features, ok := parsed["features"].(map[string]any)
	if !ok {
		t.Fatal("expected features key in JSON")
	}

	registry, ok := features["registry"].(bool)
	if !ok || !registry {
		t.Fatal("expected registry=true in JSON")
	}
}
