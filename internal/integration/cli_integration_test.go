//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOpenCodeLifecycleInstallStatusUninstall(t *testing.T) {
	sandbox := newCLISandbox(t)

	writeServiceDefinition(t, filepath.Join(sandbox.servicesDir, "remote-docs.yaml"), `name: remote-docs
description: "Remote docs service"
transport: sse
url: "https://docs.example.com/mcp"
env: []
`)

	listServicesOutput, err := sandbox.runCLI("list", "services")
	if err != nil {
		t.Fatalf("list services failed: %v\n%s", err, listServicesOutput)
	}

	if !strings.Contains(listServicesOutput, "remote-docs") {
		t.Fatalf("expected custom service in list output, got:\n%s", listServicesOutput)
	}

	listTargetsOutput, err := sandbox.runCLI("list", "targets")
	if err != nil {
		t.Fatalf("list targets failed: %v\n%s", err, listTargetsOutput)
	}

	if !strings.Contains(listTargetsOutput, "opencode") || !strings.Contains(listTargetsOutput, "installed") {
		t.Fatalf("expected opencode installed in list output, got:\n%s", listTargetsOutput)
	}

	installOutput, err := sandbox.runCLI("install", "remote-docs", "--target", "opencode", "--no-prompt")
	if err != nil {
		t.Fatalf("install failed: %v\n%s", err, installOutput)
	}

	if !strings.Contains(installOutput, "OpenCode: configured") {
		t.Fatalf("expected successful OpenCode install output, got:\n%s", installOutput)
	}

	config := sandbox.readOpenCodeConfig()
	mcp := mustObject(t, config["mcp"], "mcp")
	remoteDocs := mustObject(t, mcp["remote-docs"], "mcp.remote-docs")

	if remoteDocs["type"] != "remote" {
		t.Fatalf("expected remote type, got %#v", remoteDocs["type"])
	}

	if remoteDocs["url"] != "https://docs.example.com/mcp" {
		t.Fatalf("expected URL to match, got %#v", remoteDocs["url"])
	}

	statusOutput, err := sandbox.runCLI("status")
	if err != nil {
		t.Fatalf("status failed: %v\n%s", err, statusOutput)
	}

	if !strings.Contains(statusOutput, "remote-docs") || !strings.Contains(statusOutput, "yes") {
		t.Fatalf("expected status matrix to include installed service, got:\n%s", statusOutput)
	}

	uninstallOutput, err := sandbox.runCLI("uninstall", "remote-docs", "--target", "opencode")
	if err != nil {
		t.Fatalf("uninstall failed: %v\n%s", err, uninstallOutput)
	}

	if !strings.Contains(uninstallOutput, "OpenCode: removed") {
		t.Fatalf("expected successful OpenCode uninstall output, got:\n%s", uninstallOutput)
	}

	configAfterUninstall := sandbox.readOpenCodeConfig()
	mcpAfterUninstall := mustObject(t, configAfterUninstall["mcp"], "mcp")
	if _, exists := mcpAfterUninstall["remote-docs"]; exists {
		t.Fatalf("expected remote-docs to be removed from config, got %#v", mcpAfterUninstall)
	}
}

func TestOpenCodeInstallUsesCredentialFromEnvironment(t *testing.T) {
	sandbox := newCLISandbox(t)

	writeServiceDefinition(t, filepath.Join(sandbox.servicesDir, "secure-remote.yaml"), `name: secure-remote
description: "Secure remote service"
transport: sse
url: "https://secure.example.com/mcp"
env:
  - name: SECURE_REMOTE_TOKEN
    description: "API token"
    required: true
`)

	installOutput, err := sandbox.runCLIWithEnv(map[string]string{
		"SECURE_REMOTE_TOKEN": "integration-env-token",
	}, "install", "secure-remote", "--target", "opencode", "--no-prompt")
	if err != nil {
		t.Fatalf("install with env credential failed: %v\n%s", err, installOutput)
	}

	config := sandbox.readOpenCodeConfig()
	mcp := mustObject(t, config["mcp"], "mcp")
	secureRemote := mustObject(t, mcp["secure-remote"], "mcp.secure-remote")
	headers := mustObject(t, secureRemote["headers"], "mcp.secure-remote.headers")

	if headers["SECURE_REMOTE_TOKEN"] != "integration-env-token" {
		t.Fatalf("expected header value from environment, got %#v", headers["SECURE_REMOTE_TOKEN"])
	}
}

