package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/andreagrandi/mcp-wire/internal/app"
	"github.com/andreagrandi/mcp-wire/internal/catalog"
	"github.com/andreagrandi/mcp-wire/internal/config"
	"github.com/andreagrandi/mcp-wire/internal/service"
	"github.com/andreagrandi/mcp-wire/internal/target"
	"github.com/spf13/cobra"
)

// metadataSchemaVersion identifies the structure of the metadata document.
// Bump it only on breaking changes to the JSON shape so automation can detect
// incompatibilities.
const metadataSchemaVersion = 1

// supportedTransports lists the transport types mcp-wire can install.
var supportedTransports = []string{"http", "sse", "stdio"}

// installableScopes lists the config scopes accepted by install and uninstall.
var installableScopes = []string{
	string(target.ConfigScopeUser),
	string(target.ConfigScopeProject),
}

// metadataDocument is the top-level machine-readable capability report.
type metadataDocument struct {
	SchemaVersion  int               `json:"schema_version"`
	MCPWireVersion string            `json:"mcp_wire_version"`
	Transports     []string          `json:"transports"`
	Scopes         []string          `json:"scopes"`
	Targets        []metadataTarget  `json:"targets"`
	Services       []metadataService `json:"services"`
	Features       []metadataFeature `json:"features"`
}

// metadataTarget describes a target tool and where it writes configuration.
type metadataTarget struct {
	Name          string   `json:"name"`
	Slug          string   `json:"slug"`
	Installed     bool     `json:"installed"`
	ConfigPath    string   `json:"config_path,omitempty"`
	Scopes        []string `json:"scopes"`
	SupportsOAuth bool     `json:"supports_oauth"`
}

// metadataService describes an installable service and what it needs.
type metadataService struct {
	Name          string           `json:"name"`
	DisplayName   string           `json:"display_name,omitempty"`
	Description   string           `json:"description"`
	Source        string           `json:"source"`
	Transport     string           `json:"transport"`
	Auth          string           `json:"auth"`
	InstallMethod string           `json:"install_method,omitempty"`
	Env           []metadataEnvVar `json:"env"`
}

// metadataEnvVar describes a credential or environment variable a service needs.
type metadataEnvVar struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
	SetupURL    string `json:"setup_url,omitempty"`
	SetupHint   string `json:"setup_hint,omitempty"`
}

// metadataFeature describes a feature flag and its current state.
type metadataFeature struct {
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}

// metadataDeps wires the data sources metadata reads, so tests can substitute them.
type metadataDeps struct {
	loadConfig  func() (*config.Config, error)
	allTargets  func() []target.Target
	loadCatalog func(source string, registryEnabled bool) (*catalog.Catalog, error)
	version     string
}

func defaultMetadataDeps() metadataDeps {
	return metadataDeps{
		loadConfig:  loadConfig,
		allTargets:  allTargets,
		loadCatalog: loadCatalog,
		version:     app.Version,
	}
}

func init() {
	rootCmd.AddCommand(newMetadataCmd())
}

