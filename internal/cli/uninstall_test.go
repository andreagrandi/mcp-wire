package cli

import (
	"bytes"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andreagrandi/mcp-wire/internal/credential"
	"github.com/andreagrandi/mcp-wire/internal/service"
	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
	"github.com/spf13/cobra"
)

type fakeUninstallTarget struct {
	name           string
	slug           string
	installed      bool
	uninstallErr   error
	uninstallCalls int
	lastService    string
}

func (t *fakeUninstallTarget) Name() string {
	return t.name
}

func (t *fakeUninstallTarget) Slug() string {
	return t.slug
}

func (t *fakeUninstallTarget) IsInstalled() bool {
	return t.installed
}

func (t *fakeUninstallTarget) Install(_ service.Service, _ map[string]string) error {
	return nil
}

func (t *fakeUninstallTarget) Uninstall(serviceName string) error {
	t.uninstallCalls++
	t.lastService = serviceName
	return t.uninstallErr
}

func (t *fakeUninstallTarget) List() ([]string, error) {
	return nil, nil
}

func TestUninstallCommandUninstallsFromAllInstalledTargetsByDefault(t *testing.T) {
	restore := overrideUninstallCommandDependencies(t)
	defer restore()

	alpha := &fakeUninstallTarget{name: "Alpha CLI", slug: "alpha", installed: true}
	beta := &fakeUninstallTarget{name: "Beta CLI", slug: "beta", installed: true}

	listInstalledTargets = func() []targetpkg.Target {
		return []targetpkg.Target{alpha, beta}
	}

	output, err := executeUninstallCommand(t, "demo-service")
	if err != nil {
		t.Fatalf("expected uninstall command to succeed: %v", err)
	}

	if alpha.uninstallCalls != 1 || beta.uninstallCalls != 1 {
		t.Fatalf("expected both targets to uninstall once, got alpha=%d beta=%d", alpha.uninstallCalls, beta.uninstallCalls)
	}

	if alpha.lastService != "demo-service" || beta.lastService != "demo-service" {
		t.Fatalf("expected service name demo-service to be passed through, got alpha=%q beta=%q", alpha.lastService, beta.lastService)
	}

	if !strings.Contains(output, "Uninstalling from: Alpha CLI, Beta CLI") {
		t.Fatalf("expected uninstall plan output, got %q", output)
	}

	if !strings.Contains(output, "Alpha CLI: removed") || !strings.Contains(output, "Beta CLI: removed") {
		t.Fatalf("expected remove output per target, got %q", output)
	}
}

func TestUninstallCommandUsesSelectedTargets(t *testing.T) {
	restore := overrideUninstallCommandDependencies(t)
	defer restore()

	selected := &fakeUninstallTarget{name: "Selected CLI", slug: "selected", installed: true}

	lookupTarget = func(slug string) (targetpkg.Target, bool) {
		if slug == "selected" {
			return selected, true
		}

		return nil, false
	}

	_, err := executeUninstallCommand(t, "demo-service", "--target", "selected")
	if err != nil {
		t.Fatalf("expected uninstall command to succeed: %v", err)
	}

	if selected.uninstallCalls != 1 {
		t.Fatalf("expected selected target to uninstall once, got %d", selected.uninstallCalls)
	}
}

func TestUninstallCommandReturnsErrorForUnknownTarget(t *testing.T) {
	restore := overrideUninstallCommandDependencies(t)
	defer restore()

	lookupTarget = func(string) (targetpkg.Target, bool) { return nil, false }

	_, err := executeUninstallCommand(t, "demo-service", "--target", "unknown")
	if err == nil {
		t.Fatal("expected uninstall command to fail for unknown target")
	}

	if !strings.Contains(err.Error(), "is not known") {
		t.Fatalf("expected unknown target error, got %v", err)
	}
}

func TestUninstallCommandReturnsErrorWhenNoInstalledTargetsAvailable(t *testing.T) {
	restore := overrideUninstallCommandDependencies(t)
	defer restore()

	listInstalledTargets = func() []targetpkg.Target { return []targetpkg.Target{} }

	_, err := executeUninstallCommand(t, "demo-service")
	if err == nil {
		t.Fatal("expected uninstall command to fail when no installed targets are found")
	}

	if !strings.Contains(err.Error(), "no installed targets found") {
		t.Fatalf("expected no-targets error, got %v", err)
	}
}

