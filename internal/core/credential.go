package core

import "time"

// ProviderKind identifies the authentication mechanism used.
type ProviderKind string

const (
	ProviderOAuthUser   ProviderKind = "oauth2_user"
	ProviderOAuthDevice ProviderKind = "oauth2_device"
	ProviderOAuthClient ProviderKind = "oauth2_client"
	ProviderPAT         ProviderKind = "pat"
	ProviderAPIKey      ProviderKind = "api_key"
	ProviderAWSKeys     ProviderKind = "aws_keys"
)

// Credential stores the authentication state for one account.
type Credential struct {
	Provider     ProviderKind
	Service      ServiceID
	Account      AccountAlias
	AccessToken  Secret
	RefreshToken Secret
	ExpiresAt    *time.Time
	Scopes       []string
}

// NeedsRefresh reports whether the access token is expired at the given time.
// Returns false when no expiry is set (e.g. PAT, API key).
func (c Credential) NeedsRefresh(now time.Time) bool {
	if c.ExpiresAt == nil {
		return false
	}
	return now.After(*c.ExpiresAt)
}

// Ref returns the AccountRef for this credential.
func (c Credential) Ref() AccountRef {
	return AccountRef{Service: c.Service, Alias: c.Account}
}
