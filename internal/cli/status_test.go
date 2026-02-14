package cli

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/andreagrandi/mcp-wire/internal/service"
	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
)

type fakeStatusTarget struct {
	name      string
	slug      string
	installed bool
	services  []string
	listErr   error
}

type fakeScopedStatusTarget struct {
	name            string
	slug            string
	installed       bool
	listErr         error
	servicesByScope map[targetpkg.ConfigScope][]string
	lastScope       targetpkg.ConfigScope
}

func (t fakeStatusTarget) Name() string {
	return t.name
}

func (t fakeStatusTarget) Slug() string {
	return t.slug
}

func (t fakeStatusTarget) IsInstalled() bool {
	return t.installed
}

func (t fakeStatusTarget) Install(_ service.Service, _ map[string]string) error {
	return nil
}

func (t fakeStatusTarget) Uninstall(_ string) error {
	return nil
}

func (t fakeStatusTarget) List() ([]string, error) {
	if t.listErr != nil {
		return nil, t.listErr
	}

	copyValues := make([]string, len(t.services))
	copy(copyValues, t.services)
	return copyValues, nil
}

func (t *fakeScopedStatusTarget) Name() string {
	return t.name
}

func (t *fakeScopedStatusTarget) Slug() string {
	return t.slug
}

func (t *fakeScopedStatusTarget) IsInstalled() bool {
	return t.installed
}

func (t *fakeScopedStatusTarget) Install(_ service.Service, _ map[string]string) error {
	return nil
}

func (t *fakeScopedStatusTarget) Uninstall(_ string) error {
	return nil
}

func (t *fakeScopedStatusTarget) List() ([]string, error) {
	values := t.servicesByScope[targetpkg.ConfigScopeEffective]
	copyValues := make([]string, len(values))
	copy(copyValues, values)
	return copyValues, t.listErr
}

func (t *fakeScopedStatusTarget) SupportedScopes() []targetpkg.ConfigScope {
	return []targetpkg.ConfigScope{targetpkg.ConfigScopeUser, targetpkg.ConfigScopeProject, targetpkg.ConfigScopeEffective}
}

func (t *fakeScopedStatusTarget) InstallWithScope(_ service.Service, _ map[string]string, _ targetpkg.ConfigScope) error {
	return nil
}

func (t *fakeScopedStatusTarget) UninstallWithScope(_ string, _ targetpkg.ConfigScope) error {
	return nil
}

func (t *fakeScopedStatusTarget) ListWithScope(scope targetpkg.ConfigScope) ([]string, error) {
	t.lastScope = scope
	if t.listErr != nil {
		return nil, t.listErr
	}

	values := t.servicesByScope[scope]
	copyValues := make([]string, len(values))
	copy(copyValues, values)
	return copyValues, nil
}

func TestStatusCommandPrintsServiceTargetMatrix(t *testing.T) {
	restore := overrideStatusDependencies(t)
	defer restore()

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"service-a": {Name: "service-a", Transport: "sse", URL: "https://example.com/a"},
			"service-b": {Name: "service-b", Transport: "sse", URL: "https://example.com/b"},
			"service-c": {Name: "service-c", Transport: "sse", URL: "https://example.com/c"},
		}, nil
	}

	listInstalledTargets = func() []targetpkg.Target {
		return []targetpkg.Target{
			fakeStatusTarget{name: "Beta CLI", slug: "beta", installed: true, services: []string{"service-b", "unknown-service"}},
			fakeStatusTarget{name: "Alpha CLI", slug: "alpha", installed: true, services: []string{"service-a"}},
		}
	}

	output, err := executeRootCommand(t, "status")
	if err != nil {
		t.Fatalf("expected status command to succeed: %v", err)
	}

	if !strings.Contains(output, "Status (") {
		t.Fatalf("expected status heading, got %q", output)
	}

	if !strings.Contains(output, "Alpha CLI") || !strings.Contains(output, "Beta CLI") {
		t.Fatalf("expected target headers in output, got %q", output)
	}

	assertLineMatches(t, output, `(?m)^\s*service-a\s+yes\s+no\s*$`)
	assertLineMatches(t, output, `(?m)^\s*service-b\s+no\s+yes\s*$`)
	assertLineMatches(t, output, `(?m)^\s*service-c\s+no\s+no\s*$`)
}

func TestStatusCommandPrintsNoInstalledTargetsMessage(t *testing.T) {
	restore := overrideStatusDependencies(t)
	defer restore()

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"service-a": {Name: "service-a", Transport: "sse", URL: "https://example.com/a"},
		}, nil
	}

	listInstalledTargets = func() []targetpkg.Target { return []targetpkg.Target{} }

	output, err := executeRootCommand(t, "status")
	if err != nil {
		t.Fatalf("expected status command to succeed: %v", err)
	}

	if !strings.Contains(output, "(no installed targets found)") {
		t.Fatalf("expected no-targets message, got %q", output)
	}
}

