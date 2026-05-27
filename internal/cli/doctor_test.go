package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andreagrandi/mcp-wire/internal/config"
	"github.com/andreagrandi/mcp-wire/internal/service"
	"github.com/andreagrandi/mcp-wire/internal/target"
)

type fakeDoctorTarget struct {
	name       string
	slug       string
	installed  bool
	configPath string
}

func (t fakeDoctorTarget) Name() string                                         { return t.name }
func (t fakeDoctorTarget) Slug() string                                         { return t.slug }
func (t fakeDoctorTarget) IsInstalled() bool                                    { return t.installed }
func (t fakeDoctorTarget) Install(_ service.Service, _ map[string]string) error { return nil }
func (t fakeDoctorTarget) Uninstall(_ string) error                             { return nil }
func (t fakeDoctorTarget) List() ([]string, error)                              { return nil, nil }
func (t fakeDoctorTarget) ConfigPath() string                                   { return t.configPath }

func newTestDoctorDeps(t *testing.T, targets []target.Target) doctorDeps {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "config.json")
	credsPath := filepath.Join(t.TempDir(), "credentials")
	servicesDir := filepath.Join(t.TempDir(), "services")
	registryCachePath := filepath.Join(t.TempDir(), "servers.json")

	return doctorDeps{
		loadConfig:        func() (*config.Config, error) { return config.LoadFrom(configPath) },
		allTargets:        func() []target.Target { return targets },
		registryCachePath: func() string { return registryCachePath },
		credentialsPath:   func() string { return credsPath },
		userServicesPath:  func() string { return servicesDir },
		version:           "test-version",
		stat:              os.Stat,
	}
}

