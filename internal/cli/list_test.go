package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/andreagrandi/mcp-wire/internal/service"
)

func TestListServicesCommandPrintsSortedServices(t *testing.T) {
	originalLoadServices := loadServices
	t.Cleanup(func() {
		loadServices = originalLoadServices
	})

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{
			"zeta": {
				Name:        "zeta",
				Description: "Last service",
			},
			"alpha": {
				Name:        "alpha",
				Description: "First service",
			},
		}, nil
	}

	output, err := executeRootCommand(t, "list", "services")
	if err != nil {
		t.Fatalf("expected list services command to succeed: %v", err)
	}

	if !strings.Contains(output, "Available services:") {
		t.Fatalf("expected heading in output, got %q", output)
	}

	alphaIndex := strings.Index(output, "alpha")
	zetaIndex := strings.Index(output, "zeta")
	if alphaIndex == -1 || zetaIndex == -1 {
		t.Fatalf("expected both services in output, got %q", output)
	}

	if alphaIndex > zetaIndex {
		t.Fatalf("expected services sorted alphabetically, got %q", output)
	}
}

func TestListServicesCommandPrintsEmptyState(t *testing.T) {
	originalLoadServices := loadServices
	t.Cleanup(func() {
		loadServices = originalLoadServices
	})

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return map[string]service.Service{}, nil
	}

	output, err := executeRootCommand(t, "list", "services")
	if err != nil {
		t.Fatalf("expected list services command to succeed: %v", err)
	}

	if !strings.Contains(output, "(none)") {
		t.Fatalf("expected empty state marker, got %q", output)
	}
}

func TestListServicesCommandReturnsLoaderError(t *testing.T) {
	originalLoadServices := loadServices
	t.Cleanup(func() {
		loadServices = originalLoadServices
	})

	loadServices = func(_ ...string) (map[string]service.Service, error) {
		return nil, errors.New("loader failed")
	}

	_, err := executeRootCommand(t, "list", "services")
	if err == nil {
		t.Fatal("expected list services command to fail")
	}

	if !strings.Contains(err.Error(), "load services") {
		t.Fatalf("expected wrapped loader error, got %v", err)
	}
}

func TestPrintServicesListPrintsServiceWithoutDescription(t *testing.T) {
	services := map[string]service.Service{
		"demo": {
			Name: "demo",
		},
	}

	var output bytes.Buffer
	printServicesList(&output, services)

	if !strings.Contains(output.String(), "  demo\n") {
		t.Fatalf("expected service name line without description, got %q", output.String())
	}
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
