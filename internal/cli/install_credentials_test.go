package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/andreagrandi/mcp-wire/internal/credential"
	"github.com/andreagrandi/mcp-wire/internal/service"
)

type fakeCredentialSource struct {
	name     string
	values   map[string]string
	stored   map[string]string
	storeErr error
}

func (s *fakeCredentialSource) Name() string {
	if s.name == "" {
		return "fake-source"
	}

	return s.name
}

func (s *fakeCredentialSource) Get(envName string) (string, bool) {
	value, found := s.values[envName]
	return value, found
}

func (s *fakeCredentialSource) Store(envName string, value string) error {
	if s.storeErr != nil {
		return s.storeErr
	}

	if s.stored == nil {
		s.stored = map[string]string{}
	}

	s.stored[envName] = value
	return nil
}

func TestResolveServiceCredentialsUsesResolverValueWithoutPrompt(t *testing.T) {
	resolverSource := &fakeCredentialSource{
		name: "environment",
		values: map[string]string{
			"DEMO_TOKEN": "resolved-value",
		},
	}

	resolver := credential.NewResolver(resolverSource)
	svc := service.Service{
		Name: "demo-service",
		Env: []service.EnvVar{
			{Name: "DEMO_TOKEN", Required: true},
		},
	}

	var output bytes.Buffer
	resolved, err := resolveServiceCredentials(svc, resolver, interactiveCredentialOptions{
		input:  strings.NewReader(""),
		output: &output,
	})
	if err != nil {
		t.Fatalf("expected resolver flow to succeed: %v", err)
	}

	if resolved["DEMO_TOKEN"] != "resolved-value" {
		t.Fatalf("expected resolver value, got %q", resolved["DEMO_TOKEN"])
	}

	if output.String() != "" {
		t.Fatalf("expected no prompt output, got %q", output.String())
	}
}

func TestResolveServiceCredentialsPromptsAndStoresValue(t *testing.T) {
	resolver := credential.NewResolver(&fakeCredentialSource{values: map[string]string{}})
	store := &fakeCredentialSource{name: "file"}

	svc := service.Service{
		Name:        "demo-service",
		Description: "Demo service",
		Env: []service.EnvVar{
			{Name: "DEMO_TOKEN", Description: "Token for access", Required: true},
		},
	}

	input := strings.NewReader("token-value\ny\n")
	var output bytes.Buffer

	resolved, err := resolveServiceCredentials(svc, resolver, interactiveCredentialOptions{
		input:      input,
		output:     &output,
		fileSource: store,
	})
	if err != nil {
		t.Fatalf("expected prompt flow to succeed: %v", err)
	}

	if resolved["DEMO_TOKEN"] != "token-value" {
		t.Fatalf("expected token-value, got %q", resolved["DEMO_TOKEN"])
	}

	if store.stored["DEMO_TOKEN"] != "token-value" {
		t.Fatalf("expected value to be stored, got %q", store.stored["DEMO_TOKEN"])
	}

	console := output.String()
	if !strings.Contains(console, "DEMO_TOKEN required") {
		t.Fatalf("expected prompt output to mention required token, got %q", console)
	}

	if !strings.Contains(console, "Save to credential store") {
		t.Fatalf("expected prompt output to include store confirmation, got %q", console)
	}
}

func TestResolveServiceCredentialsReturnsErrorWhenNoPromptEnabled(t *testing.T) {
	resolver := credential.NewResolver(&fakeCredentialSource{values: map[string]string{}})
	svc := service.Service{
		Name: "demo-service",
		Env: []service.EnvVar{
			{Name: "DEMO_TOKEN", Required: true},
		},
	}

	_, err := resolveServiceCredentials(svc, resolver, interactiveCredentialOptions{noPrompt: true})
	if err == nil {
		t.Fatal("expected no-prompt mode to fail when required credential is missing")
	}

	if !strings.Contains(err.Error(), "DEMO_TOKEN") {
		t.Fatalf("expected error to include env name, got %v", err)
	}
}

func TestResolveServiceCredentialsOpensSetupURLWhenChosen(t *testing.T) {
	resolver := credential.NewResolver(&fakeCredentialSource{values: map[string]string{}})

	openCount := 0
	openURL := func(url string) error {
		if url != "https://example.com/token" {
			t.Fatalf("expected setup URL to be passed through, got %q", url)
		}

		openCount++
		return nil
	}

	svc := service.Service{
		Name: "demo-service",
		Env: []service.EnvVar{
			{
				Name:      "DEMO_TOKEN",
				Required:  true,
				SetupURL:  "https://example.com/token",
				SetupHint: "Create a read-only token",
			},
		},
	}

	var output bytes.Buffer
	resolved, err := resolveServiceCredentials(svc, resolver, interactiveCredentialOptions{
		input:      strings.NewReader("y\ntoken-value\nn\n"),
		output:     &output,
		openURL:    openURL,
		fileSource: &fakeCredentialSource{name: "file"},
	})
	if err != nil {
		t.Fatalf("expected setup URL flow to succeed: %v", err)
	}

	if openCount != 1 {
		t.Fatalf("expected browser opener to run once, got %d", openCount)
	}

	if resolved["DEMO_TOKEN"] != "token-value" {
		t.Fatalf("expected token-value, got %q", resolved["DEMO_TOKEN"])
	}
}

