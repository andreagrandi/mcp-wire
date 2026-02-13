package credential

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileSourceName(t *testing.T) {
	source := NewFileSource(filepath.Join(t.TempDir(), "credentials"))

	if source.Name() != "file" {
		t.Fatalf("expected source name file, got %q", source.Name())
	}
}

func TestFileSourceUsesDefaultPathWhenEmpty(t *testing.T) {
	source := NewFileSource("")
	if source.path == "" {
		t.Fatal("expected default path to be set")
	}

	if !strings.HasSuffix(source.path, filepath.Join(".config", "mcp-wire", "credentials")) {
		t.Fatalf("unexpected default path: %q", source.path)
	}
}

func TestFileSourceGetReturnsFalseWhenFileMissing(t *testing.T) {
	source := NewFileSource(filepath.Join(t.TempDir(), "credentials"))

	value, found := source.Get("DEMO_TOKEN")
	if found {
		t.Fatal("expected missing credentials file to return not found")
	}

	if value != "" {
		t.Fatalf("expected empty value, got %q", value)
	}
}

func TestFileSourceStoreAndGetRoundTrip(t *testing.T) {
	credentialsPath := filepath.Join(t.TempDir(), "nested", "credentials")
	source := NewFileSource(credentialsPath)

	err := source.Store("DEMO_TOKEN", "token-value")
	if err != nil {
		t.Fatalf("expected store to succeed: %v", err)
	}

	value, found := source.Get("DEMO_TOKEN")
	if !found {
		t.Fatal("expected value to be found")
	}

	if value != "token-value" {
		t.Fatalf("expected token-value, got %q", value)
	}
}

func TestFileSourceStoreUpdatesExistingKey(t *testing.T) {
	credentialsPath := filepath.Join(t.TempDir(), "credentials")
	source := NewFileSource(credentialsPath)

	err := source.Store("DEMO_TOKEN", "first-value")
	if err != nil {
		t.Fatalf("expected first store to succeed: %v", err)
	}

	err = source.Store("DEMO_TOKEN", "updated-value")
	if err != nil {
		t.Fatalf("expected second store to succeed: %v", err)
	}

	value, found := source.Get("DEMO_TOKEN")
	if !found {
		t.Fatal("expected updated key to be found")
	}

	if value != "updated-value" {
		t.Fatalf("expected updated-value, got %q", value)
	}
}

func TestFileSourceStoreKeepsOtherKeys(t *testing.T) {
	credentialsPath := filepath.Join(t.TempDir(), "credentials")
	source := NewFileSource(credentialsPath)

	if err := source.Store("TOKEN_A", "value-a"); err != nil {
		t.Fatalf("expected first store to succeed: %v", err)
	}

	if err := source.Store("TOKEN_B", "value-b"); err != nil {
		t.Fatalf("expected second store to succeed: %v", err)
	}

	if err := source.Store("TOKEN_A", "value-a-updated"); err != nil {
		t.Fatalf("expected third store to succeed: %v", err)
	}

	valueA, foundA := source.Get("TOKEN_A")
	if !foundA || valueA != "value-a-updated" {
		t.Fatalf("expected TOKEN_A value-a-updated, got %q (found=%v)", valueA, foundA)
	}

	valueB, foundB := source.Get("TOKEN_B")
	if !foundB || valueB != "value-b" {
		t.Fatalf("expected TOKEN_B value-b, got %q (found=%v)", valueB, foundB)
	}
}

