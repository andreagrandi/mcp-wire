package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRootCommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		contains string
	}{
		{
			name:     "version flag",
			args:     []string{"--version"},
			contains: "version",
		},
		{
			name:     "help flag",
			args:     []string{"--help"},
			contains: "mcp-wire",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			rootCmd.SetOut(&stdout)
			rootCmd.SetErr(&stderr)
			rootCmd.SetArgs(tt.args)
			rootCmd.ParseFlags([]string{})

			err := rootCmd.Execute()
			assert.NoError(t, err)

			output := stdout.String() + stderr.String()
			assert.Contains(t, output, tt.contains)

			rootCmd.SetArgs([]string{})
		})
	}
}

func TestRootCommandGuidedMenuExit(t *testing.T) {
	var stdout bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetIn(strings.NewReader("3\n"))

	err := runGuidedMainMenu(cmd)
	assert.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "Main Menu")
	assert.Contains(t, output, "Goodbye")
}
