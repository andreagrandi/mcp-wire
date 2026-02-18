package cli

import (
	"bytes"
	"testing"

	"github.com/andreagrandi/mcp-wire/internal/service"
)

type fakeListTarget struct {
	name      string
	slug      string
	installed bool
}

func (t fakeListTarget) Name() string {
	return t.name
}

func (t fakeListTarget) Slug() string {
	return t.slug
}

func (t fakeListTarget) IsInstalled() bool {
	return t.installed
}

func (t fakeListTarget) Install(_ service.Service, _ map[string]string) error {
	return nil
}

func (t fakeListTarget) Uninstall(_ string) error {
	return nil
}

func (t fakeListTarget) List() ([]string, error) {
	return nil, nil
}

func executeRootCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer

	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs(args)
	t.Cleanup(func() {
		rootCmd.SetArgs([]string{})
	})

	err := rootCmd.Execute()
	output := stdout.String() + stderr.String()

	return output, err
}
