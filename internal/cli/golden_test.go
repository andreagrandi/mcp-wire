package cli

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/andreagrandi/mcp-wire/internal/catalog"
	"github.com/andreagrandi/mcp-wire/internal/credential"
	"github.com/andreagrandi/mcp-wire/internal/registry"
	"github.com/andreagrandi/mcp-wire/internal/service"
	targetpkg "github.com/andreagrandi/mcp-wire/internal/target"
)

// errGoldenDiskFull is a fixed error used to render the install-failure golden
// deterministically.
var errGoldenDiskFull = errors.New("disk full")

// updateGolden regenerates the golden files instead of comparing against them.
// Run: go test ./internal/cli/ -run Golden -update
var updateGolden = flag.Bool("update", false, "update golden files in testdata/golden")

// assertGolden compares actual against the golden file named name (without the
// .golden suffix), or rewrites it when -update is passed. The plain wizard uses
// no colour, so its output is already deterministic plain text.
func assertGolden(t *testing.T, name, actual string) {
	t.Helper()

	path := filepath.Join("testdata", "golden", name+".golden")

	if *updateGolden {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(actual), 0o644))
		return
	}

	want, err := os.ReadFile(path)
	require.NoErrorf(t, err, "missing golden file %s; regenerate with: go test ./internal/cli/ -run Golden -update", path)
	assert.Equalf(t, string(want), actual,
		"plain wizard output differs from %s; if the change is intentional, regenerate with: go test ./internal/cli/ -run Golden -update", path)
}

func goldenReviewTargets() []targetpkg.Target {
	return []targetpkg.Target{
		&fakeInstallTarget{name: "Claude Code", slug: "claude", installed: true},
		&fakeInstallTarget{name: "Codex", slug: "codex", installed: true},
	}
}

func goldenReviewService() service.Service {
	return service.Service{Name: "sentry", Description: "Error tracking"}
}

func TestPlainReviewGolden(t *testing.T) {
	t.Run("install", func(t *testing.T) {
		var buf bytes.Buffer
		reader := bufio.NewReader(strings.NewReader("y\n"))
		_, err := confirmInstallSelection(&buf, reader, goldenReviewService(), goldenReviewTargets(), false, targetpkg.ConfigScopeUser, nil)
		require.NoError(t, err)
		assertGolden(t, "plain_install_review", buf.String())
	})

	t.Run("install_project_scope", func(t *testing.T) {
		targets := []targetpkg.Target{
			&fakeScopedInstallTarget{fakeInstallTarget: &fakeInstallTarget{name: "Claude Code", slug: "claude", installed: true}},
		}
		var buf bytes.Buffer
		reader := bufio.NewReader(strings.NewReader("y\n"))
		_, err := confirmInstallSelection(&buf, reader, goldenReviewService(), targets, false, targetpkg.ConfigScopeProject, nil)
		require.NoError(t, err)
		assertGolden(t, "plain_install_review_project_scope", buf.String())
	})

	t.Run("uninstall", func(t *testing.T) {
		var buf bytes.Buffer
		reader := bufio.NewReader(strings.NewReader("y\n"))
		_, err := confirmUninstallSelection(&buf, reader, goldenReviewService(), goldenReviewTargets(), targetpkg.ConfigScopeUser)
		require.NoError(t, err)
		assertGolden(t, "plain_uninstall_review", buf.String())
	})
}

