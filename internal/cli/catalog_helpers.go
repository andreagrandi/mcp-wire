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

var fetchServerLatest = defaultFetchServerLatest

func defaultFetchServerLatest(serverName string) (*registry.ServerResponse, error) {
	client := registry.NewClient()
	return client.GetServerLatest(serverName)
}

// refreshRegistryEntry fetches the latest version details for a registry
// catalog entry. It returns the updated entry on success, or the original
// entry unchanged on network/API errors (graceful degradation).
func refreshRegistryEntry(entry catalog.Entry) catalog.Entry {
	if entry.Source != catalog.SourceRegistry || entry.Registry == nil {
		return entry
	}

	resp, err := fetchServerLatest(entry.Registry.Server.Name)
	if err != nil || resp == nil {
		return entry
	}

	return catalog.Entry{
		Source:   catalog.SourceRegistry,
		Name:     entry.Name,
		Registry: resp,
	}
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
	if entry.HasPackages() {
		pkg := entry.Registry.Server.Packages[0]
		identifier := pkg.Identifier
		if pkg.Version != "" {
			identifier += "@" + pkg.Version
		}

		fmt.Fprintf(output, "  Package:   %s (%s)\n", pkg.RegistryType, identifier)

		if pkg.RuntimeHint != "" {
			fmt.Fprintf(output, "  Runtime:   %s\n", pkg.RuntimeHint)
		}
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

	if entry.Source == catalog.SourceRegistry && entry.Registry != nil {
		if svc, ok := registryRemoteToService(entry); ok {
			return svc, true
		}

		return registryPackageToService(entry)
	}

	return service.Service{}, false
}

func registryRemoteToService(entry catalog.Entry) (service.Service, bool) {
	if entry.Registry == nil || len(entry.Registry.Server.Remotes) == 0 {
		return service.Service{}, false
	}

	var remote registry.Transport
	found := false
	for _, r := range entry.Registry.Server.Remotes {
		t := strings.ToLower(r.Type)
		if t == "streamable-http" || t == "sse" {
			remote = r
			found = true
			break
		}
	}

	if !found {
		return service.Service{}, false
	}

	transport := strings.ToLower(remote.Type)
	if transport == "streamable-http" {
		transport = "http"
	}

	var envVars []service.EnvVar
	seen := map[string]int{} // name -> index in envVars

	addVar := func(name, description, defaultVal string, required bool) {
		if name == "" {
			return
		}
		if i, exists := seen[name]; exists {
			envVars[i].Required = envVars[i].Required || required
			if envVars[i].Description == "" && description != "" {
				envVars[i].Description = description
			}
			return
		}
		seen[name] = len(envVars)
		envVars = append(envVars, service.EnvVar{
			Name:        name,
			Description: description,
			Required:    required,
			Default:     defaultVal,
		})
	}

	for varName, field := range remote.Variables {
		addVar(varName, field.Description, field.Default, field.IsRequired)
	}

	headers := make(map[string]string, len(remote.Headers))
	for _, hdr := range remote.Headers {
		if hdr.Value != "" {
			headers[hdr.Name] = hdr.Value
			for varName, field := range hdr.Variables {
				addVar(varName, field.Description, field.Default, field.IsRequired)
			}
		} else if hdr.IsSecret || hdr.IsRequired {
			placeholder := "{" + hdr.Name + "}"
			headers[hdr.Name] = placeholder
			addVar(hdr.Name, hdr.Description, hdr.Default, hdr.IsRequired)
		} else {
			headers[hdr.Name] = hdr.Default
		}
	}

	svc := service.Service{
		Name:        entry.Registry.Server.Name,
		Description: entry.Registry.Server.Description,
		Transport:   transport,
		URL:         remote.URL,
		Env:         envVars,
		Headers:     headers,
	}

	return svc, true
}

func registryPackageToService(entry catalog.Entry) (service.Service, bool) {
	if entry.Registry == nil || len(entry.Registry.Server.Packages) == 0 {
		return service.Service{}, false
	}

	// Find the first package with a supported registry type.
	var pkg registry.Package
	found := false
	for _, candidate := range entry.Registry.Server.Packages {
		if _, _, ok := packageRunCommand(candidate, nil); ok {
			pkg = candidate
			found = true
			break
		}
	}

	if !found {
		return service.Service{}, false
	}

	var envVars []service.EnvVar
	seen := map[string]int{}

	addVar := func(name, description, defaultVal string, required bool) {
		if name == "" {
			return
		}

		if i, exists := seen[name]; exists {
			envVars[i].Required = envVars[i].Required || required
			if envVars[i].Description == "" && description != "" {
				envVars[i].Description = description
			}

			return
		}

		seen[name] = len(envVars)
		envVars = append(envVars, service.EnvVar{
			Name:        name,
			Description: description,
			Required:    required,
			Default:     defaultVal,
		})
	}

	command, baseArgs, _ := packageRunCommand(pkg, addVar)

	runtimeArgs := resolvePackageArguments(pkg.RuntimeArguments, addVar)
	args := append(baseArgs, runtimeArgs...)

	for _, ev := range pkg.EnvironmentVariables {
		addVar(ev.Name, ev.Description, ev.Default, ev.IsRequired)
	}

	svc := service.Service{
		Name:        entry.Registry.Server.Name,
		Description: entry.Registry.Server.Description,
		Transport:   "stdio",
		Command:     command,
		Args:        args,
		Env:         envVars,
	}

	return svc, true
}

type addVarFunc func(name, desc, defaultVal string, required bool)

func packageRunCommand(pkg registry.Package, addVar addVarFunc) (string, []string, bool) {
	switch strings.ToLower(pkg.RegistryType) {
	case "npm":
		return npmRunCommand(pkg, addVar)
	case "pypi":
		return pypiRunCommand(pkg, addVar)
	case "docker", "oci":
		return dockerRunCommand(pkg, addVar)
	case "nuget":
		return nugetRunCommand(pkg, addVar)
	case "mcpb":
		return mcpbRunCommand(pkg, addVar)
	default:
		return "", nil, false
	}
}

func npmRunCommand(pkg registry.Package, addVar addVarFunc) (string, []string, bool) {
	identifier := strings.TrimSpace(pkg.Identifier)
	if identifier == "" {
		return "", nil, false
	}

	if v := strings.TrimSpace(pkg.Version); v != "" {
		identifier = identifier + "@" + v
	}

	args := []string{"-y"}
	args = append(args, resolvePackageArguments(pkg.PackageArguments, addVar)...)
	args = append(args, identifier)

	return "npx", args, true
}

func pypiRunCommand(pkg registry.Package, addVar addVarFunc) (string, []string, bool) {
	identifier := strings.TrimSpace(pkg.Identifier)
	if identifier == "" {
		return "", nil, false
	}

	if v := strings.TrimSpace(pkg.Version); v != "" {
		identifier = identifier + "@" + v
	}

	args := resolvePackageArguments(pkg.PackageArguments, addVar)
	args = append(args, identifier)

	return "uvx", args, true
}

func dockerRunCommand(pkg registry.Package, addVar addVarFunc) (string, []string, bool) {
	identifier := strings.TrimSpace(pkg.Identifier)
	if identifier == "" {
		return "", nil, false
	}

	if v := strings.TrimSpace(pkg.Version); v != "" {
		identifier = identifier + ":" + v
	}

	args := []string{"run", "-i", "--rm"}
	args = append(args, resolvePackageArguments(pkg.PackageArguments, addVar)...)
	args = append(args, identifier)

	return "docker", args, true
}

func nugetRunCommand(pkg registry.Package, addVar addVarFunc) (string, []string, bool) {
	identifier := strings.TrimSpace(pkg.Identifier)
	if identifier == "" {
		return "", nil, false
	}

	args := []string{"tool", "run", identifier}
	args = append(args, resolvePackageArguments(pkg.PackageArguments, addVar)...)

	return "dotnet", args, true
}

func mcpbRunCommand(pkg registry.Package, addVar addVarFunc) (string, []string, bool) {
	identifier := strings.TrimSpace(pkg.Identifier)
	if identifier == "" {
		return "", nil, false
	}

	args := []string{"run", identifier}
	args = append(args, resolvePackageArguments(pkg.PackageArguments, addVar)...)

	return "mcpb", args, true
}

func resolvePackageArguments(args []registry.Argument, addVar func(name, desc, defaultVal string, required bool)) []string {
	var result []string

	for _, arg := range args {
		if arg.Value != "" {
			result = append(result, arg.Value)
			continue
		}

		if arg.Name != "" && addVar != nil {
			addVar(arg.Name, arg.Description, arg.Default, arg.IsRequired)
			result = append(result, "{"+arg.Name+"}")
		}
	}

	return result
}
