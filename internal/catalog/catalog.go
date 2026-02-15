package catalog

import (
	"sort"
	"strings"

	"github.com/andreagrandi/mcp-wire/internal/registry"
	"github.com/andreagrandi/mcp-wire/internal/service"
)

// Source identifies where a catalog entry originated.
type Source string

const (
	SourceCurated  Source = "curated"
	SourceRegistry Source = "registry"
)

// Entry wraps either a curated service or a registry server into a
// single type with explicit source metadata.
type Entry struct {
	Source   Source
	Name     string
	Curated  *service.Service
	Registry *registry.ServerResponse
}

// Catalog holds a merged collection of entries from all sources.
type Catalog struct {
	entries []Entry
}

// FromCurated converts a curated service into a catalog entry.
func FromCurated(svc service.Service) Entry {
	return Entry{
		Source:  SourceCurated,
		Name:    svc.Name,
		Curated: &svc,
	}
}

// FromCuratedMap converts a map of curated services into catalog entries.
func FromCuratedMap(services map[string]service.Service) []Entry {
	entries := make([]Entry, 0, len(services))
	for _, svc := range services {
		entries = append(entries, FromCurated(svc))
	}
	return entries
}

// FromRegistry converts a registry server response into a catalog entry.
func FromRegistry(resp registry.ServerResponse) Entry {
	return Entry{
		Source:   SourceRegistry,
		Name:     resp.Server.Name,
		Registry: &resp,
	}
}

// FromRegistrySlice converts a slice of registry server responses into catalog entries.
func FromRegistrySlice(servers []registry.ServerResponse) []Entry {
	entries := make([]Entry, 0, len(servers))
	for _, srv := range servers {
		entries = append(entries, FromRegistry(srv))
	}
	return entries
}

// DisplayName returns a human-friendly name for the entry.
func (e Entry) DisplayName() string {
	if e.Source == SourceCurated && e.Curated != nil {
		return e.Curated.Name
	}
	if e.Registry != nil {
		if e.Registry.Server.Title != "" {
			return e.Registry.Server.Title
		}
		return e.Registry.Server.Name
	}
	return e.Name
}

// Description returns the entry's description text.
func (e Entry) Description() string {
	if e.Source == SourceCurated && e.Curated != nil {
		return e.Curated.Description
	}
	if e.Registry != nil {
		return e.Registry.Server.Description
	}
	return ""
}

// EnvVars returns environment variables required by this entry.
func (e Entry) EnvVars() []service.EnvVar {
	if e.Source == SourceCurated && e.Curated != nil {
		return e.Curated.Env
	}
	if e.Registry != nil {
		return envVarsFromRegistry(e.Registry)
	}
	return nil
}

// Transport returns the transport type for this entry.
func (e Entry) Transport() string {
	if e.Source == SourceCurated && e.Curated != nil {
		return e.Curated.Transport
	}
	if e.Registry != nil {
		if len(e.Registry.Server.Remotes) > 0 {
			return e.Registry.Server.Remotes[0].Type
		}
		if len(e.Registry.Server.Packages) > 0 {
			return e.Registry.Server.Packages[0].Transport.Type
		}
	}
	return ""
}

// RepositoryURL returns the source repository URL, if available.
func (e Entry) RepositoryURL() string {
	if e.Registry != nil && e.Registry.Server.Repository != nil {
		return e.Registry.Server.Repository.URL
	}
	return ""
}

// WebsiteURL returns the server's website URL, if available.
func (e Entry) WebsiteURL() string {
	if e.Registry != nil {
		return e.Registry.Server.WebsiteURL
	}
	return ""
}

// HasRemotes reports whether this entry has remote (HTTP/SSE) transports.
func (e Entry) HasRemotes() bool {
	if e.Source == SourceCurated && e.Curated != nil {
		t := strings.ToLower(e.Curated.Transport)
		return t == "http" || t == "sse"
	}
	if e.Registry != nil {
		return len(e.Registry.Server.Remotes) > 0
	}
	return false
}

// HasPackages reports whether this entry has package-based install methods.
func (e Entry) HasPackages() bool {
	if e.Registry != nil {
		return len(e.Registry.Server.Packages) > 0
	}
	return false
}

