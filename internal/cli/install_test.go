package cli

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/andreagrandi/mcp-wire/internal/credential"
	"github.com/andreagrandi/mcp-wire/internal/service"
	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
	"github.com/spf13/cobra"
)

type fakeInstallTarget struct {
	name         string
	slug         string
	installed    bool
	installErr   error
	installCalls int
	lastService  service.Service
	lastEnv      map[string]string
}

type fakeAuthInstallTarget struct {
	*fakeInstallTarget
	authErr         error
	authCalls       int
	lastAuthService string
}

func (t *fakeInstallTarget) Name() string {
	return t.name
}

func (t *fakeInstallTarget) Slug() string {
	return t.slug
}

func (t *fakeInstallTarget) IsInstalled() bool {
	return t.installed
}

func (t *fakeInstallTarget) Install(svc service.Service, resolvedEnv map[string]string) error {
	t.installCalls++
	t.lastService = svc
	t.lastEnv = copyStringMap(resolvedEnv)
	return t.installErr
}

func (t *fakeInstallTarget) Uninstall(_ string) error {
	return nil
}

func (t *fakeInstallTarget) List() ([]string, error) {
	return nil, nil
}

func (t *fakeAuthInstallTarget) Authenticate(serviceName string, _ io.Reader, _ io.Writer, _ io.Writer) error {
	t.authCalls++
	t.lastAuthService = serviceName
	return t.authErr
}

type testCredentialSource struct {
	name   string
	values map[string]string
}

func (s *testCredentialSource) Name() string {
	if s.name == "" {
		return "test-source"
	}

	return s.name
}

func (s *testCredentialSource) Get(envName string) (string, bool) {
	value, ok := s.values[envName]
	return value, ok
}

func (s *testCredentialSource) Store(_ string, _ string) error {
	return nil
}

func TestInstallCommandInstallsToAllInstalledTargetsByDefault(t *testing.T) {
	restore := overrideInstallCommandDependencies(t)
	defer restore()

	alpha := &fakeInstallTarget{name: "Alpha CLI", slug: "alpha-cli", installed: true}
	beta := &fakeInstallTarget{name: "Beta CLI", slug: "beta-cli", installed: true}

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"demo-service": {
				Name:      "demo-service",
				Transport: "sse",
				URL:       "https://example.com/mcp",
				Env: []service.EnvVar{
					{Name: "DEMO_TOKEN", Required: true},
				},
			},
		}, nil
	}
	listInstalledTargets = func() []targetpkg.Target { return []targetpkg.Target{alpha, beta} }
	lookupTarget = func(slug string) (targetpkg.Target, bool) { return nil, false }
	newCredentialEnvSource = func() credential.Source {
		return &testCredentialSource{name: "environment", values: map[string]string{"DEMO_TOKEN": "env-token"}}
	}
	newCredentialFileSource = func(string) credential.Source {
		return &testCredentialSource{name: "file", values: map[string]string{}}
	}

	output, err := executeInstallCommand(t, "demo-service", "--no-prompt")
	if err != nil {
		t.Fatalf("expected install command to succeed: %v", err)
	}

	if alpha.installCalls != 1 || beta.installCalls != 1 {
		t.Fatalf("expected both targets to be installed once, got alpha=%d beta=%d", alpha.installCalls, beta.installCalls)
	}

	if alpha.lastEnv["DEMO_TOKEN"] != "env-token" {
		t.Fatalf("expected resolved env to be passed to alpha target, got %#v", alpha.lastEnv)
	}

	if !strings.Contains(output, "Installing to: Alpha CLI, Beta CLI") {
		t.Fatalf("expected install plan output, got %q", output)
	}

	if !strings.Contains(output, "Alpha CLI: configured") || !strings.Contains(output, "Beta CLI: configured") {
		t.Fatalf("expected success lines in output, got %q", output)
	}
}

