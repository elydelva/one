// Package catalog defines the YAML schema for a service definition in the One
// CLI catalog. It is the unmarshaling target for `service.yaml` and the
// per-action `actions/<id>.yaml` files.
package catalog

// ServiceDef mirrors service.yaml.
type ServiceDef struct {
	Version     int                  `yaml:"version" json:"version"`
	ID          string               `yaml:"id" json:"id"`
	Name        string               `yaml:"name" json:"name"`
	Description string               `yaml:"description,omitempty" json:"description,omitempty"`
	BaseURL     string               `yaml:"base_url" json:"base_url"`
	Auth        AuthDef              `yaml:"auth,omitempty" json:"auth,omitempty"`
	// Actions can either be inlined here OR placed as separate files under actions/<id>.yaml.
	Actions     map[string]ActionDef `yaml:"actions,omitempty" json:"actions,omitempty"`
}

// AuthDef describes supported authentication providers and how to inject them.
type AuthDef struct {
	Providers []string               `yaml:"providers,omitempty" json:"providers,omitempty"`
	Injection map[string]InjectionDef `yaml:"injection,omitempty" json:"injection,omitempty"`
}

// InjectionDef tells the runtime how to inject a credential into HTTP requests.
type InjectionDef struct {
	// Header is the HTTP header name (e.g. "Authorization").
	Header string `yaml:"header" json:"header"`
	// Format is a template like "Bearer {access_token}".
	Format string `yaml:"format" json:"format"`
}

// ActionDef describes a single action in YAML.
type ActionDef struct {
	ID          string        `yaml:"id,omitempty" json:"id,omitempty"` // optional; falls back to filename or map key
	Description string        `yaml:"description" json:"description"`
	Permission  string        `yaml:"permission" json:"permission"`
	Request     *RequestDef   `yaml:"request,omitempty" json:"request,omitempty"`
	Inputs      []InputDef    `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Pagination  *PaginationDef `yaml:"pagination,omitempty" json:"pagination,omitempty"`
	Errors      map[string]ErrorDef `yaml:"errors,omitempty" json:"errors,omitempty"`
	Handler     *HandlerDef   `yaml:"handler,omitempty" json:"handler,omitempty"`
}

// RequestDef declares the HTTP request shape for declarative actions.
type RequestDef struct {
	Method  string            `yaml:"method" json:"method"` // GET, POST, PUT, PATCH, DELETE
	Path    string            `yaml:"path" json:"path"`     // supports {var} interpolation
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	// Body can be:
	//   - "$inputs"            → marshal all body-located inputs as JSON
	//   - "{template}"         → interpolated string
	//   - empty                → no body
	Body string `yaml:"body,omitempty" json:"body,omitempty"`
}

// InputDef describes an action parameter (mirrors core.InputDef).
type InputDef struct {
	Name        string   `yaml:"name" json:"name"`
	Type        string   `yaml:"type" json:"type"`
	Required    bool     `yaml:"required,omitempty" json:"required,omitempty"`
	Location    string   `yaml:"location,omitempty" json:"location,omitempty"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Pattern     string   `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	Enum        []any    `yaml:"enum,omitempty" json:"enum,omitempty"`
	Default     any      `yaml:"default,omitempty" json:"default,omitempty"`
	Min         *float64 `yaml:"min,omitempty" json:"min,omitempty"`
	Max         *float64 `yaml:"max,omitempty" json:"max,omitempty"`
	MinLen      *int     `yaml:"min_length,omitempty" json:"min_length,omitempty"`
	MaxLen      *int     `yaml:"max_length,omitempty" json:"max_length,omitempty"`
	Items       *InputDef `yaml:"items,omitempty" json:"items,omitempty"`
}

// PaginationDef declares cursor-based pagination for an action.
type PaginationDef struct {
	Style          string `yaml:"style" json:"style"` // "cursor" only in v0.2
	RequestParam   string `yaml:"request_param" json:"request_param"`
	RequestLocation string `yaml:"request_location,omitempty" json:"request_location,omitempty"` // default: query
	ResponseToken  string `yaml:"response_token" json:"response_token"`
	ResponseItems  string `yaml:"response_items,omitempty" json:"response_items,omitempty"` // JSON path to items array; if empty assume top-level
	MaxPages       int    `yaml:"max_pages" json:"max_pages"`
}

// ErrorDef declares a custom mapping for an HTTP status code.
type ErrorDef struct {
	Code         string `yaml:"code" json:"code"`
	Hint         string `yaml:"hint,omitempty" json:"hint,omitempty"`
	InstallGuide string `yaml:"install_guide,omitempty" json:"install_guide,omitempty"`
	Retry        string `yaml:"retry,omitempty" json:"retry,omitempty"`
	MaxAttempts  int    `yaml:"max_attempts,omitempty" json:"max_attempts,omitempty"`
}

// HandlerDef points to a WASM handler binary (Phase 3+).
type HandlerDef struct {
	File           string `yaml:"file" json:"file"`
	Sha256         string `yaml:"sha256" json:"sha256"`
	HostAPIVersion int    `yaml:"host_api_version" json:"host_api_version"`
}
