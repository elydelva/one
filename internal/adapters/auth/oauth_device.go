package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// OAuthDeviceProvider implements the OAuth 2.0 Device Authorization Grant (RFC 8628).
type OAuthDeviceProvider struct {
	transport ports.Transport
	catalog   ports.Catalog
	clock     ports.Clock
}

// NewOAuthDeviceProvider creates a device-code provider.
func NewOAuthDeviceProvider(transport ports.Transport, catalog ports.Catalog, clock ports.Clock) *OAuthDeviceProvider {
	return &OAuthDeviceProvider{transport: transport, catalog: catalog, clock: clock}
}

func (p *OAuthDeviceProvider) Supports(kind core.ProviderKind) bool {
	return kind == core.ProviderOAuthDevice
}

type deviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

func (p *OAuthDeviceProvider) Login(ctx context.Context, svc core.ServiceID, alias core.AccountAlias) (core.Credential, error) {
	cfg, err := p.lookupConfig(ctx, svc)
	if err != nil {
		return core.Credential{}, err
	}
	if cfg.ClientID == "" || cfg.DeviceEndpoint == "" || cfg.TokenEndpoint == "" {
		return core.Credential{}, fmt.Errorf("oauth2_device: missing client_id/device_endpoint/token_endpoint for %s", svc)
	}
	dev, err := p.requestDeviceCode(ctx, cfg)
	if err != nil {
		return core.Credential{}, err
	}
	if dev.Interval <= 0 {
		dev.Interval = 5
	}
	fmt.Fprintf(os.Stderr, "Visit %s and enter code: %s\n", dev.VerificationURI, dev.UserCode)
	if dev.VerificationURIComplete != "" {
		_ = openBrowser(dev.VerificationURIComplete)
	}

	deadline := p.clock.Now().Add(time.Duration(dev.ExpiresIn) * time.Second)
	if dev.ExpiresIn <= 0 {
		deadline = p.clock.Now().Add(15 * time.Minute)
	}
	interval := time.Duration(dev.Interval) * time.Second

	for {
		if !p.clock.Now().Before(deadline) {
			return core.Credential{}, errors.New("oauth2_device: code expired before user completed login")
		}
		select {
		case <-ctx.Done():
			return core.Credential{}, ctx.Err()
		case <-time.After(interval):
		}
		form := url.Values{}
		form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
		form.Set("device_code", dev.DeviceCode)
		form.Set("client_id", cfg.ClientID)
		tok, errCode, err := pollDeviceToken(ctx, p.transport, cfg.TokenEndpoint, form)
		if err != nil {
			return core.Credential{}, err
		}
		switch errCode {
		case "":
			return tok.toCredential(core.ProviderOAuthDevice, svc, alias, p.clock.Now()), nil
		case "authorization_pending":
			continue
		case "slow_down":
			interval += 5 * time.Second
		case "expired_token":
			return core.Credential{}, errors.New("oauth2_device: device code expired")
		case "access_denied":
			return core.Credential{}, errors.New("oauth2_device: access denied by user")
		default:
			return core.Credential{}, fmt.Errorf("oauth2_device: %s", errCode)
		}
	}
}

func (p *OAuthDeviceProvider) Refresh(ctx context.Context, cred core.Credential) (core.Credential, error) {
	cfg, err := p.lookupConfig(ctx, cred.Service)
	if err != nil {
		return core.Credential{}, err
	}
	if cred.RefreshToken.Reveal() == "" {
		return core.Credential{}, core.ErrReAuthRequired{Service: cred.Service, Account: cred.Account}
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", cred.RefreshToken.Reveal())
	form.Set("client_id", cfg.ClientID)
	tok, err := postToken(ctx, p.transport, cfg.TokenEndpoint, form)
	if err != nil {
		return core.Credential{}, err
	}
	out := tok.toCredential(core.ProviderOAuthDevice, cred.Service, cred.Account, p.clock.Now())
	if out.RefreshToken.Reveal() == "" {
		out.RefreshToken = cred.RefreshToken
	}
	return out, nil
}

func (p *OAuthDeviceProvider) lookupConfig(ctx context.Context, svc core.ServiceID) (core.AuthConfig, error) {
	s, err := p.catalog.GetService(ctx, svc)
	if err != nil {
		return core.AuthConfig{}, err
	}
	cfg, ok := s.AuthConfigs[core.ProviderOAuthDevice]
	if !ok {
		return core.AuthConfig{}, fmt.Errorf("oauth2_device: no config for %s", svc)
	}
	return cfg, nil
}

func (p *OAuthDeviceProvider) requestDeviceCode(ctx context.Context, cfg core.AuthConfig) (deviceAuthResponse, error) {
	form := url.Values{}
	form.Set("client_id", cfg.ClientID)
	if len(cfg.Scopes) > 0 {
		form.Set("scope", strings.Join(cfg.Scopes, " "))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.DeviceEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return deviceAuthResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := p.transport.Do(req)
	if err != nil {
		return deviceAuthResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return deviceAuthResponse{}, err
	}
	if resp.StatusCode >= 300 {
		return deviceAuthResponse{}, fmt.Errorf("oauth2_device: device endpoint returned %d: %s", resp.StatusCode, string(body))
	}
	var d deviceAuthResponse
	if err := json.Unmarshal(body, &d); err != nil {
		return deviceAuthResponse{}, err
	}
	return d, nil
}

// pollDeviceToken posts the device-token request and returns either a token,
// a transient OAuth error code (authorization_pending/slow_down/...), or a hard error.
func pollDeviceToken(ctx context.Context, tr ports.Transport, endpoint string, form url.Values) (tokenResponse, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := tr.Do(req)
	if err != nil {
		return tokenResponse{}, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return tokenResponse{}, "", err
	}
	var t tokenResponse
	if err := json.Unmarshal(body, &t); err != nil {
		return tokenResponse{}, "", fmt.Errorf("decode (status %d): %w", resp.StatusCode, err)
	}
	if t.Error != "" {
		return tokenResponse{}, t.Error, nil
	}
	if resp.StatusCode >= 300 {
		return tokenResponse{}, "", fmt.Errorf("oauth2_device: token endpoint returned %d", resp.StatusCode)
	}
	if t.AccessToken == "" {
		return tokenResponse{}, "", errors.New("oauth2_device: empty access_token")
	}
	return t, "", nil
}

var _ ports.AuthProvider = (*OAuthDeviceProvider)(nil)
