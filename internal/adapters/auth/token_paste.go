package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// TokenPasteProvider prompts the user to paste a Personal Access Token.
// Optionally validates the token by hitting AuthConfig.ValidateURL.
type TokenPasteProvider struct {
	transport ports.Transport
	catalog   ports.Catalog
}

// NewTokenPasteProvider creates a PAT provider (no validation).
func NewTokenPasteProvider() *TokenPasteProvider { return &TokenPasteProvider{} }

// WithValidation enables post-prompt validation via the service's validate_url.
func (p *TokenPasteProvider) WithValidation(transport ports.Transport, catalog ports.Catalog) *TokenPasteProvider {
	p.transport = transport
	p.catalog = catalog
	return p
}

func (p *TokenPasteProvider) Supports(kind core.ProviderKind) bool {
	return kind == core.ProviderPAT
}

func (p *TokenPasteProvider) Login(ctx context.Context, svc core.ServiceID, alias core.AccountAlias) (core.Credential, error) {
	tok, err := promptSecret("Paste your token for " + string(svc) + " (input hidden): ")
	if err != nil {
		return core.Credential{}, err
	}
	cred := core.Credential{
		Provider:    core.ProviderPAT,
		Service:     svc,
		Account:     alias,
		AccessToken: tok,
	}
	if err := validateToken(ctx, p.transport, p.catalog, svc, core.ProviderPAT, cred); err != nil {
		return core.Credential{}, err
	}
	return cred, nil
}

func (p *TokenPasteProvider) Refresh(_ context.Context, cred core.Credential) (core.Credential, error) {
	return cred, nil
}

// validateToken hits the service's validate_url with the credential injected
// per the service's Injection rule. 401/403 → reject; other statuses pass.
// No-op when transport/catalog are nil or no validate_url is configured.
func validateToken(ctx context.Context, tr ports.Transport, cat ports.Catalog, svc core.ServiceID, kind core.ProviderKind, cred core.Credential) error {
	if tr == nil || cat == nil {
		return nil
	}
	s, err := cat.GetService(ctx, svc)
	if err != nil {
		return nil
	}
	cfg, ok := s.AuthConfigs[kind]
	if !ok || cfg.ValidateURL == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.ValidateURL, nil)
	if err != nil {
		return err
	}
	if inj, ok := s.Injection[kind]; ok && inj.Header != "" {
		req.Header.Set(inj.Header, expandInjection(inj.Format, cred))
	}
	resp, err := tr.Do(req)
	if err != nil {
		return fmt.Errorf("token validation request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("token validation failed: %s returned %d", cfg.ValidateURL, resp.StatusCode)
	}
	return nil
}

// expandInjection substitutes {access_token}/{refresh_token} in the injection format.
func expandInjection(format string, cred core.Credential) string {
	if format == "" {
		return cred.AccessToken.Reveal()
	}
	out := strings.ReplaceAll(format, "{access_token}", cred.AccessToken.Reveal())
	out = strings.ReplaceAll(out, "{refresh_token}", cred.RefreshToken.Reveal())
	return out
}

var _ ports.AuthProvider = (*TokenPasteProvider)(nil)
