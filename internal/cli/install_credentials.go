package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/andreagrandi/mcp-wire/internal/credential"
	"github.com/andreagrandi/mcp-wire/internal/service"
	"golang.org/x/term"
)

type interactiveCredentialOptions struct {
	noPrompt     bool
	input        io.Reader
	output       io.Writer
	openURL      func(string) error
	secretReader func(fd int) ([]byte, error)
	fileSource   credential.Source
}

func resolveServiceCredentials(
	svc service.Service,
	resolver *credential.Resolver,
	opts interactiveCredentialOptions,
) (map[string]string, error) {
	opts = normalizeInteractiveCredentialOptions(opts)
	reader := bufio.NewReader(opts.input)
	resolvedEnv := map[string]string{}
	headerPrinted := false

	for _, envVar := range svc.Env {
		envName := strings.TrimSpace(envVar.Name)
		if envName == "" {
			continue
		}

		value, _, found := resolver.Resolve(envName)
		if found {
			resolvedEnv[envName] = value
			continue
		}

		if !envVar.Required {
			continue
		}

		if opts.noPrompt {
			return nil, fmt.Errorf("required credential %q not found and prompting is disabled", envName)
		}

		if !headerPrinted {
			fmt.Fprintf(opts.output, "\nConfiguring: %s\n\n", serviceDisplayName(svc))
			headerPrinted = true
		}

		credentialValue, err := promptForCredentialValue(envVar, reader, opts)
		if err != nil {
			return nil, err
		}

		resolvedEnv[envName] = credentialValue
	}

	return resolvedEnv, nil
}

func normalizeInteractiveCredentialOptions(opts interactiveCredentialOptions) interactiveCredentialOptions {
	if opts.input == nil {
		opts.input = os.Stdin
	}

	if opts.output == nil {
		opts.output = os.Stdout
	}

	if opts.openURL == nil {
		opts.openURL = openSetupURL
	}

	if opts.secretReader == nil {
		opts.secretReader = term.ReadPassword
	}

	return opts
}

func promptForCredentialValue(
	envVar service.EnvVar,
	reader *bufio.Reader,
	opts interactiveCredentialOptions,
) (string, error) {
	envName := strings.TrimSpace(envVar.Name)
	description := strings.TrimSpace(envVar.Description)
	setupURL := strings.TrimSpace(envVar.SetupURL)
	setupHint := strings.TrimSpace(envVar.SetupHint)

	if description == "" {
		fmt.Fprintf(opts.output, "  %s is required.\n", envName)
	} else {
		fmt.Fprintf(opts.output, "  %s is required (%s).\n", envName, description)
	}

	if setupURL != "" {
		fmt.Fprintf(opts.output, "  -> Create one here: %s\n", setupURL)
	}

	if setupHint != "" {
		fmt.Fprintf(opts.output, "     Tip: %s\n", setupHint)
	}

	if setupURL != "" {
		shouldOpen, err := askYesNo(reader, opts.output, "\n  Open URL in browser? [Y/n]: ", true)
		if err != nil {
			return "", fmt.Errorf("read browser confirmation: %w", err)
		}

		if shouldOpen {
			if err := opts.openURL(setupURL); err != nil {
				return "", fmt.Errorf("open setup URL: %w", err)
			}
		}

		fmt.Fprintln(opts.output)
	}

	for {
		value, err := promptSecretValue(reader, opts, "  Paste your token: ")
		if err != nil {
			return "", fmt.Errorf("read credential value for %q: %w", envName, err)
		}

		if value == "" {
			fmt.Fprintln(opts.output, "  Value cannot be empty.")
			continue
		}

		if opts.fileSource != nil {
			shouldStore, err := askYesNo(reader, opts.output, "\n  Save to mcp-wire credential store? [Y/n]: ", true)
			if err != nil {
				return "", fmt.Errorf("read storage confirmation: %w", err)
			}

			if shouldStore {
				if err := opts.fileSource.Store(envName, value); err != nil {
					return "", fmt.Errorf("store credential %q: %w", envName, err)
				}

				fmt.Fprintln(opts.output, "  Saved.")
			}
		}

		fmt.Fprintln(opts.output)
		return value, nil
	}
}

func promptSecretValue(
	reader *bufio.Reader,
	opts interactiveCredentialOptions,
	prompt string,
) (string, error) {
	if inputFile, ok := opts.input.(*os.File); ok && term.IsTerminal(int(inputFile.Fd())) {
		fmt.Fprint(opts.output, prompt)

		value, err := opts.secretReader(int(inputFile.Fd()))
		fmt.Fprintln(opts.output)
		if err != nil {
			return "", err
		}

		return strings.TrimSpace(string(value)), nil
	}

	fmt.Fprint(opts.output, prompt)

	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	trimmedLine := strings.TrimSpace(line)
	if errors.Is(err, io.EOF) && trimmedLine == "" {
		return "", io.EOF
	}

	return trimmedLine, nil
}

func askYesNo(reader *bufio.Reader, output io.Writer, prompt string, defaultYes bool) (bool, error) {
	for {
		fmt.Fprint(output, prompt)

		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return false, err
		}

		answer := strings.ToLower(strings.TrimSpace(line))
		if answer == "" {
			return defaultYes, nil
		}

		switch answer {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			if errors.Is(err, io.EOF) {
				return false, fmt.Errorf("invalid answer %q", answer)
			}

			fmt.Fprintln(output, "  Please answer y or n.")
		}
	}
}

func serviceDisplayName(svc service.Service) string {
	if description := strings.TrimSpace(svc.Description); description != "" {
		return description
	}

	if name := strings.TrimSpace(svc.Name); name != "" {
		return name
	}

	return "service"
}

func openSetupURL(url string) error {
	cmd, err := browserOpenCommand(runtime.GOOS, url)
	if err != nil {
		return err
	}

	return cmd.Run()
}

func browserOpenCommand(goos string, url string) (*exec.Cmd, error) {
	switch goos {
	case "darwin":
		return exec.Command("open", url), nil
	case "linux":
		return exec.Command("xdg-open", url), nil
	case "windows":
		return exec.Command("cmd", "/c", "start", "", url), nil
	default:
		return nil, fmt.Errorf("unsupported operating system %q", goos)
	}
}