func TestInstallCommandUsesSelectedTargets(t *testing.T) {
	restore := overrideInstallCommandDependencies(t)
	defer restore()

	selectedTarget := &fakeInstallTarget{name: "Selected CLI", slug: "selected", installed: true}

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"demo-service": {
				Name:      "demo-service",
				Transport: "sse",
				URL:       "https://example.com/mcp",
			},
		}, nil
	}
	listInstalledTargets = func() []targetpkg.Target {
		t.Fatal("listInstalledTargets should not be called when --target is provided")
		return nil
	}
	lookupTarget = func(slug string) (targetpkg.Target, bool) {
		if slug == "selected" {
			return selectedTarget, true
		}

		return nil, false
	}
	newCredentialEnvSource = func() credential.Source { return &testCredentialSource{values: map[string]string{}} }
	newCredentialFileSource = func(string) credential.Source { return &testCredentialSource{values: map[string]string{}} }

	_, err := executeInstallCommand(t, "demo-service", "--target", "selected", "--no-prompt")
	if err != nil {
		t.Fatalf("expected install command to succeed: %v", err)
	}

	if selectedTarget.installCalls != 1 {
		t.Fatalf("expected selected target to be installed once, got %d", selectedTarget.installCalls)
	}
}

func TestInstallCommandReturnsErrorForUnknownTarget(t *testing.T) {
	restore := overrideInstallCommandDependencies(t)
	defer restore()

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"demo-service": {
				Name:      "demo-service",
				Transport: "sse",
				URL:       "https://example.com/mcp",
			},
		}, nil
	}
	lookupTarget = func(string) (targetpkg.Target, bool) { return nil, false }
	newCredentialEnvSource = func() credential.Source { return &testCredentialSource{values: map[string]string{}} }
	newCredentialFileSource = func(string) credential.Source { return &testCredentialSource{values: map[string]string{}} }

	_, err := executeInstallCommand(t, "demo-service", "--target", "unknown", "--no-prompt")
	if err == nil {
		t.Fatal("expected install command to fail for unknown target")
	}

	if !strings.Contains(err.Error(), "is not known") {
		t.Fatalf("expected unknown target error, got %v", err)
	}
}

func TestInstallCommandReturnsErrorForNotInstalledSelectedTarget(t *testing.T) {
	restore := overrideInstallCommandDependencies(t)
	defer restore()

	notInstalledTarget := &fakeInstallTarget{name: "Offline CLI", slug: "offline", installed: false}

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"demo-service": {
				Name:      "demo-service",
				Transport: "sse",
				URL:       "https://example.com/mcp",
			},
		}, nil
	}
	lookupTarget = func(string) (targetpkg.Target, bool) { return notInstalledTarget, true }
	newCredentialEnvSource = func() credential.Source { return &testCredentialSource{values: map[string]string{}} }
	newCredentialFileSource = func(string) credential.Source { return &testCredentialSource{values: map[string]string{}} }

	_, err := executeInstallCommand(t, "demo-service", "--target", "offline", "--no-prompt")
	if err == nil {
		t.Fatal("expected install command to fail for not installed target")
	}

	if !strings.Contains(err.Error(), "is not installed") {
		t.Fatalf("expected not installed target error, got %v", err)
	}
}

func TestInstallCommandReturnsErrorWhenNoInstalledTargetsAvailable(t *testing.T) {
	restore := overrideInstallCommandDependencies(t)
	defer restore()

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"demo-service": {
				Name:      "demo-service",
				Transport: "sse",
				URL:       "https://example.com/mcp",
			},
		}, nil
	}
	listInstalledTargets = func() []targetpkg.Target { return []targetpkg.Target{} }
	newCredentialEnvSource = func() credential.Source { return &testCredentialSource{values: map[string]string{}} }
	newCredentialFileSource = func(string) credential.Source { return &testCredentialSource{values: map[string]string{}} }

	_, err := executeInstallCommand(t, "demo-service", "--no-prompt")
	if err == nil {
		t.Fatal("expected install command to fail when no targets are installed")
	}

	if !strings.Contains(err.Error(), "no installed targets found") {
		t.Fatalf("expected missing targets error, got %v", err)
	}
}

