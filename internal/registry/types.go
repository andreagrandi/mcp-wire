package registry

import "time"

// ServerListResponse is the top-level response from the list/search endpoint.
type ServerListResponse struct {
	Servers  []ServerResponse `json:"servers"`
	Metadata Metadata         `json:"metadata"`
}

// Metadata contains pagination information.
type Metadata struct {
	Count      int    `json:"count"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// ServerResponse wraps a server definition with registry metadata.
type ServerResponse struct {
	Server ServerJSON   `json:"server"`
	Meta   ResponseMeta `json:"_meta"`
}

// ServerJSON is the server definition as published to the registry.
type ServerJSON struct {
	Schema      string      `json:"$schema"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Version     string      `json:"version"`
	Title       string      `json:"title,omitempty"`
	WebsiteURL  string      `json:"websiteUrl,omitempty"`
	Icons       []Icon      `json:"icons,omitempty"`
	Repository  *Repository `json:"repository,omitempty"`
	Packages    []Package   `json:"packages,omitempty"`
	Remotes     []Transport `json:"remotes,omitempty"`
}

// ResponseMeta holds registry-managed metadata.
type ResponseMeta struct {
	Official *RegistryExtensions `json:"io.modelcontextprotocol.registry/official,omitempty"`
}

// RegistryExtensions contains registry lifecycle metadata.
type RegistryExtensions struct {
	Status      string    `json:"status"`
	PublishedAt time.Time `json:"publishedAt"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty"`
	IsLatest    bool      `json:"isLatest"`
}

// Icon describes a server icon.
type Icon struct {
	Src      string   `json:"src"`
	MimeType string   `json:"mimeType,omitempty"`
	Sizes    []string `json:"sizes,omitempty"`
	Theme    string   `json:"theme,omitempty"`
}

// Repository contains source code repository metadata.
type Repository struct {
	URL       string `json:"url,omitempty"`
	Source    string `json:"source,omitempty"`
	ID        string `json:"id,omitempty"`
	Subfolder string `json:"subfolder,omitempty"`
}

// Package describes a package-backed install method.
type Package struct {
	RegistryType         string          `json:"registryType"`
	Identifier           string          `json:"identifier"`
	Transport            Transport       `json:"transport"`
	Version              string          `json:"version,omitempty"`
	EnvironmentVariables []KeyValueInput `json:"environmentVariables,omitempty"`
	PackageArguments     []Argument      `json:"packageArguments,omitempty"`
	RuntimeArguments     []Argument      `json:"runtimeArguments,omitempty"`
	RuntimeHint          string          `json:"runtimeHint,omitempty"`
	RegistryBaseURL      string          `json:"registryBaseUrl,omitempty"`
}

// Transport describes a transport protocol configuration.
type Transport struct {
	Type      string                `json:"type"`
	URL       string                `json:"url,omitempty"`
	Headers   []KeyValueInput       `json:"headers,omitempty"`
	Variables map[string]InputField `json:"variables,omitempty"`
}

// KeyValueInput describes a named environment variable or header.
type KeyValueInput struct {
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	IsRequired  bool                  `json:"isRequired,omitempty"`
	IsSecret    bool                  `json:"isSecret,omitempty"`
	Value       string                `json:"value,omitempty"`
	Default     string                `json:"default,omitempty"`
	Placeholder string                `json:"placeholder,omitempty"`
	Format      string                `json:"format,omitempty"`
	Choices     []string              `json:"choices,omitempty"`
	Variables   map[string]InputField `json:"variables,omitempty"`
}

// InputField describes a variable input for URL templating.
type InputField struct {
	Description string   `json:"description,omitempty"`
	IsRequired  bool     `json:"isRequired,omitempty"`
	IsSecret    bool     `json:"isSecret,omitempty"`
	Value       string   `json:"value,omitempty"`
	Default     string   `json:"default,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Format      string   `json:"format,omitempty"`
	Choices     []string `json:"choices,omitempty"`
}

// Argument describes a command-line argument for a package.
type Argument struct {
	Type        string                `json:"type"`
	Name        string                `json:"name,omitempty"`
	Value       string                `json:"value,omitempty"`
	ValueHint   string                `json:"valueHint,omitempty"`
	Description string                `json:"description,omitempty"`
	IsRequired  bool                  `json:"isRequired,omitempty"`
	IsSecret    bool                  `json:"isSecret,omitempty"`
	IsRepeated  bool                  `json:"isRepeated,omitempty"`
	Default     string                `json:"default,omitempty"`
	Placeholder string                `json:"placeholder,omitempty"`
	Format      string                `json:"format,omitempty"`
	Choices     []string              `json:"choices,omitempty"`
	Variables   map[string]InputField `json:"variables,omitempty"`
}

// APIError represents an application/problem+json error response.
type APIError struct {
	Type     string        `json:"type"`
	Title    string        `json:"title"`
	Status   int           `json:"status"`
	Detail   string        `json:"detail"`
	Instance string        `json:"instance,omitempty"`
	Errors   []ErrorDetail `json:"errors,omitempty"`
}

// ErrorDetail describes a specific validation or field-level error.
type ErrorDetail struct {
	Location string `json:"location,omitempty"`
	Message  string `json:"message,omitempty"`
	Value    any    `json:"value,omitempty"`
}

// Error implements the error interface for APIError.
func (e *APIError) Error() string {
	if e.Detail != "" {
		return e.Detail
	}

	if e.Title != "" {
		return e.Title
	}

	return "registry API error"
}