func TestPlainTrustSummaryGolden(t *testing.T) {
	t.Run("remote", func(t *testing.T) {
		entry := catalog.Entry{
			Source: catalog.SourceRegistry,
			Name:   "community-svc",
			Registry: &registry.ServerResponse{
				Server: registry.ServerJSON{
					Name:        "community-svc",
					Description: "A community service",
					Remotes: []registry.Transport{
						{
							Type: "sse",
							URL:  "https://example.com/sse",
							Headers: []registry.KeyValueInput{
								{Name: "API_KEY", Description: "API key", IsRequired: true, IsSecret: true},
							},
						},
					},
					Repository: &registry.Repository{URL: "https://github.com/example/svc"},
				},
			},
		}
		var buf bytes.Buffer
		printRegistryTrustSummary(&buf, entry)
		assertGolden(t, "plain_trust_remote", buf.String())
	})

	t.Run("package", func(t *testing.T) {
		entry := catalog.Entry{
			Source: catalog.SourceRegistry,
			Name:   "pkg-svc",
			Registry: &registry.ServerResponse{
				Server: registry.ServerJSON{
					Name:        "pkg-svc",
					Description: "A package service",
					Packages: []registry.Package{
						{
							RegistryType: "npm",
							Identifier:   "@example/mcp-server",
							Version:      "1.2.3",
							RuntimeHint:  "requires Node.js 18+",
						},
					},
				},
			},
		}
		var buf bytes.Buffer
		printRegistryTrustSummary(&buf, entry)
		assertGolden(t, "plain_trust_package", buf.String())
	})
}

func TestPlainCredentialPromptGolden(t *testing.T) {
	resolver := credential.NewResolver(&fakeCredentialSource{values: map[string]string{}})
	svc := service.Service{
		Name:        "sentry",
		Description: "Sentry error tracking",
		Env: []service.EnvVar{
			{
				Name:        "SENTRY_TOKEN",
				Description: "API token",
				Required:    true,
				SetupURL:    "https://sentry.io/settings/auth-tokens/",
				SetupHint:   "Create a token with read scope",
			},
		},
	}

	var buf bytes.Buffer
	// Decline opening the URL, then supply the token value.
	_, err := resolveServiceCredentials(svc, resolver, interactiveCredentialOptions{
		input:   strings.NewReader("n\nsecret-token\n"),
		output:  &buf,
		openURL: func(_ string) error { return nil },
	})
	require.NoError(t, err)
	assertGolden(t, "plain_credential_prompt", buf.String())
}

func TestPlainApplySummaryGolden(t *testing.T) {
	restore := stubGoldenInstallDependencies()
	defer restore()

	svc := service.Service{
		Name:        "sentry",
		Description: "Error tracking",
		Transport:   "stdio",
		Command:     "npx",
		Args:        []string{"-y", "@sentry/mcp-server"},
	}

	t.Run("success", func(t *testing.T) {
		targets := goldenReviewTargets()
		cmd, buf := goldenInstallCommand()
		err := executeInstall(cmd, svc, targets, true, targetpkg.ConfigScopeUser)
		require.NoError(t, err)
		assertGolden(t, "plain_apply_install", buf.String())
	})

	t.Run("partial_failure", func(t *testing.T) {
		targets := []targetpkg.Target{
			&fakeInstallTarget{name: "Claude Code", slug: "claude", installed: true},
			&fakeInstallTarget{name: "Codex", slug: "codex", installed: true, installErr: errGoldenDiskFull},
		}
		cmd, buf := goldenInstallCommand()
		err := executeInstall(cmd, svc, targets, true, targetpkg.ConfigScopeUser)
		require.Error(t, err)
		assertGolden(t, "plain_apply_install_failure", buf.String())
	})
}

// stubGoldenInstallDependencies replaces the credential and auth hooks with
// deterministic no-ops so executeInstall touches no real environment, files, or
// flags. It returns a restore function.
func stubGoldenInstallDependencies() func() {
	originalEnvSource := newCredentialEnvSource
	originalFileSource := newCredentialFileSource
	originalResolver := newCredentialResolver
	originalAutoAuth := shouldAutoAuthenticate

	newCredentialEnvSource = func() credential.Source { return nil }
	newCredentialFileSource = func(string) credential.Source { return nil }
	newCredentialResolver = func(_ ...credential.Source) *credential.Resolver { return credential.NewResolver() }
	shouldAutoAuthenticate = func(_ *cobra.Command) bool { return false }

	return func() {
		newCredentialEnvSource = originalEnvSource
		newCredentialFileSource = originalFileSource
		newCredentialResolver = originalResolver
		shouldAutoAuthenticate = originalAutoAuth
	}
}

func goldenInstallCommand() (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetIn(strings.NewReader(""))
	return cmd, &buf
}