func TestUninstallCommandContinuesAfterTargetFailureAndReturnsError(t *testing.T) {
	restore := overrideUninstallCommandDependencies(t)
	defer restore()

	success := &fakeUninstallTarget{name: "Alpha CLI", slug: "alpha", installed: true}
	failure := &fakeUninstallTarget{name: "Beta CLI", slug: "beta", installed: true, uninstallErr: errors.New("write failed")}

	listInstalledTargets = func() []targetpkg.Target {
		return []targetpkg.Target{success, failure}
	}

	output, err := executeUninstallCommand(t, "demo-service")
	if err == nil {
		t.Fatal("expected uninstall command to fail when one target fails")
	}

	if success.uninstallCalls != 1 || failure.uninstallCalls != 1 {
		t.Fatalf("expected both targets to be attempted once, got success=%d failure=%d", success.uninstallCalls, failure.uninstallCalls)
	}

	if !strings.Contains(output, "Alpha CLI: removed") {
		t.Fatalf("expected success output for first target, got %q", output)
	}

	if !strings.Contains(output, "Beta CLI: failed") {
		t.Fatalf("expected failure output for second target, got %q", output)
	}
}

func TestServiceEnvNamesDeduplicatesNames(t *testing.T) {
	envNames := serviceEnvNames(service.Service{
		Env: []service.EnvVar{
			{Name: "TOKEN_A"},
			{Name: "TOKEN_B"},
			{Name: "TOKEN_A"},
			{Name: " "},
		},
	})

	if len(envNames) != 2 {
		t.Fatalf("expected 2 unique names, got %#v", envNames)
	}

	if envNames[0] != "TOKEN_A" || envNames[1] != "TOKEN_B" {
		t.Fatalf("unexpected env name order/content: %#v", envNames)
	}
}

func TestRemoveStoredCredentialsRemovesMatchingKeys(t *testing.T) {
	credentialsPath := filepath.Join(t.TempDir(), "credentials")
	fileSource := credential.NewFileSource(credentialsPath)

	if err := fileSource.Store("TOKEN_A", "value-a"); err != nil {
		t.Fatalf("failed storing TOKEN_A: %v", err)
	}

	if err := fileSource.Store("TOKEN_B", "value-b"); err != nil {
		t.Fatalf("failed storing TOKEN_B: %v", err)
	}

	removedCount, err := removeStoredCredentials(fileSource, []string{"TOKEN_A", "TOKEN_C"})
	if err != nil {
		t.Fatalf("expected credential removal to succeed: %v", err)
	}

	if removedCount != 1 {
		t.Fatalf("expected one credential removal, got %d", removedCount)
	}

	if _, found := fileSource.Get("TOKEN_A"); found {
		t.Fatal("expected TOKEN_A to be removed")
	}

	if value, found := fileSource.Get("TOKEN_B"); !found || value != "value-b" {
		t.Fatalf("expected TOKEN_B to remain, got %q (found=%v)", value, found)
	}
}

func TestMaybeRemoveStoredCredentialsSkipsWhenNotInteractive(t *testing.T) {
	restore := overrideUninstallCommandDependencies(t)
	defer restore()

	isTerminalReader = func(io.Reader) bool { return false }

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&bytes.Buffer{})

	err := maybeRemoveStoredCredentials(cmd, "demo-service")
	if err != nil {
		t.Fatalf("expected non-interactive cleanup to skip without error: %v", err)
	}
}

func executeUninstallCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()

	uninstallCmd := newUninstallCmd()
	var stdout, stderr bytes.Buffer

	uninstallCmd.SetOut(&stdout)
	uninstallCmd.SetErr(&stderr)
	uninstallCmd.SetIn(strings.NewReader(""))
	uninstallCmd.SetArgs(args)

	err := uninstallCmd.Execute()
	output := stdout.String() + stderr.String()

	return output, err
}

func overrideUninstallCommandDependencies(t *testing.T) func() {
	t.Helper()

	originalLoadServices := loadServices
	originalListInstalledTargets := listInstalledTargets
	originalLookupTarget := lookupTarget
	originalNewCredentialFileSourceForCleanup := newCredentialFileSourceForCleanup
	originalIsTerminalReader := isTerminalReader

	return func() {
		loadServices = originalLoadServices
		listInstalledTargets = originalListInstalledTargets
		lookupTarget = originalLookupTarget
		newCredentialFileSourceForCleanup = originalNewCredentialFileSourceForCleanup
		isTerminalReader = originalIsTerminalReader
	}
}