func TestStatusCommandPrintsNoServicesMessage(t *testing.T) {
	restore := overrideStatusDependencies(t)
	defer restore()

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{}, nil
	}

	listInstalledTargets = func() []targetpkg.Target {
		return []targetpkg.Target{fakeStatusTarget{name: "Alpha CLI", slug: "alpha", installed: true}}
	}

	output, err := executeRootCommand(t, "status")
	if err != nil {
		t.Fatalf("expected status command to succeed: %v", err)
	}

	if !strings.Contains(output, "(no services found)") {
		t.Fatalf("expected no-services message, got %q", output)
	}
}

func TestStatusCommandReturnsLoadServicesError(t *testing.T) {
	restore := overrideStatusDependencies(t)
	defer restore()

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return nil, errors.New("loader failed")
	}

	listInstalledTargets = func() []targetpkg.Target { return []targetpkg.Target{} }

	_, err := executeRootCommand(t, "status")
	if err == nil {
		t.Fatal("expected status command to fail when service loading fails")
	}

	if !strings.Contains(err.Error(), "load services") {
		t.Fatalf("expected wrapped service loading error, got %v", err)
	}
}

func TestStatusCommandReturnsTargetListError(t *testing.T) {
	restore := overrideStatusDependencies(t)
	defer restore()

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"service-a": {Name: "service-a", Transport: "sse", URL: "https://example.com/a"},
		}, nil
	}

	listInstalledTargets = func() []targetpkg.Target {
		return []targetpkg.Target{fakeStatusTarget{name: "Alpha CLI", slug: "alpha", installed: true, listErr: errors.New("list failed")}}
	}

	_, err := executeRootCommand(t, "status")
	if err == nil {
		t.Fatal("expected status command to fail when target listing fails")
	}

	if !strings.Contains(err.Error(), "target \"alpha\"") {
		t.Fatalf("expected target-specific error, got %v", err)
	}
}

func TestStatusCommandUsesScopeAwareListWhenScopeIsRequested(t *testing.T) {
	restore := overrideStatusDependencies(t)
	defer restore()

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"service-a": {Name: "service-a", Transport: "sse", URL: "https://example.com/a"},
			"service-b": {Name: "service-b", Transport: "sse", URL: "https://example.com/b"},
		}, nil
	}

	scopedTarget := &fakeScopedStatusTarget{
		name:      "Claude Code",
		slug:      "claude",
		installed: true,
		servicesByScope: map[targetpkg.ConfigScope][]string{
			targetpkg.ConfigScopeEffective: {"service-a", "service-b"},
			targetpkg.ConfigScopeProject:   {"service-b"},
		},
	}

	listInstalledTargets = func() []targetpkg.Target {
		return []targetpkg.Target{scopedTarget}
	}

	output, err := executeRootCommand(t, "status", "--scope", "project")
	if err != nil {
		t.Fatalf("expected status command with project scope to succeed: %v", err)
	}

	if scopedTarget.lastScope != targetpkg.ConfigScopeProject {
		t.Fatalf("expected project scope to be used, got %q", scopedTarget.lastScope)
	}

	assertLineMatches(t, output, `(?m)^\s*service-a\s+no\s*$`)
	assertLineMatches(t, output, `(?m)^\s*service-b\s+yes\s*$`)
}

func TestStatusCommandRejectsInvalidScope(t *testing.T) {
	restore := overrideStatusDependencies(t)
	defer restore()

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{}, nil
	}

	_, err := executeRootCommand(t, "status", "--scope", "invalid")
	if err == nil {
		t.Fatal("expected status command to fail for invalid scope")
	}

	if !strings.Contains(err.Error(), "invalid scope") {
		t.Fatalf("expected invalid scope error, got %v", err)
	}
}

func overrideStatusDependencies(t *testing.T) func() {
	t.Helper()

	originalLoadServices := loadServices
	originalListInstalledTargets := listInstalledTargets

	return func() {
		loadServices = originalLoadServices
		listInstalledTargets = originalListInstalledTargets
	}
}

func assertLineMatches(t *testing.T, output string, pattern string) {
	t.Helper()

	matched, err := regexp.MatchString(pattern, output)
	if err != nil {
		t.Fatalf("invalid regex pattern %q: %v", pattern, err)
	}

	if !matched {
		t.Fatalf("expected output to match %q, got %q", pattern, output)
	}
}
