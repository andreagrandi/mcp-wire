package cli

import (
	"github.com/andreagrandi/mcp-wire/internal/app"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mcp-wire",
	Short: "Install and configure MCP servers across AI coding tools",
	Long: `mcp-wire is a CLI tool that lets you install and configure MCP (Model Context Protocol)
servers across multiple AI coding CLI tools (Claude Code, Codex, Gemini CLI, etc.)
from a single interface.

Services are defined as YAML files — no code needed to add one.
Targets are the AI tools where services get installed.`,
	Version: app.Version,
}

func Execute() error {
	return rootCmd.Execute()
}
