package credential

import (
	"errors"
	"testing"
)

func TestEnvSourceName(t *testing.T) {
	source := NewEnvSource()

	if source.Name() != "environment" {
		t.Fatalf("expected source name environment, got %q", source.Name())
	}
}

func TestEnvSourceGetReturnsValueWhenPresent(t *testing.T) {
	source := NewEnvSource()
	t.Setenv("MCP_WIRE_TEST_ENV_TOKEN", "token-value")

	value, found := source.Get("MCP_WIRE_TEST_ENV_TOKEN")
	if !found {
		t.Fatal("expected variable to be found")
	}

	if value != "token-value" {
		t.Fatalf("expected token-value, got %q", value)
	}
}

func TestEnvSourceGetReturnsFalseWhenMissing(t *testing.T) {
	source := NewEnvSource()

	value, found := source.Get("MCP_WIRE_TEST_ENV_MISSING")
	if found {
		t.Fatal("expected missing variable to not be found")
	}

	if value != "" {
		t.Fatalf("expected empty value for missing variable, got %q", value)
	}
}

func TestEnvSourceGetReturnsFalseForEmptyName(t *testing.T) {
	source := NewEnvSource()

	_, found := source.Get("   ")
	if found {
		t.Fatal("expected empty env name to return not found")
	}
}

func TestEnvSourceStoreReturnsErrNotSupported(t *testing.T) {
	source := NewEnvSource()
	err := source.Store("MCP_WIRE_TEST_ENV_TOKEN", "value")

	if !errors.Is(err, ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported, got %v", err)
	}
}