func TestOpenCodeInstallUsesCredentialFromFileStore(t *testing.T) {
	sandbox := newCLISandbox(t)

	writeServiceDefinition(t, filepath.Join(sandbox.servicesDir, "file-backed.yaml"), `name: file-backed
description: "File-backed credential service"
transport: sse
url: "https://file.example.com/mcp"
env:
  - name: FILE_BACKED_TOKEN
    description: "API token"
    required: true
`)

	credentialsDir := filepath.Join(sandbox.homeDir, ".config", "mcp-wire")
	err := os.MkdirAll(credentialsDir, 0o700)
	if err != nil {
		t.Fatalf("failed to create credentials directory: %v", err)
	}

	credentialsPath := filepath.Join(credentialsDir, "credentials")
	err = os.WriteFile(credentialsPath, []byte("FILE_BACKED_TOKEN=integration-file-token\n"), 0o600)
	if err != nil {
		t.Fatalf("failed to write credentials file: %v", err)
	}

	installOutput, err := sandbox.runCLI("install", "file-backed", "--target", "opencode", "--no-prompt")
	if err != nil {
		t.Fatalf("install with file credential failed: %v\n%s", err, installOutput)
	}

	config := sandbox.readOpenCodeConfig()
	mcp := mustObject(t, config["mcp"], "mcp")
	fileBacked := mustObject(t, mcp["file-backed"], "mcp.file-backed")
	headers := mustObject(t, fileBacked["headers"], "mcp.file-backed.headers")

	if headers["FILE_BACKED_TOKEN"] != "integration-file-token" {
		t.Fatalf("expected header value from file store, got %#v", headers["FILE_BACKED_TOKEN"])
	}
}

type cliSandbox struct {
	binaryPath  string
	homeDir     string
	servicesDir string
	path        string
	repoRoot    string
}

func newCLISandbox(t *testing.T) cliSandbox {
	t.Helper()

	repoRoot := repoRootPath(t)
	homeDir := t.TempDir()

	servicesDir := filepath.Join(homeDir, ".config", "mcp-wire", "services")
	err := os.MkdirAll(servicesDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create user services directory: %v", err)
	}

	binDir := filepath.Join(t.TempDir(), "bin")
	err = os.MkdirAll(binDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create fake bin directory: %v", err)
	}

	createExecutable(t, filepath.Join(binDir, "opencode"), "#!/bin/sh\nexit 0\n")

	binaryPath := filepath.Join(t.TempDir(), "mcp-wire")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/mcp-wire")
	buildCmd.Dir = repoRoot
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build mcp-wire binary: %v\n%s", err, string(buildOutput))
	}

	return cliSandbox{
		binaryPath:  binaryPath,
		homeDir:     homeDir,
		servicesDir: servicesDir,
		path:        binDir,
		repoRoot:    repoRoot,
	}
}

func (s cliSandbox) runCLI(args ...string) (string, error) {
	return s.runCLIWithEnv(map[string]string{}, args...)
}

func (s cliSandbox) runCLIWithEnv(extraEnv map[string]string, args ...string) (string, error) {
	cmd := exec.Command(s.binaryPath, args...)
	cmd.Dir = s.repoRoot
	cmd.Env = []string{
		"HOME=" + s.homeDir,
		"PATH=" + s.path,
	}

	for key, value := range extraEnv {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (s cliSandbox) readOpenCodeConfig() map[string]any {
	configPath := filepath.Join(s.homeDir, ".config", "opencode", "opencode.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return map[string]any{}
	}

	config := map[string]any{}
	if err := json.Unmarshal(data, &config); err != nil {
		return map[string]any{}
	}

	return config
}

func repoRootPath(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve current file path")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}

func createExecutable(t *testing.T, path string, content string) {
	t.Helper()

	err := os.WriteFile(path, []byte(content), 0o755)
	if err != nil {
		t.Fatalf("failed to create executable %q: %v", path, err)
	}
}

func writeServiceDefinition(t *testing.T, path string, content string) {
	t.Helper()

	err := os.WriteFile(path, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("failed to write service definition %q: %v", path, err)
	}
}

func mustObject(t *testing.T, value any, path string) map[string]any {
	t.Helper()

	objectValue, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected %s to be an object, got %#v", path, value)
	}

	return objectValue
}