func TestInstallCommandReturnsErrorWhenServiceIsMissing(t *testing.T) {
	restore := overrideInstallCommandDependencies(t)
	defer restore()

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"available-service": {
				Name:      "available-service",
				Transport: "sse",
				URL:       "https://example.com/mcp",
			},
		}, nil
	}
	listInstalledTargets = func() []targetpkg.Target { return []targetpkg.Target{} }

	_, err := executeInstallCommand(t, "missing-service", "--no-prompt")
	if err == nil {
		t.Fatal("expected install command to fail for unknown service")
	}

	if !strings.Contains(err.Error(), "missing-service") {
		t.Fatalf("expected missing service name in error, got %v", err)
	}
}

func TestInstallCommandReturnsErrorWhenRequiredCredentialIsMissingWithNoPrompt(t *testing.T) {
	restore := overrideInstallCommandDependencies(t)
	defer restore()

	installTarget := &fakeInstallTarget{name: "Alpha CLI", slug: "alpha", installed: true}

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"demo-service": {
				Name:      "demo-service",
				Transport: "sse",
				URL:       "https://example.com/mcp",
				Env: []service.EnvVar{
					{Name: "DEMO_TOKEN", Required: true},
				},
			},
		}, nil
	}
	listInstalledTargets = func() []targetpkg.Target { return []targetpkg.Target{installTarget} }
	newCredentialEnvSource = func() credential.Source { return &testCredentialSource{values: map[string]string{}} }
	newCredentialFileSource = func(string) credential.Source { return &testCredentialSource{values: map[string]string{}} }

	_, err := executeInstallCommand(t, "demo-service", "--no-prompt")
	if err == nil {
		t.Fatal("expected install command to fail when required credential is missing")
	}

	if !strings.Contains(err.Error(), "DEMO_TOKEN") {
		t.Fatalf("expected missing credential name in error, got %v", err)
	}

	if installTarget.installCalls != 0 {
		t.Fatalf("expected target install to not run, got %d calls", installTarget.installCalls)
	}
}

func TestInstallCommandContinuesAfterTargetFailureAndReturnsError(t *testing.T) {
	restore := overrideInstallCommandDependencies(t)
	defer restore()

	successTarget := &fakeInstallTarget{name: "Alpha CLI", slug: "alpha", installed: true}
	failingTarget := &fakeInstallTarget{name: "Beta CLI", slug: "beta", installed: true, installErr: errors.New("write failed")}

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"demo-service": {
				Name:      "demo-service",
				Transport: "sse",
				URL:       "https://example.com/mcp",
			},
		}, nil
	}
	listInstalledTargets = func() []targetpkg.Target {
		return []targetpkg.Target{successTarget, failingTarget}
	}
	newCredentialEnvSource = func() credential.Source { return &testCredentialSource{values: map[string]string{}} }
	newCredentialFileSource = func(string) credential.Source { return &testCredentialSource{values: map[string]string{}} }

	output, err := executeInstallCommand(t, "demo-service", "--no-prompt")
	if err == nil {
		t.Fatal("expected install command to fail when one target install fails")
	}

	if successTarget.installCalls != 1 || failingTarget.installCalls != 1 {
		t.Fatalf("expected both targets to be attempted once, got success=%d failure=%d", successTarget.installCalls, failingTarget.installCalls)
	}

	if !strings.Contains(output, "Alpha CLI: configured") {
		t.Fatalf("expected success output for first target, got %q", output)
	}

	if !strings.Contains(output, "Beta CLI: failed") {
		t.Fatalf("expected failure output for second target, got %q", output)
	}
}

func TestInstallCommandPromptsForServiceWhenArgMissing(t *testing.T) {
	restore := overrideInstallCommandDependencies(t)
	defer restore()

	alpha := &fakeInstallTarget{name: "Alpha CLI", slug: "alpha", installed: true}

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"demo-service": {
				Name:        "demo-service",
				Description: "Demo service",
				Transport:   "sse",
				URL:         "https://example.com/mcp",
			},
		}, nil
	}
	allTargets = func() []targetpkg.Target {
		return []targetpkg.Target{alpha}
	}
	newCredentialEnvSource = func() credential.Source { return &testCredentialSource{values: map[string]string{}} }
	newCredentialFileSource = func(string) credential.Source { return &testCredentialSource{values: map[string]string{}} }

	output, err := executeInstallCommandWithInput(t, "\n1\n1\n\n", "--no-prompt")
	if err != nil {
		t.Fatalf("expected interactive install command to succeed: %v", err)
	}

	if alpha.installCalls != 1 {
		t.Fatalf("expected selected target to be installed once, got %d", alpha.installCalls)
	}

	if !strings.Contains(output, "Equivalent command: mcp-wire install demo-service --target alpha") {
		t.Fatalf("expected equivalent command output, got %q", output)
	}
}

