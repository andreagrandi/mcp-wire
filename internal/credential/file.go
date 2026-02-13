package credential

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	credentialsFileName = "credentials"
	fileSourceName      = "file"
)

// FileSource resolves and stores credentials in a local file.
type FileSource struct {
	path string
}

// NewFileSource creates a source backed by a credentials file.
//
// If path is empty, it defaults to ~/.config/mcp-wire/credentials.
func NewFileSource(path string) *FileSource {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		trimmedPath = defaultCredentialsFilePath()
	}

	return &FileSource{path: trimmedPath}
}

// Name returns a stable source name.
func (s *FileSource) Name() string {
	return fileSourceName
}

// Get returns the credential value when present in the file.
func (s *FileSource) Get(envName string) (string, bool) {
	if s == nil {
		return "", false
	}

	trimmedName := strings.TrimSpace(envName)
	if trimmedName == "" {
		return "", false
	}

	entries, err := s.readAll()
	if err != nil {
		return "", false
	}

	value, found := entries[trimmedName]
	return value, found
}

// Store saves or updates a credential in the file.
func (s *FileSource) Store(envName string, value string) error {
	if s == nil {
		return errors.New("file source is nil")
	}

	trimmedName := strings.TrimSpace(envName)
	if trimmedName == "" {
		return errors.New("environment variable name is required")
	}

	entries, err := s.readAll()
	if err != nil {
		return err
	}

	entries[trimmedName] = value

	return s.writeAll(entries)
}

// Delete removes a credential key from the file.
func (s *FileSource) Delete(envName string) error {
	return s.DeleteMany(envName)
}

// DeleteMany removes multiple credential keys from the file.
func (s *FileSource) DeleteMany(envNames ...string) error {
	if s == nil {
		return errors.New("file source is nil")
	}

	if len(envNames) == 0 {
		return nil
	}

	_, err := os.Stat(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("stat credentials file %q: %w", s.path, err)
	}

	entries, err := s.readAll()
	if err != nil {
		return err
	}

	hasChanges := false
	for _, rawName := range envNames {
		name := strings.TrimSpace(rawName)
		if name == "" {
			continue
		}

		if _, exists := entries[name]; !exists {
			continue
		}

		delete(entries, name)
		hasChanges = true
	}

	if !hasChanges {
		return nil
	}

	return s.writeAll(entries)
}

func (s *FileSource) readAll() (map[string]string, error) {
	entries := map[string]string{}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return entries, nil
		}

		return nil, fmt.Errorf("read credentials file %q: %w", s.path, err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		name, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" {
			continue
		}

		entries[trimmedName] = strings.TrimSpace(rawValue)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan credentials file %q: %w", s.path, err)
	}

	return entries, nil
}

func (s *FileSource) writeAll(entries map[string]string) error {
	credentialsDir := filepath.Dir(s.path)
	if err := os.MkdirAll(credentialsDir, 0o700); err != nil {
		return fmt.Errorf("create credentials directory %q: %w", credentialsDir, err)
	}

	keys := make([]string, 0, len(entries))
	for name := range entries {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for _, name := range keys {
		builder.WriteString(name)
		builder.WriteByte('=')
		builder.WriteString(entries[name])
		builder.WriteByte('\n')
	}

	if err := os.WriteFile(s.path, []byte(builder.String()), 0o600); err != nil {
		return fmt.Errorf("write credentials file %q: %w", s.path, err)
	}

	if err := os.Chmod(s.path, 0o600); err != nil {
		return fmt.Errorf("set credentials file permissions on %q: %w", s.path, err)
	}

	return nil
}

func defaultCredentialsFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "mcp-wire", credentialsFileName)
	}

	return filepath.Join(homeDir, ".config", "mcp-wire", credentialsFileName)
}
