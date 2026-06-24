package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// OAuthUserProvider implements the OAuth 2.0 authorization code flow with PKCE (S256).
// A local HTTP server listens on a free port for the callback, receives the code,
// then exchanges it for tokens at the token endpoint.
type OAuthUserProvider struct {
	transport    ports.Transport
	catalog      ports.Catalog
	loginTimeout time.Duration
}

// NewOAuthUserProvider creates a provider that resolves per-service config from catalog.
func NewOAuthUserProvider(transport ports.Transport, catalog ports.Catalog) *OAuthUserProvider {
	return &OAuthUserProvider{transport: transport, catalog: catalog, loginTimeout: 5 * time.Minute}
}

func (p *OAuthUserProvider) Supports(kind core.ProviderKind) bool {
	return kind == core.ProviderOAuthUser
}

func (p *OAuthUserProvider) Login(ctx context.Context, svc core.ServiceID, alias core.AccountAlias) (core.Credential, error) {
	cfg, err := p.lookupConfig(ctx, svc)
	if err != nil {
		return core.Credential{}, err
	}
	if cfg.ClientID == "" || cfg.AuthEndpoint == "" || cfg.TokenEndpoint == "" {
		return core.Credential{}, fmt.Errorf("oauth2_user: missing client_id/auth_endpoint/token_endpoint in catalog for %s", svc)
	}

	verifier, err := randomURLBase64(48)
	if err != nil {
		return core.Credential{}, err
	}
	challengeSum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(challengeSum[:])
	state, err := randomURLBase64(24)
	if err != nil {
		return core.Credential{}, err
	}

	redirectPath := cfg.RedirectPath
	if redirectPath == "" {
		redirectPath = "/callback"
	}
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return core.Credential{}, err
	}
	defer func() { _ = listener.Close() }()
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d%s", port, redirectPath)

	authURL := buildAuthURL(cfg, redirectURI, state, challenge)
	_ = openBrowser(authURL)
	fmt.Fprintf(os.Stderr, "Open this URL to continue:\n  %s\n", authURL)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	srv := &http.Server{ReadHeaderTimeout: 5 * time.Second}
	mux := http.NewServeMux()
	mux.HandleFunc(redirectPath, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("state"); got != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			errCh <- errors.New("oauth2_user: state mismatch")
			return
		}
		if e := q.Get("error"); e != "" {
			http.Error(w, e, http.StatusBadRequest)
			errCh <- fmt.Errorf("oauth2_user: %s — %s", e, q.Get("error_description"))
			return
		}
		code := q.Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			errCh <- errors.New("oauth2_user: missing code")
			return
		}
		_, _ = io.WriteString(w, "Login complete. You can close this window.")
		codeCh <- code
	})
	srv.Handler = mux
	go func() { _ = srv.Serve(listener) }()
	defer func() { _ = srv.Close() }()

	loginCtx, cancel := context.WithTimeout(ctx, p.loginTimeout)
	defer cancel()
	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return core.Credential{}, err
	case <-loginCtx.Done():
		return core.Credential{}, fmt.Errorf("oauth2_user: timeout waiting for callback after %s", p.loginTimeout)
	}

	tok, err := p.exchangeCode(ctx, cfg, code, verifier, redirectURI)
	if err != nil {
		return core.Credential{}, err
	}
	return tok.toCredential(core.ProviderOAuthUser, svc, alias, time.Now()), nil
}

func (p *OAuthUserProvider) Refresh(ctx context.Context, cred core.Credential) (core.Credential, error) {
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
	out := tok.toCredential(core.ProviderOAuthUser, cred.Service, cred.Account, time.Now())
	if out.RefreshToken.Reveal() == "" {
		out.RefreshToken = cred.RefreshToken
	}
	return out, nil
}

func (p *OAuthUserProvider) exchangeCode(ctx context.Context, cfg core.AuthConfig, code, verifier, redirectURI string) (tokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", cfg.ClientID)
	form.Set("code_verifier", verifier)
	return postToken(ctx, p.transport, cfg.TokenEndpoint, form)
}

func (p *OAuthUserProvider) lookupConfig(ctx context.Context, svc core.ServiceID) (core.AuthConfig, error) {
	s, err := p.catalog.GetService(ctx, svc)
	if err != nil {
		return core.AuthConfig{}, err
	}
	cfg, ok := s.AuthConfigs[core.ProviderOAuthUser]
	if !ok {
		return core.AuthConfig{}, fmt.Errorf("oauth2_user: no config for service %s", svc)
	}
	return cfg, nil
}

func buildAuthURL(cfg core.AuthConfig, redirectURI, state, challenge string) string {
	u, _ := url.Parse(cfg.AuthEndpoint)
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", cfg.ClientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	if len(cfg.Scopes) > 0 {
		q.Set("scope", strings.Join(cfg.Scopes, " "))
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func randomURLBase64(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// tokenResponse mirrors the OAuth 2.0 token response.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	Error        string `json:"error,omitempty"`
	ErrorDesc    string `json:"error_description,omitempty"`
}

func (t tokenResponse) toCredential(kind core.ProviderKind, svc core.ServiceID, alias core.AccountAlias, now time.Time) core.Credential {
	cred := core.Credential{
		Provider:     kind,
		Service:      svc,
		Account:      alias,
		AccessToken:  core.NewSecret(t.AccessToken),
		RefreshToken: core.NewSecret(t.RefreshToken),
	}
	if t.ExpiresIn > 0 {
		exp := now.Add(time.Duration(t.ExpiresIn) * time.Second)
		cred.ExpiresAt = &exp
	}
	if t.Scope != "" {
		cred.Scopes = strings.Fields(t.Scope)
	}
	return cred
}

// postToken sends a form-encoded POST and decodes the JSON response.
func postToken(ctx context.Context, transport ports.Transport, endpoint string, form url.Values) (tokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := transport.Do(req)
	if err != nil {
		return tokenResponse{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return tokenResponse{}, err
	}
	var t tokenResponse
	if err := json.Unmarshal(body, &t); err != nil {
		return tokenResponse{}, fmt.Errorf("decode token response (status %d): %w", resp.StatusCode, err)
	}
	if t.Error != "" {
		return tokenResponse{}, fmt.Errorf("oauth: %s — %s", t.Error, t.ErrorDesc)
	}
	if resp.StatusCode >= 300 {
		return tokenResponse{}, fmt.Errorf("oauth: token endpoint returned %d", resp.StatusCode)
	}
	if t.AccessToken == "" {
		return tokenResponse{}, errors.New("oauth: token response missing access_token")
	}
	return t, nil
}

var _ ports.AuthProvider = (*OAuthUserProvider)(nil)
