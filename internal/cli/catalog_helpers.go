package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/andreagrandi/mcp-wire/internal/catalog"
	"github.com/andreagrandi/mcp-wire/internal/registry"
	"github.com/andreagrandi/mcp-wire/internal/service"
)

var loadRegistryCache = defaultLoadRegistryCache

func defaultLoadRegistryCache() []registry.ServerResponse {
	cache := registry.NewCache(nil)
	if err := cache.Load(); err != nil {
		return nil
	}

	return cache.All()
}

func loadCatalog(source string, registryEnabled bool) (*catalog.Catalog, error) {
	var curatedEntries []catalog.Entry
	var registryEntries []catalog.Entry

	if source != "registry" {
		services, err := loadServices()
		if err != nil {
			return nil, fmt.Errorf("load services: %w", err)
		}

		curatedEntries = catalog.FromCuratedMap(services)
	}

	if registryEnabled && (source == "registry" || source == "all") {
		servers := loadRegistryCache()
		registryEntries = catalog.FromRegistrySlice(servers)
	}

	return catalog.Merge(curatedEntries, registryEntries), nil
}

func printCatalogEntries(output io.Writer, entries []catalog.Entry, showMarkers bool) {
	fmt.Fprintln(output, "Available services:")

	if showMarkers {
		fmt.Fprintln(output, "  (* = curated by mcp-wire)")
	}

	fmt.Fprintln(output)

	if len(entries) == 0 {
		fmt.Fprintln(output, "  (none)")
		return
	}

	sorted := make([]catalog.Entry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})

	maxNameWidth := 0
	for _, e := range sorted {
		label := e.Name
		if showMarkers {
			label = "  " + label
		}

		if len(label) > maxNameWidth {
			maxNameWidth = len(label)
		}
	}

	for _, e := range sorted {
		prefix := ""
		if showMarkers && e.Source == catalog.SourceCurated {
			prefix = "* "
		} else if showMarkers {
			prefix = "  "
		}

		label := prefix + e.Name

		description := strings.TrimSpace(e.Description())
		if description == "" {
			fmt.Fprintf(output, "  %s\n", label)
			continue
		}

		fmt.Fprintf(output, "  %-*s  %s\n", maxNameWidth, label, description)
	}
}

func sourceLabel(source string) string {
	switch source {
	case "curated":
		return "curated"
	case "registry":
		return "registry"
	default:
		return "all"
	}
}

var validSources = map[string]bool{
	"curated":  true,
	"registry": true,
	"all":      true,
}

func validateSource(source string) error {
	if !validSources[source] {
		return fmt.Errorf("invalid --source value %q (valid: curated, registry, all)", source)
	}

	return nil
}

func printRegistryTrustSummary(output io.Writer, entry catalog.Entry) {
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Registry Service Information:")
	fmt.Fprintf(output, "  Source:    %s (community, not vetted by mcp-wire)\n", entry.Source)
	if installType := entry.InstallType(); installType != "" {
		fmt.Fprintf(output, "  Install:   %s\n", installType)
	}
	if transport := entry.Transport(); transport != "" {
		fmt.Fprintf(output, "  Transport: %s\n", transport)
	}
	var secretNames []string
	for _, v := range entry.EnvVars() {
		if v.Required {
			secretNames = append(secretNames, v.Name)
		}
	}
	if len(secretNames) > 0 {
		fmt.Fprintf(output, "  Secrets:   %s\n", strings.Join(secretNames, ", "))
	}
	if repoURL := entry.RepositoryURL(); repoURL != "" {
		fmt.Fprintf(output, "  Repo:      %s\n", repoURL)
	}
	fmt.Fprintln(output)
}

func catalogEntryToService(entry catalog.Entry) (service.Service, bool) {
	if entry.Source == catalog.SourceCurated && entry.Curated != nil {
		return *entry.Curated, true
	}

	return service.Service{}, false
}
