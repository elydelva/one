package auth

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// OAuthClientProvider implements the OAuth 2.0 client credentials grant (machine-to-machine).
// The client secret is prompted no-echo at login time and stored as RefreshToken
// (re-used for subsequent refreshes — there is no refresh token in this flow).
type OAuthClientProvider struct {
	transport ports.Transport
	catalog   ports.Catalog
}

// NewOAuthClientProvider creates a client-credentials provider.
func NewOAuthClientProvider(transport ports.Transport, catalog ports.Catalog) *OAuthClientProvider {
	return &OAuthClientProvider{transport: transport, catalog: catalog}
}

func (p *OAuthClientProvider) Supports(kind core.ProviderKind) bool {
	return kind == core.ProviderOAuthClient
}

func (p *OAuthClientProvider) Login(ctx context.Context, svc core.ServiceID, alias core.AccountAlias) (core.Credential, error) {
	cfg, err := p.lookupConfig(ctx, svc)
	if err != nil {
		return core.Credential{}, err
	}
	if cfg.ClientID == "" || cfg.TokenEndpoint == "" {
		return core.Credential{}, fmt.Errorf("oauth2_client: missing client_id/token_endpoint for %s", svc)
	}
	secret, err := promptSecret("Paste client secret for " + string(svc) + " (input hidden): ")
	if err != nil {
		return core.Credential{}, err
	}
	tok, err := p.exchange(ctx, cfg, secret.Reveal())
	if err != nil {
		return core.Credential{}, err
	}
	cred := tok.toCredential(core.ProviderOAuthClient, svc, alias, time.Now())
	// Persist client secret in RefreshToken slot so Refresh can re-fetch.
	cred.RefreshToken = secret
	return cred, nil
}

func (p *OAuthClientProvider) Refresh(ctx context.Context, cred core.Credential) (core.Credential, error) {
	cfg, err := p.lookupConfig(ctx, cred.Service)
	if err != nil {
		return core.Credential{}, err
	}
	if cred.RefreshToken.Reveal() == "" {
		return core.Credential{}, core.ErrReAuthRequired{Service: cred.Service, Account: cred.Account}
	}
	tok, err := p.exchange(ctx, cfg, cred.RefreshToken.Reveal())
	if err != nil {
		return core.Credential{}, err
	}
	out := tok.toCredential(core.ProviderOAuthClient, cred.Service, cred.Account, time.Now())
	out.RefreshToken = cred.RefreshToken
	return out, nil
}

func (p *OAuthClientProvider) exchange(ctx context.Context, cfg core.AuthConfig, secret string) (tokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", secret)
	if len(cfg.Scopes) > 0 {
		form.Set("scope", strings.Join(cfg.Scopes, " "))
	}
	return postToken(ctx, p.transport, cfg.TokenEndpoint, form)
}

func (p *OAuthClientProvider) lookupConfig(ctx context.Context, svc core.ServiceID) (core.AuthConfig, error) {
	s, err := p.catalog.GetService(ctx, svc)
	if err != nil {
		return core.AuthConfig{}, err
	}
	cfg, ok := s.AuthConfigs[core.ProviderOAuthClient]
	if !ok {
		return core.AuthConfig{}, fmt.Errorf("oauth2_client: no config for %s", svc)
	}
	return cfg, nil
}

var _ ports.AuthProvider = (*OAuthClientProvider)(nil)