func TestInstallCommandDoesNotPrintNativeOAuthHintForSentry(t *testing.T) {
	restore := overrideInstallCommandDependencies(t)
	defer restore()

	opencodeTarget := &fakeInstallTarget{name: "OpenCode", slug: "opencode", installed: true}

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"sentry": {
				Name:      "sentry",
				Transport: "sse",
				URL:       "https://mcp.sentry.dev/mcp",
			},
		}, nil
	}
	lookupTarget = func(slug string) (targetpkg.Target, bool) {
		if slug == "opencode" {
			return opencodeTarget, true
		}

		return nil, false
	}
	newCredentialEnvSource = func() credential.Source { return &testCredentialSource{values: map[string]string{}} }
	newCredentialFileSource = func(string) credential.Source { return &testCredentialSource{values: map[string]string{}} }

	output, err := executeInstallCommand(t, "sentry", "--target", "opencode", "--no-prompt")
	if err != nil {
		t.Fatalf("expected sentry install to succeed: %v", err)
	}

	if strings.Contains(output, "opencode mcp auth sentry") {
		t.Fatalf("did not expect native OAuth hint for sentry, got %q", output)
	}
}

func TestInstallCommandDoesNotPrintNativeOAuthHintForContext7(t *testing.T) {
	restore := overrideInstallCommandDependencies(t)
	defer restore()

	opencodeTarget := &fakeInstallTarget{name: "OpenCode", slug: "opencode", installed: true}

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"context7": {
				Name:      "context7",
				Transport: "sse",
				URL:       "https://mcp.context7.com/mcp/oauth",
			},
		}, nil
	}
	lookupTarget = func(slug string) (targetpkg.Target, bool) {
		if slug == "opencode" {
			return opencodeTarget, true
		}

		return nil, false
	}
	newCredentialEnvSource = func() credential.Source { return &testCredentialSource{values: map[string]string{}} }
	newCredentialFileSource = func(string) credential.Source { return &testCredentialSource{values: map[string]string{}} }

	output, err := executeInstallCommand(t, "context7", "--target", "opencode", "--no-prompt")
	if err != nil {
		t.Fatalf("expected context7 install to succeed: %v", err)
	}

	if strings.Contains(output, "opencode mcp auth context7") {
		t.Fatalf("did not expect native OAuth hint for context7, got %q", output)
	}
}

func TestInstallCommandPrintsClaudeOAuthHintWhenAutoAuthIsUnsupported(t *testing.T) {
	restore := overrideInstallCommandDependencies(t)
	defer restore()

	claudeTarget := &fakeInstallTarget{name: "Claude Code", slug: "claude", installed: true}

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"jira": {
				Name:      "jira",
				Transport: "sse",
				Auth:      "oauth",
				URL:       "https://mcp.atlassian.com/v1/mcp",
			},
		}, nil
	}
	lookupTarget = func(slug string) (targetpkg.Target, bool) {
		if slug == "claude" {
			return claudeTarget, true
		}

		return nil, false
	}
	newCredentialEnvSource = func() credential.Source { return &testCredentialSource{values: map[string]string{}} }
	newCredentialFileSource = func(string) credential.Source { return &testCredentialSource{values: map[string]string{}} }
	shouldAutoAuthenticate = func(*cobra.Command) bool { return true }

	output, err := executeInstallCommand(t, "jira", "--target", "claude", "--no-prompt")
	if err != nil {
		t.Fatalf("expected jira install to succeed: %v", err)
	}

	if !strings.Contains(output, "Claude Code: complete OAuth in Claude Code with /mcp") {
		t.Fatalf("expected Claude OAuth guidance in output, got %q", output)
	}

	if strings.Contains(output, "authentication skipped") {
		t.Fatalf("expected Claude-specific OAuth hint instead of generic skipped line, got %q", output)
	}
}

