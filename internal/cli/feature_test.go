package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andreagrandi/mcp-wire/internal/config"
)

func withTestConfig(t *testing.T) func() {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.json")
	original := loadConfig

	loadConfig = func() (*config.Config, error) {
		return config.LoadFrom(configPath)
	}

	return func() {
		loadConfig = original
	}
}

func TestFeatureEnableCommand(t *testing.T) {
	cleanup := withTestConfig(t)
	defer cleanup()

	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"feature", "enable", "registry"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected command to succeed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "enabled") {
		t.Fatalf("expected output to contain 'enabled', got %q", output)
	}

	if !strings.Contains(output, "registry") {
		t.Fatalf("expected output to contain 'registry', got %q", output)
	}
}

func TestFeatureDisableCommand(t *testing.T) {
	cleanup := withTestConfig(t)
	defer cleanup()

	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"feature", "disable", "registry"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected command to succeed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "disabled") {
		t.Fatalf("expected output to contain 'disabled', got %q", output)
	}
}

func TestFeatureListCommand(t *testing.T) {
	cleanup := withTestConfig(t)
	defer cleanup()

	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"feature", "list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected command to succeed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "registry") {
		t.Fatalf("expected output to contain 'registry', got %q", output)
	}

	if !strings.Contains(output, "disabled") {
		t.Fatalf("expected output to show 'disabled' for registry, got %q", output)
	}
}

func TestFeatureEnableUnknownFeature(t *testing.T) {
	cleanup := withTestConfig(t)
	defer cleanup()

	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"feature", "enable", "nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown feature")
	}
}

func TestSetFeatureFlagFunction(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	original := loadConfig
	defer func() { loadConfig = original }()

	loadConfig = func() (*config.Config, error) {
		return config.LoadFrom(configPath)
	}

	buf := new(bytes.Buffer)

	if err := setFeatureFlag(buf, "registry", true); err != nil {
		t.Fatalf("expected enable to succeed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "enabled") {
		t.Fatalf("expected 'enabled' in output, got %q", output)
	}

	buf.Reset()

	if err := setFeatureFlag(buf, "registry", false); err != nil {
		t.Fatalf("expected disable to succeed: %v", err)
	}

	output = buf.String()
	if !strings.Contains(output, "disabled") {
		t.Fatalf("expected 'disabled' in output, got %q", output)
	}
}

func TestListFeaturesFunction(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	original := loadConfig
	defer func() { loadConfig = original }()

	loadConfig = func() (*config.Config, error) {
		return config.LoadFrom(configPath)
	}

	buf := new(bytes.Buffer)

	if err := listFeatures(buf); err != nil {
		t.Fatalf("expected list to succeed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Feature flags:") {
		t.Fatalf("expected header in output, got %q", output)
	}

	if !strings.Contains(output, "registry") {
		t.Fatalf("expected 'registry' in output, got %q", output)
	}
}