func TestDoctorReportsTargetInstallStatus(t *testing.T) {
	targets := []target.Target{
		fakeDoctorTarget{
			name:       "Claude Code",
			slug:       "claude",
			installed:  true,
			configPath: filepath.Join(t.TempDir(), "claude.json"),
		},
		fakeDoctorTarget{
			name:       "Codex CLI",
			slug:       "codex",
			installed:  false,
			configPath: filepath.Join(t.TempDir(), "codex.toml"),
		},
	}

	buf := new(bytes.Buffer)

	if err := runDoctor(buf, newTestDoctorDeps(t, targets)); err != nil {
		t.Fatalf("expected doctor to succeed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "mcp-wire test-version") {
		t.Fatalf("expected version header, got %q", output)
	}

	if !strings.Contains(output, "Claude Code (claude)") {
		t.Fatalf("expected Claude target line, got %q", output)
	}

	if !strings.Contains(output, "Installed:    yes") {
		t.Fatalf("expected installed=yes for Claude, got %q", output)
	}

	if !strings.Contains(output, "Codex CLI (codex)") {
		t.Fatalf("expected Codex target line, got %q", output)
	}

	if !strings.Contains(output, "Installed:    no") {
		t.Fatalf("expected installed=no for Codex, got %q", output)
	}
}

func TestDoctorReportsMissingTargetConfig(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "does-not-exist.json")
	targets := []target.Target{
		fakeDoctorTarget{
			name:       "Claude Code",
			slug:       "claude",
			installed:  true,
			configPath: missingPath,
		},
	}

	buf := new(bytes.Buffer)

	if err := runDoctor(buf, newTestDoctorDeps(t, targets)); err != nil {
		t.Fatalf("expected doctor to succeed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, missingPath) {
		t.Fatalf("expected output to mention config path %q, got %q", missingPath, output)
	}

	if !strings.Contains(output, "Config:       missing") {
		t.Fatalf("expected 'missing' config status, got %q", output)
	}
}

func TestDoctorReportsExistingTargetConfig(t *testing.T) {
	configDir := t.TempDir()
	existingPath := filepath.Join(configDir, "claude.json")
	if err := os.WriteFile(existingPath, []byte("{}"), 0o600); err != nil {
		t.Fatalf("failed to write fixture config: %v", err)
	}

	targets := []target.Target{
		fakeDoctorTarget{
			name:       "Claude Code",
			slug:       "claude",
			installed:  true,
			configPath: existingPath,
		},
	}

	buf := new(bytes.Buffer)

	if err := runDoctor(buf, newTestDoctorDeps(t, targets)); err != nil {
		t.Fatalf("expected doctor to succeed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Config:       exists") {
		t.Fatalf("expected 'exists' config status, got %q", output)
	}
}

func TestDoctorReportsFeatureFlags(t *testing.T) {
	targets := []target.Target{
		fakeDoctorTarget{name: "Claude Code", slug: "claude", installed: true},
	}

	buf := new(bytes.Buffer)

	if err := runDoctor(buf, newTestDoctorDeps(t, targets)); err != nil {
		t.Fatalf("expected doctor to succeed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Feature flags:") {
		t.Fatalf("expected 'Feature flags:' header, got %q", output)
	}

	if !strings.Contains(output, "registry") {
		t.Fatalf("expected 'registry' feature in output, got %q", output)
	}

	if !strings.Contains(output, "disabled") {
		t.Fatalf("expected 'disabled' state for registry, got %q", output)
	}
}

func TestDoctorReportsPathsSection(t *testing.T) {
	targets := []target.Target{
		fakeDoctorTarget{name: "Claude Code", slug: "claude", installed: true},
	}

	deps := newTestDoctorDeps(t, targets)

	credsPath := deps.credentialsPath()
	if err := os.WriteFile(credsPath, []byte(""), 0o600); err != nil {
		t.Fatalf("failed to seed credentials file: %v", err)
	}

	buf := new(bytes.Buffer)

	if err := runDoctor(buf, deps); err != nil {
		t.Fatalf("expected doctor to succeed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Credentials file") || !strings.Contains(output, credsPath) {
		t.Fatalf("expected credentials path %q in output, got %q", credsPath, output)
	}

	if !strings.Contains(output, "User services dir") {
		t.Fatalf("expected 'User services dir' label in output, got %q", output)
	}

	if !strings.Contains(output, "Registry cache") {
		t.Fatalf("expected 'Registry cache' label in output, got %q", output)
	}

	// The credentials file exists but the services dir and registry cache do not.
	if !strings.Contains(output, "missing") {
		t.Fatalf("expected at least one 'missing' status, got %q", output)
	}
}

func TestDoctorHintsListMissingTargetsAndDisabledRegistry(t *testing.T) {
	targets := []target.Target{
		fakeDoctorTarget{name: "Claude Code", slug: "claude", installed: true},
		fakeDoctorTarget{name: "Codex CLI", slug: "codex", installed: false},
	}

	buf := new(bytes.Buffer)

	if err := runDoctor(buf, newTestDoctorDeps(t, targets)); err != nil {
		t.Fatalf("expected doctor to succeed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Hints:") {
		t.Fatalf("expected 'Hints:' section, got %q", output)
	}

	if !strings.Contains(output, "Codex CLI (codex) is not detected") {
		t.Fatalf("expected missing-target hint for Codex, got %q", output)
	}

	if !strings.Contains(output, "Registry feature is disabled") {
		t.Fatalf("expected disabled-registry hint, got %q", output)
	}
}

func TestDoctorHintsHiddenWhenEverythingHealthy(t *testing.T) {
	targets := []target.Target{
		fakeDoctorTarget{name: "Claude Code", slug: "claude", installed: true},
	}

	deps := newTestDoctorDeps(t, targets)

	cfg, err := deps.loadConfig()
	if err != nil {
		t.Fatalf("failed to load test config: %v", err)
	}
	if err := cfg.SetFeature("registry", true); err != nil {
		t.Fatalf("failed to enable registry: %v", err)
	}

	buf := new(bytes.Buffer)

	if err := runDoctor(buf, deps); err != nil {
		t.Fatalf("expected doctor to succeed: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "Hints:") {
		t.Fatalf("expected no Hints section when all targets installed and registry enabled, got %q", output)
	}
}

func TestDoctorHandlesConfigLoadFailure(t *testing.T) {
	targets := []target.Target{
		fakeDoctorTarget{name: "Claude Code", slug: "claude", installed: true},
	}

	deps := newTestDoctorDeps(t, targets)
	deps.loadConfig = func() (*config.Config, error) { return nil, errors.New("boom") }

	buf := new(bytes.Buffer)

	if err := runDoctor(buf, deps); err != nil {
		t.Fatalf("expected doctor to succeed even when config load fails: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "failed to load config") {
		t.Fatalf("expected config load failure surfaced in output, got %q", output)
	}
}

func TestDoctorCommandRegistered(t *testing.T) {
	output, err := executeRootCommand(t, "doctor", "--help")
	if err != nil {
		t.Fatalf("expected doctor --help to succeed: %v", err)
	}

	if !strings.Contains(output, "read-only") {
		t.Fatalf("expected doctor help to mention read-only, got %q", output)
	}
}
