package core

import (
	"encoding/json"
	"time"
)

// ProviderKind identifies the authentication mechanism used.
type ProviderKind string

const (
	ProviderOAuthUser   ProviderKind = "oauth2_user"
	ProviderOAuthDevice ProviderKind = "oauth2_device"
	ProviderOAuthClient ProviderKind = "oauth2_client"
	ProviderPAT         ProviderKind = "pat"
	ProviderAPIKey      ProviderKind = "api_key"
	ProviderAWSKeys     ProviderKind = "aws_keys"
	ProviderCertificate ProviderKind = "certificate"
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

// credentialStorage is the on-disk / in-keychain shape of a credential.
// Secrets are stored in plaintext at this boundary; the vault layer is responsible
// for encryption (OS keychain, age, etc.).
type credentialStorage struct {
	Provider     ProviderKind `json:"provider"`
	Service      ServiceID    `json:"service"`
	Account      AccountAlias `json:"account"`
	AccessToken  string       `json:"access_token,omitempty"`
	RefreshToken string       `json:"refresh_token,omitempty"`
	ExpiresAt    *time.Time   `json:"expires_at,omitempty"`
	Scopes       []string     `json:"scopes,omitempty"`
}

// MarshalForStorage serializes the credential with secrets in plaintext.
// Only vault adapters should call this. Never log or transmit the result.
func (c Credential) MarshalForStorage() ([]byte, error) {
	return json.Marshal(credentialStorage{
		Provider:     c.Provider,
		Service:      c.Service,
		Account:      c.Account,
		AccessToken:  c.AccessToken.Reveal(),
		RefreshToken: c.RefreshToken.Reveal(),
		ExpiresAt:    c.ExpiresAt,
		Scopes:       c.Scopes,
	})
}

// UnmarshalCredentialFromStorage reverses MarshalForStorage.
func UnmarshalCredentialFromStorage(data []byte) (Credential, error) {
	var s credentialStorage
	if err := json.Unmarshal(data, &s); err != nil {
		return Credential{}, err
	}
	return Credential{
		Provider:     s.Provider,
		Service:      s.Service,
		Account:      s.Account,
		AccessToken:  NewSecret(s.AccessToken),
		RefreshToken: NewSecret(s.RefreshToken),
		ExpiresAt:    s.ExpiresAt,
		Scopes:       s.Scopes,
	}, nil
}