func TestFileSourceStoreCreatesFileWithStrictPermissions(t *testing.T) {
	credentialsPath := filepath.Join(t.TempDir(), "credentials")
	source := NewFileSource(credentialsPath)

	err := source.Store("DEMO_TOKEN", "token-value")
	if err != nil {
		t.Fatalf("expected store to succeed: %v", err)
	}

	info, err := os.Stat(credentialsPath)
	if err != nil {
		t.Fatalf("expected credentials file to exist: %v", err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected credentials file mode 0600, got %#o", info.Mode().Perm())
	}
}

func TestFileSourceGetParsesAndIgnoresInvalidLines(t *testing.T) {
	credentialsPath := filepath.Join(t.TempDir(), "credentials")
	content := "invalid-line\n# comment\nTOKEN_A=value-a\nTOKEN_B=value=b\n=broken\n"
	err := os.WriteFile(credentialsPath, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("failed to seed credentials file: %v", err)
	}

	source := NewFileSource(credentialsPath)

	valueA, foundA := source.Get("TOKEN_A")
	if !foundA || valueA != "value-a" {
		t.Fatalf("expected TOKEN_A value-a, got %q (found=%v)", valueA, foundA)
	}

	valueB, foundB := source.Get("TOKEN_B")
	if !foundB || valueB != "value=b" {
		t.Fatalf("expected TOKEN_B value=b, got %q (found=%v)", valueB, foundB)
	}

	_, foundBroken := source.Get("invalid-line")
	if foundBroken {
		t.Fatal("expected invalid line to be ignored")
	}
}

func TestFileSourceStoreRejectsEmptyName(t *testing.T) {
	source := NewFileSource(filepath.Join(t.TempDir(), "credentials"))
	err := source.Store("   ", "value")
	if err == nil {
		t.Fatal("expected error for empty env name")
	}
}

func TestFileSourceGetReturnsFalseForEmptyName(t *testing.T) {
	source := NewFileSource(filepath.Join(t.TempDir(), "credentials"))

	_, found := source.Get(" ")
	if found {
		t.Fatal("expected empty env name to return not found")
	}
}

func TestFileSourceStoreReturnsErrorOnNilReceiver(t *testing.T) {
	var source *FileSource
	err := source.Store("DEMO_TOKEN", "value")
	if err == nil {
		t.Fatal("expected nil receiver to return error")
	}
}

func TestFileSourceGetReturnsFalseOnNilReceiver(t *testing.T) {
	var source *FileSource

	_, found := source.Get("DEMO_TOKEN")
	if found {
		t.Fatal("expected nil receiver to return not found")
	}
}

func TestFileSourceStoreCanPersistEmptyValue(t *testing.T) {
	credentialsPath := filepath.Join(t.TempDir(), "credentials")
	source := NewFileSource(credentialsPath)

	err := source.Store("OPTIONAL_TOKEN", "")
	if err != nil {
		t.Fatalf("expected store to succeed with empty value: %v", err)
	}

	value, found := source.Get("OPTIONAL_TOKEN")
	if !found {
		t.Fatal("expected key with empty value to be found")
	}

	if value != "" {
		t.Fatalf("expected empty value, got %q", value)
	}
}

func TestFileSourceGetReturnsFalseWhenReadFails(t *testing.T) {
	credentialsPath := t.TempDir()
	source := NewFileSource(credentialsPath)

	_, found := source.Get("DEMO_TOKEN")
	if found {
		t.Fatal("expected read failure to return not found")
	}
}

func TestFileSourceStoreReturnsErrorWhenReadFails(t *testing.T) {
	credentialsPath := t.TempDir()
	source := NewFileSource(credentialsPath)

	err := source.Store("DEMO_TOKEN", "value")
	if err == nil {
		t.Fatal("expected error when path is a directory")
	}
}

func TestFileSourceStoreReturnsErrorWhenWriteFails(t *testing.T) {
	credentialsPath := filepath.Join(t.TempDir(), "credentials")
	source := NewFileSource(credentialsPath)

	err := os.MkdirAll(credentialsPath, 0o700)
	if err != nil {
		t.Fatalf("failed to create directory at credentials path: %v", err)
	}

	err = source.Store("DEMO_TOKEN", "value")
	if err == nil {
		t.Fatal("expected error when writing into directory path")
	}
}

func TestFileSourceStoreSortsKeysWhenWriting(t *testing.T) {
	credentialsPath := filepath.Join(t.TempDir(), "credentials")
	source := NewFileSource(credentialsPath)

	if err := source.Store("TOKEN_B", "value-b"); err != nil {
		t.Fatalf("expected store TOKEN_B to succeed: %v", err)
	}

	if err := source.Store("TOKEN_A", "value-a"); err != nil {
		t.Fatalf("expected store TOKEN_A to succeed: %v", err)
	}

	data, err := os.ReadFile(credentialsPath)
	if err != nil {
		t.Fatalf("expected credentials file to be readable: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least two lines, got %d", len(lines))
	}

	if lines[0] != "TOKEN_A=value-a" {
		t.Fatalf("expected first line TOKEN_A=value-a, got %q", lines[0])
	}

	if lines[1] != "TOKEN_B=value-b" {
		t.Fatalf("expected second line TOKEN_B=value-b, got %q", lines[1])
	}
}

func TestFileSourceStoreErrorIsNotErrNotSupported(t *testing.T) {
	credentialsPath := t.TempDir()
	source := NewFileSource(credentialsPath)
	err := source.Store("DEMO_TOKEN", "value")

	if err == nil {
		t.Fatal("expected error when path points to directory")
	}

	if errors.Is(err, ErrNotSupported) {
		t.Fatal("expected concrete storage error, not ErrNotSupported")
	}
}

func TestFileSourceDeleteRemovesSingleKey(t *testing.T) {
	credentialsPath := filepath.Join(t.TempDir(), "credentials")
	source := NewFileSource(credentialsPath)

	if err := source.Store("TOKEN_A", "value-a"); err != nil {
		t.Fatalf("expected TOKEN_A store to succeed: %v", err)
	}

	if err := source.Store("TOKEN_B", "value-b"); err != nil {
		t.Fatalf("expected TOKEN_B store to succeed: %v", err)
	}

	if err := source.Delete("TOKEN_A"); err != nil {
		t.Fatalf("expected delete to succeed: %v", err)
	}

	if _, found := source.Get("TOKEN_A"); found {
		t.Fatal("expected TOKEN_A to be removed")
	}

	if value, found := source.Get("TOKEN_B"); !found || value != "value-b" {
		t.Fatalf("expected TOKEN_B to remain, got %q (found=%v)", value, found)
	}
}

func TestFileSourceDeleteManyRemovesRequestedKeys(t *testing.T) {
	credentialsPath := filepath.Join(t.TempDir(), "credentials")
	source := NewFileSource(credentialsPath)

	if err := source.Store("TOKEN_A", "value-a"); err != nil {
		t.Fatalf("expected TOKEN_A store to succeed: %v", err)
	}

	if err := source.Store("TOKEN_B", "value-b"); err != nil {
		t.Fatalf("expected TOKEN_B store to succeed: %v", err)
	}

	if err := source.Store("TOKEN_C", "value-c"); err != nil {
		t.Fatalf("expected TOKEN_C store to succeed: %v", err)
	}

	err := source.DeleteMany("TOKEN_A", "TOKEN_C")
	if err != nil {
		t.Fatalf("expected delete many to succeed: %v", err)
	}

	if _, found := source.Get("TOKEN_A"); found {
		t.Fatal("expected TOKEN_A to be removed")
	}

	if _, found := source.Get("TOKEN_C"); found {
		t.Fatal("expected TOKEN_C to be removed")
	}

	if value, found := source.Get("TOKEN_B"); !found || value != "value-b" {
		t.Fatalf("expected TOKEN_B to remain, got %q (found=%v)", value, found)
	}
}

func TestFileSourceDeleteManyNoopsWhenFileMissing(t *testing.T) {
	source := NewFileSource(filepath.Join(t.TempDir(), "credentials"))

	err := source.DeleteMany("TOKEN_A", "TOKEN_B")
	if err != nil {
		t.Fatalf("expected delete many on missing file to succeed: %v", err)
	}
}

func TestFileSourceDeleteManyRejectsNilReceiver(t *testing.T) {
	var source *FileSource

	err := source.DeleteMany("TOKEN_A")
	if err == nil {
		t.Fatal("expected nil receiver to return error")
	}
}
