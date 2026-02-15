package cli

import (
	"errors"
	"strings"
	"testing"
)

func TestCacheClearCommandReportsCleared(t *testing.T) {
	original := clearRegistryCache
	t.Cleanup(func() { clearRegistryCache = original })

	clearRegistryCache = func() (string, bool, error) {
		return "/tmp/registry-cache.json", true, nil
	}

	output, err := executeRootCommand(t, "cache", "clear")
	if err != nil {
		t.Fatalf("expected cache clear to succeed: %v", err)
	}

	if !strings.Contains(output, "Registry cache cleared: /tmp/registry-cache.json") {
		t.Fatalf("expected cleared message, got %q", output)
	}
}

func TestCacheClearCommandReportsAlreadyEmpty(t *testing.T) {
	original := clearRegistryCache
	t.Cleanup(func() { clearRegistryCache = original })

	clearRegistryCache = func() (string, bool, error) {
		return "/tmp/registry-cache.json", false, nil
	}

	output, err := executeRootCommand(t, "cache", "clear")
	if err != nil {
		t.Fatalf("expected cache clear to succeed: %v", err)
	}

	if !strings.Contains(output, "Registry cache already empty: /tmp/registry-cache.json") {
		t.Fatalf("expected already-empty message, got %q", output)
	}
}

func TestCacheClearCommandReturnsError(t *testing.T) {
	original := clearRegistryCache
	t.Cleanup(func() { clearRegistryCache = original })

	clearRegistryCache = func() (string, bool, error) {
		return "", false, errors.New("boom")
	}

	_, err := executeRootCommand(t, "cache", "clear")
	if err == nil {
		t.Fatal("expected cache clear to fail")
	}

	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}
