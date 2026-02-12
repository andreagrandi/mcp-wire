package service

// Service represents an MCP server definition loaded from a YAML file.
type Service struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Transport   string   `yaml:"transport"` // "sse" or "stdio"
	URL         string   `yaml:"url,omitempty"`
	Command     string   `yaml:"command,omitempty"`
	Args        []string `yaml:"args,omitempty"`
	Env         []EnvVar `yaml:"env,omitempty"`
}

// EnvVar describes an environment variable required by a service.
type EnvVar struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	SetupURL    string `yaml:"setup_url,omitempty"`
	SetupHint   string `yaml:"setup_hint,omitempty"`
}