func TestInstallCommandKeepsGenericOAuthHintForOtherUnsupportedTargets(t *testing.T) {
	restore := overrideInstallCommandDependencies(t)
	defer restore()

	unsupportedTarget := &fakeInstallTarget{name: "Other CLI", slug: "other", installed: true}

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"sentry": {
				Name:      "sentry",
				Transport: "sse",
				Auth:      "oauth",
				URL:       "https://mcp.sentry.dev/mcp",
			},
		}, nil
	}
	lookupTarget = func(slug string) (targetpkg.Target, bool) {
		if slug == "other" {
			return unsupportedTarget, true
		}

		return nil, false
	}
	newCredentialEnvSource = func() credential.Source { return &testCredentialSource{values: map[string]string{}} }
	newCredentialFileSource = func(string) credential.Source { return &testCredentialSource{values: map[string]string{}} }
	shouldAutoAuthenticate = func(*cobra.Command) bool { return true }

	output, err := executeInstallCommand(t, "sentry", "--target", "other", "--no-prompt")
	if err != nil {
		t.Fatalf("expected sentry install to succeed: %v", err)
	}

	if !strings.Contains(output, "Other CLI: authentication skipped (automatic OAuth is not supported by this target)") {
		t.Fatalf("expected generic OAuth skip output, got %q", output)
	}
}

func TestInstallCommandAutomaticallyAuthenticatesOAuthService(t *testing.T) {
	restore := overrideInstallCommandDependencies(t)
	defer restore()

	authTarget := &fakeAuthInstallTarget{
		fakeInstallTarget: &fakeInstallTarget{name: "Codex CLI", slug: "codex", installed: true},
	}

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"context7": {
				Name:      "context7",
				Transport: "sse",
				Auth:      "oauth",
				URL:       "https://mcp.context7.com/mcp/oauth",
			},
		}, nil
	}
	lookupTarget = func(slug string) (targetpkg.Target, bool) {
		if slug == "codex" {
			return authTarget, true
		}

		return nil, false
	}
	newCredentialEnvSource = func() credential.Source { return &testCredentialSource{values: map[string]string{}} }
	newCredentialFileSource = func(string) credential.Source { return &testCredentialSource{values: map[string]string{}} }
	shouldAutoAuthenticate = func(*cobra.Command) bool { return true }

	output, err := executeInstallCommand(t, "context7", "--target", "codex", "--no-prompt")
	if err != nil {
		t.Fatalf("expected OAuth install to succeed: %v", err)
	}

	if authTarget.installCalls != 1 {
		t.Fatalf("expected install to run once, got %d", authTarget.installCalls)
	}

	if authTarget.authCalls != 1 {
		t.Fatalf("expected authentication to run once, got %d", authTarget.authCalls)
	}

	if authTarget.lastAuthService != "context7" {
		t.Fatalf("expected auth service context7, got %q", authTarget.lastAuthService)
	}

	if !strings.Contains(output, "Codex CLI: authenticated") {
		t.Fatalf("expected authenticated output, got %q", output)
	}
}

