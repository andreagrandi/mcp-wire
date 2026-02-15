package service

// Service represents an MCP server definition loaded from a YAML file.
type Service struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Transport   string            `yaml:"transport"` // "http", "sse", or "stdio"
	Auth        string            `yaml:"auth,omitempty"`
	URL         string            `yaml:"url,omitempty"`
	Command     string            `yaml:"command,omitempty"`
	Args        []string          `yaml:"args,omitempty"`
	Env         []EnvVar          `yaml:"env,omitempty"`
	Headers     map[string]string `yaml:"-"`
}

// EnvVar describes an environment variable required by a service.
type EnvVar struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Default     string `yaml:"default,omitempty"`
	SetupURL    string `yaml:"setup_url,omitempty"`
	SetupHint   string `yaml:"setup_hint,omitempty"`
}
