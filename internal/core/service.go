package core

// ServiceID uniquely identifies a service in the catalog (e.g. "github", "notion").
type ServiceID string

// AuthInjection tells the runtime how to inject a credential into HTTP requests
// for a given provider kind.
type AuthInjection struct {
	Header string // e.g. "Authorization"
	Format string // e.g. "Bearer {access_token}"
}

// AuthConfig holds per-service, per-provider authentication parameters
// (OAuth endpoints, client_id, validation URL, etc.).
// Sensitive material (client secret, tokens) never belongs here — store in vault.
type AuthConfig struct {
	ClientID       string
	AuthEndpoint   string   // OAuth authorization endpoint (user flow)
	TokenEndpoint  string   // OAuth token endpoint
	DeviceEndpoint string   // OAuth device authorization endpoint (RFC 8628)
	Scopes         []string // OAuth scopes requested at login
	RedirectPath   string   // local server callback path (default /callback)
	ValidateURL    string   // health URL hit after token paste / api key entry
}

// Service is the catalog entry for a third-party API.
type Service struct {
	ID          ServiceID
	Name        string
	Description string
	Version     string
	Sha256      string                         // tarball sha256 from HTTP index (empty for FS catalog)
	BaseURL     string                         // e.g. "https://api.github.com"
	Providers   []ProviderKind                 // declared auth providers
	Injection   map[ProviderKind]AuthInjection // per-provider injection rule
	AuthConfigs map[ProviderKind]AuthConfig    // per-provider configuration
	Actions     []Action
	// Source identifies where this service was loaded from. Empty for the
	// official built-in catalog; "tap:<name>" for a third-party tap. Action
	// Source fields are populated from this.
	Source string
}
