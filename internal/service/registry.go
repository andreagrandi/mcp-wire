package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	bundledservices "github.com/andreagrandi/mcp-wire/services"
	"gopkg.in/yaml.v3"
)

// LoadServices loads service definitions from one or more directories.
//
// If no paths are provided, default locations are used in this order:
//  1. services/ relative to the executable
//  2. services/ relative to the current working directory
//  3. ~/.config/mcp-wire/services
//
// When multiple files define the same service name, the last loaded definition
// wins. With default paths, this means user-local definitions override bundled
// ones.
func LoadServices(paths ...string) (map[string]Service, error) {
	loadBundledDefaults := len(paths) == 0

	loadPaths, err := resolveServicePaths(paths...)
	if err != nil {
		return nil, err
	}

	services := make(map[string]Service)
	if loadBundledDefaults {
		if err := loadEmbeddedServices(services); err != nil {
			return nil, err
		}
	}

	for _, rawPath := range loadPaths {
		path, err := expandHome(rawPath)
		if err != nil {
			return nil, fmt.Errorf("expand services path %q: %w", rawPath, err)
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return nil, fmt.Errorf("read services directory %q: %w", path, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
				continue
			}

			filePath := filepath.Join(path, entry.Name())
			service, err := loadServiceFile(filePath)
			if err != nil {
				return nil, err
			}

			services[service.Name] = service
		}
	}

	return services, nil
}

// ValidateService validates required fields for a service definition.
func ValidateService(s Service) error {
	name := strings.TrimSpace(s.Name)
	if name == "" {
		return errors.New("service name is required")
	}

	transport := strings.ToLower(strings.TrimSpace(s.Transport))
	if transport == "" {
		return fmt.Errorf("service %q transport is required", name)
	}

	switch transport {
	case "sse":
		if strings.TrimSpace(s.URL) == "" {
			return fmt.Errorf("service %q with sse transport requires url", name)
		}
	case "stdio":
		if strings.TrimSpace(s.Command) == "" {
			return fmt.Errorf("service %q with stdio transport requires command", name)
		}
	default:
		return fmt.Errorf("service %q has unsupported transport %q", name, s.Transport)
	}

	return nil
}

func resolveServicePaths(paths ...string) ([]string, error) {
	if len(paths) > 0 {
		return paths, nil
	}

	binaryPath := "services"
	executablePath, err := os.Executable()
	if err == nil {
		binaryPath = filepath.Join(filepath.Dir(executablePath), "services")
	}

	loadPaths := []string{binaryPath}

	workingDirectory, err := os.Getwd()
	if err == nil {
		loadPaths = append(loadPaths, filepath.Join(workingDirectory, "services"))
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return dedupePaths(loadPaths), nil
	}

	loadPaths = append(loadPaths, filepath.Join(homeDir, ".config", "mcp-wire", "services"))

	return dedupePaths(loadPaths), nil
}

func dedupePaths(paths []string) []string {
	seenPaths := make(map[string]struct{}, len(paths))
	uniquePaths := make([]string, 0, len(paths))

	for _, path := range paths {
		normalizedPath := filepath.Clean(path)
		if normalizedPath == "" {
			continue
		}

		if _, seen := seenPaths[normalizedPath]; seen {
			continue
		}

		seenPaths[normalizedPath] = struct{}{}
		uniquePaths = append(uniquePaths, path)
	}

	return uniquePaths
}

func loadServiceFile(path string) (Service, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Service{}, fmt.Errorf("read service file %q: %w", path, err)
	}

	return parseServiceDefinition(path, data)
}

func loadEmbeddedServices(services map[string]Service) error {
	entries, err := bundledservices.FS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read embedded services: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		filePath := entry.Name()
		data, err := bundledservices.FS.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read embedded service file %q: %w", filePath, err)
		}

		service, err := parseServiceDefinition("embedded/"+filePath, data)
		if err != nil {
			return err
		}

		services[service.Name] = service
	}

	return nil
}

func parseServiceDefinition(path string, data []byte) (Service, error) {

	var service Service
	if err := yaml.Unmarshal(data, &service); err != nil {
		return Service{}, fmt.Errorf("parse service file %q: %w", path, err)
	}

	service = normalizeService(service)

	if err := ValidateService(service); err != nil {
		return Service{}, fmt.Errorf("validate service file %q: %w", path, err)
	}

	return service, nil
}

func normalizeService(s Service) Service {
	s.Name = strings.TrimSpace(s.Name)
	s.Description = strings.TrimSpace(s.Description)
	s.Transport = strings.ToLower(strings.TrimSpace(s.Transport))
	s.URL = strings.TrimSpace(s.URL)
	s.Command = strings.TrimSpace(s.Command)

	return s
}

func expandHome(path string) (string, error) {
	if path == "~" {
		return os.UserHomeDir()
	}

	if !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, "~\\") {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	relativePath := path[2:]
	return filepath.Join(homeDir, relativePath), nil
}