// InstallType returns "remote", "package", or "remote/package"
// describing how this entry would be installed.
func (e Entry) InstallType() string {
	hasRemotes := e.HasRemotes()
	hasPackages := e.HasPackages()
	if hasRemotes && hasPackages {
		return "remote/package"
	}
	if hasRemotes {
		return "remote"
	}
	if hasPackages {
		return "package"
	}
	return ""
}

// PackageTypes returns the package registry types (e.g., "npm", "pypi")
// for registry entries. Returns nil for curated entries.
func (e Entry) PackageTypes() []string {
	if e.Registry == nil {
		return nil
	}
	var types []string
	seen := make(map[string]bool)
	for _, pkg := range e.Registry.Server.Packages {
		if pkg.RegistryType != "" && !seen[pkg.RegistryType] {
			types = append(types, pkg.RegistryType)
			seen[pkg.RegistryType] = true
		}
	}
	return types
}

// envVarsFromRegistry extracts environment variables from a registry
// server response, combining package env vars and secret remote headers.
func envVarsFromRegistry(resp *registry.ServerResponse) []service.EnvVar {
	index := make(map[string]int) // name -> position in vars
	var vars []service.EnvVar

	merge := func(name, desc string, required bool) {
		if name == "" {
			return
		}
		if i, ok := index[name]; ok {
			vars[i].Required = vars[i].Required || required
			if vars[i].Description == "" && desc != "" {
				vars[i].Description = desc
			}
			return
		}
		index[name] = len(vars)
		vars = append(vars, service.EnvVar{
			Name:        name,
			Description: desc,
			Required:    required,
		})
	}

	for _, pkg := range resp.Server.Packages {
		for _, ev := range pkg.EnvironmentVariables {
			merge(ev.Name, ev.Description, ev.IsRequired)
		}
	}

	for _, remote := range resp.Server.Remotes {
		for _, hdr := range remote.Headers {
			if !hdr.IsSecret {
				continue
			}
			merge(hdr.Name, hdr.Description, hdr.IsRequired)
		}
	}

	return vars
}

// Merge creates a catalog from curated and registry entries. On
// case-insensitive name collision, curated entries take precedence.
func Merge(curated, reg []Entry) *Catalog {
	seen := make(map[string]bool)
	var merged []Entry

	for _, e := range curated {
		key := strings.ToLower(e.Name)
		seen[key] = true
		merged = append(merged, e)
	}

	for _, e := range reg {
		key := strings.ToLower(e.Name)
		if seen[key] {
			continue
		}
		seen[key] = true
		merged = append(merged, e)
	}

	return &Catalog{entries: merged}
}

// All returns all entries sorted by name.
func (c *Catalog) All() []Entry {
	cp := make([]Entry, len(c.entries))
	copy(cp, c.entries)
	sort.Slice(cp, func(i, j int) bool {
		return strings.ToLower(cp[i].Name) < strings.ToLower(cp[j].Name)
	})
	return cp
}

// BySource returns entries matching the given source, sorted by name.
func (c *Catalog) BySource(source Source) []Entry {
	var filtered []Entry
	for _, e := range c.entries {
		if e.Source == source {
			filtered = append(filtered, e)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return strings.ToLower(filtered[i].Name) < strings.ToLower(filtered[j].Name)
	})
	return filtered
}

// Search returns entries whose name, display name, or description
// contain the query string (case-insensitive). An empty query returns all.
func (c *Catalog) Search(query string) []Entry {
	if query == "" {
		return c.All()
	}
	q := strings.ToLower(query)
	var results []Entry
	for _, e := range c.entries {
		if strings.Contains(strings.ToLower(e.Name), q) ||
			strings.Contains(strings.ToLower(e.DisplayName()), q) ||
			strings.Contains(strings.ToLower(e.Description()), q) {
			results = append(results, e)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
	})
	return results
}

// Find performs a case-insensitive exact match on entry name.
func (c *Catalog) Find(name string) (Entry, bool) {
	target := strings.ToLower(name)
	for _, e := range c.entries {
		if strings.ToLower(e.Name) == target {
			return e, true
		}
	}
	return Entry{}, false
}

// Count returns the number of entries in the catalog.
func (c *Catalog) Count() int {
	return len(c.entries)
}