func TestResolveServiceCredentialsSkipsOptionalMissingValue(t *testing.T) {
	resolver := credential.NewResolver(&fakeCredentialSource{values: map[string]string{}})
	svc := service.Service{
		Name: "demo-service",
		Env: []service.EnvVar{
			{Name: "OPTIONAL_VALUE", Required: false},
		},
	}

	resolved, err := resolveServiceCredentials(svc, resolver, interactiveCredentialOptions{noPrompt: true})
	if err != nil {
		t.Fatalf("expected optional missing value to be skipped: %v", err)
	}

	if len(resolved) != 0 {
		t.Fatalf("expected no resolved env values, got %#v", resolved)
	}
}

func TestResolveServiceCredentialsReturnsStoreError(t *testing.T) {
	resolver := credential.NewResolver(&fakeCredentialSource{values: map[string]string{}})
	storeError := errors.New("store failed")
	store := &fakeCredentialSource{name: "file", storeErr: storeError}

	svc := service.Service{
		Name: "demo-service",
		Env: []service.EnvVar{
			{Name: "DEMO_TOKEN", Required: true},
		},
	}

	_, err := resolveServiceCredentials(svc, resolver, interactiveCredentialOptions{
		input:      strings.NewReader("token-value\ny\n"),
		output:     &bytes.Buffer{},
		fileSource: store,
	})
	if err == nil {
		t.Fatal("expected store error to be returned")
	}

	if !strings.Contains(err.Error(), "store credential") {
		t.Fatalf("expected store error message, got %v", err)
	}
}

func TestResolveServiceCredentialsReturnsOpenURLError(t *testing.T) {
	resolver := credential.NewResolver(&fakeCredentialSource{values: map[string]string{}})
	openErr := errors.New("open failed")

	svc := service.Service{
		Name: "demo-service",
		Env: []service.EnvVar{
			{Name: "DEMO_TOKEN", Required: true, SetupURL: "https://example.com/token"},
		},
	}

	_, err := resolveServiceCredentials(svc, resolver, interactiveCredentialOptions{
		input:      strings.NewReader("y\n"),
		output:     &bytes.Buffer{},
		fileSource: &fakeCredentialSource{name: "file"},
		openURL: func(string) error {
			return openErr
		},
	})
	if err == nil {
		t.Fatal("expected open URL error to be returned")
	}

	if !strings.Contains(err.Error(), "open setup URL") {
		t.Fatalf("expected open URL error message, got %v", err)
	}
}

func TestBrowserOpenCommandDarwin(t *testing.T) {
	cmd, err := browserOpenCommand("darwin", "https://example.com")
	if err != nil {
		t.Fatalf("expected darwin command, got error: %v", err)
	}

	if len(cmd.Args) != 2 || cmd.Args[0] != "open" || cmd.Args[1] != "https://example.com" {
		t.Fatalf("unexpected darwin args: %#v", cmd.Args)
	}
}

func TestBrowserOpenCommandLinux(t *testing.T) {
	cmd, err := browserOpenCommand("linux", "https://example.com")
	if err != nil {
		t.Fatalf("expected linux command, got error: %v", err)
	}

	if len(cmd.Args) != 2 || cmd.Args[0] != "xdg-open" || cmd.Args[1] != "https://example.com" {
		t.Fatalf("unexpected linux args: %#v", cmd.Args)
	}
}

func TestBrowserOpenCommandWindows(t *testing.T) {
	cmd, err := browserOpenCommand("windows", "https://example.com")
	if err != nil {
		t.Fatalf("expected windows command, got error: %v", err)
	}

	if len(cmd.Args) != 5 || cmd.Args[0] != "cmd" || cmd.Args[1] != "/c" || cmd.Args[2] != "start" || cmd.Args[3] != "" || cmd.Args[4] != "https://example.com" {
		t.Fatalf("unexpected windows args: %#v", cmd.Args)
	}
}

func TestBrowserOpenCommandUnsupportedOS(t *testing.T) {
	_, err := browserOpenCommand("unsupported-os", "https://example.com")
	if err == nil {
		t.Fatal("expected unsupported operating system error")
	}
}