func TestInstallCommandReturnsErrorWhenOAuthAuthenticationFails(t *testing.T) {
	restore := overrideInstallCommandDependencies(t)
	defer restore()

	authTarget := &fakeAuthInstallTarget{
		fakeInstallTarget: &fakeInstallTarget{name: "Codex CLI", slug: "codex", installed: true},
		authErr:           errors.New("oauth cancelled"),
	}

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"jira": {
				Name:      "jira",
				Transport: "sse",
				Auth:      "oauth",
				URL:       "https://mcp.atlassian.com/v1/mcp",
			},
		}, nil
	}
	lookupTarget = func(slug string) (targetpkg.Target, bool) {
		if slug == "codex" {
			return authTarget, true
		}

		return nil, false
	}
	newCredentialEnvSource = func() credential.Source { return &testCredentialSource{values: map[string]string{}} }
	newCredentialFileSource = func(string) credential.Source { return &testCredentialSource{values: map[string]string{}} }
	shouldAutoAuthenticate = func(*cobra.Command) bool { return true }

	output, err := executeInstallCommand(t, "jira", "--target", "codex", "--no-prompt")
	if err == nil {
		t.Fatal("expected install command to fail when OAuth authentication fails")
	}

	if authTarget.installCalls != 1 {
		t.Fatalf("expected install to run once, got %d", authTarget.installCalls)
	}

	if authTarget.authCalls != 1 {
		t.Fatalf("expected authentication to run once, got %d", authTarget.authCalls)
	}

	if !strings.Contains(err.Error(), "failed OAuth authentication") {
		t.Fatalf("expected OAuth authentication error, got %v", err)
	}

	if !strings.Contains(output, "Codex CLI: authentication failed") {
		t.Fatalf("expected authentication failure output, got %q", output)
	}
}

func TestInstallCommandSkipsOAuthAuthenticationForNonOAuthService(t *testing.T) {
	restore := overrideInstallCommandDependencies(t)
	defer restore()

	authTarget := &fakeAuthInstallTarget{
		fakeInstallTarget: &fakeInstallTarget{name: "Codex CLI", slug: "codex", installed: true},
	}

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"docs": {
				Name:      "docs",
				Transport: "sse",
				URL:       "https://docs.example.com/mcp",
			},
		}, nil
	}
	lookupTarget = func(slug string) (targetpkg.Target, bool) {
		if slug == "codex" {
			return authTarget, true
		}

		return nil, false
	}
	newCredentialEnvSource = func() credential.Source { return &testCredentialSource{values: map[string]string{}} }
	newCredentialFileSource = func(string) credential.Source { return &testCredentialSource{values: map[string]string{}} }
	shouldAutoAuthenticate = func(*cobra.Command) bool { return true }

	output, err := executeInstallCommand(t, "docs", "--target", "codex", "--no-prompt")
	if err != nil {
		t.Fatalf("expected non-OAuth install to succeed: %v", err)
	}

	if authTarget.authCalls != 0 {
		t.Fatalf("expected no authentication for non-OAuth service, got %d", authTarget.authCalls)
	}

	if strings.Contains(output, "starting OAuth authentication") {
		t.Fatalf("did not expect OAuth authentication output, got %q", output)
	}
}

func overrideInstallCommandDependencies(t *testing.T) func() {
	t.Helper()

	originalLoadServices := loadServices
	originalListInstalledTargets := listInstalledTargets
	originalLookupTarget := lookupTarget
	originalNewCredentialEnvSource := newCredentialEnvSource
	originalNewCredentialFileSource := newCredentialFileSource
	originalNewCredentialResolver := newCredentialResolver
	originalAllTargets := allTargets
	originalShouldAutoAuthenticate := shouldAutoAuthenticate

	return func() {
		loadServices = originalLoadServices
		listInstalledTargets = originalListInstalledTargets
		lookupTarget = originalLookupTarget
		newCredentialEnvSource = originalNewCredentialEnvSource
		newCredentialFileSource = originalNewCredentialFileSource
		newCredentialResolver = originalNewCredentialResolver
		allTargets = originalAllTargets
		shouldAutoAuthenticate = originalShouldAutoAuthenticate
	}
}

func copyStringMap(values map[string]string) map[string]string {
	copyValues := make(map[string]string, len(values))
	for key, value := range values {
		copyValues[key] = value
	}

	return copyValues
}

func executeInstallCommand(t *testing.T, args ...string) (string, error) {
	return executeInstallCommandWithInput(t, "", args...)
}

func executeInstallCommandWithInput(t *testing.T, input string, args ...string) (string, error) {
	t.Helper()

	installCmd := newInstallCmd()
	var stdout, stderr bytes.Buffer

	installCmd.SetOut(&stdout)
	installCmd.SetErr(&stderr)
	installCmd.SetIn(strings.NewReader(input))
	installCmd.SetArgs(args)

	err := installCmd.Execute()
	output := stdout.String() + stderr.String()

	return output, err
}
