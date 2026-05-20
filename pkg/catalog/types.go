package catalog

// ServiceDef mirrors the service.yaml format for parsing.
type ServiceDef struct {
	Version     int                  `yaml:"version"`
	ID          string               `yaml:"id"`
	Name        string               `yaml:"name"`
	Description string               `yaml:"description"`
	Auth        AuthDef              `yaml:"auth"`
	Actions     map[string]ActionDef `yaml:"actions"`
}

// AuthDef describes supported authentication providers.
type AuthDef struct {
	Providers []string `yaml:"providers"`
}

// ActionDef describes a single action in the service YAML.
type ActionDef struct {
	Description string      `yaml:"description"`
	Permission  string      `yaml:"permission"`
	Inputs      []InputDef  `yaml:"inputs"`
	Handler     *HandlerDef `yaml:"handler"`
}

// InputDef describes an action parameter.
type InputDef struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Required    bool   `yaml:"required"`
	Description string `yaml:"description"`
}

// HandlerDef points to a WASM handler binary.
type HandlerDef struct {
	File           string `yaml:"file"`
	Sha256         string `yaml:"sha256"`
	HostAPIVersion int    `yaml:"host_api_version"`
}