func newMetadataCmd() *cobra.Command {
	var source string

	cmd := &cobra.Command{
		Use:   "metadata",
		Short: "Print machine-readable capability metadata as JSON",
		Long: `metadata prints a stable JSON document describing what mcp-wire can do:
supported transports and scopes, every target (install status, config path,
supported scopes, OAuth support), every service in the selected source
(transport, auth requirements, environment variables), and feature-flag state.

The output is intended for automation and AI agents that need to inspect
mcp-wire's capabilities without driving the interactive UI. The top-level
"schema_version" field identifies the document structure and is only bumped on
breaking changes.

By default only curated services are included. Use --source registry or
--source all to include registry services (requires the registry feature).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validateSource(source); err != nil {
				return err
			}

			return runMetadata(cmd.OutOrStdout(), defaultMetadataDeps(), source)
		},
	}

	cmd.Flags().StringVar(&source, "source", "curated", "Service source: curated, registry, or all")

	return cmd
}

func runMetadata(output io.Writer, deps metadataDeps, source string) error {
	doc, err := buildMetadataDocument(deps, source)
	if err != nil {
		return err
	}

	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode metadata: %w", err)
	}

	encoded = append(encoded, '\n')

	if _, err := output.Write(encoded); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}

	return nil
}

func buildMetadataDocument(deps metadataDeps, source string) (metadataDocument, error) {
	cfg, err := deps.loadConfig()
	if err != nil {
		return metadataDocument{}, fmt.Errorf("load config: %w", err)
	}

	registryEnabled := cfg.IsFeatureEnabled("registry")

	cat, err := deps.loadCatalog(source, registryEnabled)
	if err != nil {
		return metadataDocument{}, err
	}

	return metadataDocument{
		SchemaVersion:  metadataSchemaVersion,
		MCPWireVersion: deps.version,
		Transports:     append([]string(nil), supportedTransports...),
		Scopes:         append([]string(nil), installableScopes...),
		Targets:        buildMetadataTargets(deps.allTargets()),
		Services:       buildMetadataServices(cat),
		Features:       buildMetadataFeatures(cfg),
	}, nil
}

func buildMetadataTargets(targets []target.Target) []metadataTarget {
	result := make([]metadataTarget, 0, len(targets))

	for _, t := range targets {
		configPath, _ := targetConfigPath(t)

		result = append(result, metadataTarget{
			Name:          t.Name(),
			Slug:          t.Slug(),
			Installed:     t.IsInstalled(),
			ConfigPath:    configPath,
			Scopes:        targetScopes(t),
			SupportsOAuth: targetSupportsOAuth(t),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Slug < result[j].Slug
	})

	return result
}

func buildMetadataServices(cat *catalog.Catalog) []metadataService {
	entries := cat.All()
	services := make([]metadataService, 0, len(entries))

	for _, entry := range entries {
		displayName := entry.DisplayName()
		if displayName == entry.Name {
			displayName = ""
		}

		services = append(services, metadataService{
			Name:          entry.Name,
			DisplayName:   displayName,
			Description:   strings.TrimSpace(entry.Description()),
			Source:        string(entry.Source),
			Transport:     entry.Transport(),
			Auth:          normalizeAuthLabel(entry.AuthLabel()),
			InstallMethod: entry.InstallMethodLabel(),
			Env:           buildMetadataEnv(entry.EnvVars()),
		})
	}

	return services
}

func buildMetadataEnv(vars []service.EnvVar) []metadataEnvVar {
	env := make([]metadataEnvVar, 0, len(vars))

	for _, v := range vars {
		env = append(env, metadataEnvVar{
			Name:        v.Name,
			Description: strings.TrimSpace(v.Description),
			Required:    v.Required,
			SetupURL:    v.SetupURL,
			SetupHint:   v.SetupHint,
		})
	}

	return env
}

func buildMetadataFeatures(cfg *config.Config) []metadataFeature {
	statuses := cfg.Features()
	features := make([]metadataFeature, 0, len(statuses))

	for _, status := range statuses {
		features = append(features, metadataFeature{
			Name:        status.Name,
			Enabled:     status.Enabled,
			Description: status.Description,
		})
	}

	return features
}

func targetScopes(t target.Target) []string {
	scopedTarget, ok := t.(target.ScopedTarget)
	if !ok {
		return []string{string(target.ConfigScopeUser)}
	}

	supported := scopedTarget.SupportedScopes()
	scopes := make([]string, 0, len(supported))
	for _, scope := range supported {
		scopes = append(scopes, string(scope))
	}

	return scopes
}

func targetSupportsOAuth(t target.Target) bool {
	_, ok := t.(target.AuthTarget)
	return ok
}

// normalizeAuthLabel maps the human-facing auth label to a stable token
// suitable for automation: "oauth", "api_key", or "none".
func normalizeAuthLabel(label string) string {
	switch label {
	case "OAuth":
		return "oauth"
	case "API key":
		return "api_key"
	default:
		return "none"
	}
}
