package core

// ServiceID uniquely identifies a service in the catalog (e.g. "github", "notion").
type ServiceID string

// AuthInjection tells the runtime how to inject a credential into HTTP requests
// for a given provider kind.
type AuthInjection struct {
	Header string // e.g. "Authorization"
	Format string // e.g. "Bearer {access_token}"
}

// Service is the catalog entry for a third-party API.
type Service struct {
	ID          ServiceID
	Name        string
	Description string
	Version     string
	BaseURL     string                          // e.g. "https://api.github.com"
	Providers   []ProviderKind                  // declared auth providers
	Injection   map[ProviderKind]AuthInjection  // per-provider injection rule
	Actions     []Action
}
