package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"elydelva/one/internal/adapters/transport"
	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
	"elydelva/one/internal/testing/fake"
)

// stubCatalog is a minimal Catalog returning a single service.
type stubCatalog struct{ svc *core.Service }

func (s *stubCatalog) GetService(_ context.Context, id core.ServiceID) (*core.Service, error) {
	if s.svc != nil && s.svc.ID == id {
		cp := *s.svc
		return &cp, nil
	}
	return nil, core.ErrUnknownService{Service: id}
}
func (s *stubCatalog) GetAction(_ context.Context, svc core.ServiceID, a core.ActionID) (*core.Action, error) {
	return nil, core.ErrUnknownAction{Service: svc, Action: a}
}
func (s *stubCatalog) ListServices(_ context.Context) ([]core.Service, error) { return nil, nil }
func (s *stubCatalog) GetSkill(_ context.Context, _ core.ServiceID) (string, error) {
	return "", nil
}
func (s *stubCatalog) GetGuide(_ context.Context, _ core.ServiceID, _ string) (*core.InstallGuide, error) {
	return nil, nil
}
func (s *stubCatalog) ListGuides(_ context.Context, _ core.ServiceID) ([]core.InstallGuide, error) {
	return nil, nil
}

func hostOf(rawURL string) string {
	s := strings.TrimPrefix(rawURL, "http://")
	if i := strings.Index(s, ":"); i >= 0 {
		return s[:i]
	}
	return s
}

func loopbackTransport(t *testing.T, srv *httptest.Server) ports.Transport {
	t.Helper()
	return transport.NewNetHTTP(transport.WithAllowHTTP(true), transport.WithAllowedHosts(hostOf(srv.URL)))
}

func TestOAuthClientProvider_LoginExchangesCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			http.NotFound(w, r)
			return
		}
		_ = r.ParseForm()
		if r.Form.Get("grant_type") != "client_credentials" {
			t.Errorf("grant_type = %q", r.Form.Get("grant_type"))
		}
		if r.Form.Get("client_id") != "client-x" {
			t.Errorf("client_id = %q", r.Form.Get("client_id"))
		}
		if r.Form.Get("client_secret") != "shh" {
			t.Errorf("client_secret = %q", r.Form.Get("client_secret"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "atk",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	cat := &stubCatalog{svc: &core.Service{
		ID: "myapi",
		AuthConfigs: map[core.ProviderKind]core.AuthConfig{
			core.ProviderOAuthClient: {
				ClientID:      "client-x",
				TokenEndpoint: srv.URL + "/token",
			},
		},
	}}
	tr := loopbackTransport(t, srv)
	p := NewOAuthClientProvider(tr, cat)
	stubPrompt(t, "shh")
	cred, err := p.Login(context.Background(), "myapi", "default")
	if err != nil {
		t.Fatal(err)
	}
	if cred.AccessToken.Reveal() != "atk" {
		t.Errorf("access_token = %q", cred.AccessToken.Reveal())
	}
	if cred.ExpiresAt == nil {
		t.Error("expected ExpiresAt populated")
	}
}

func TestOAuthDeviceProvider_HappyPath(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/device":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"device_code":      "DEV-1",
				"user_code":        "ABCD-EFGH",
				"verification_uri": "https://example.com/device",
				"interval":         1,
				"expires_in":       60,
			})
		case "/token":
			calls++
			if calls < 2 {
				w.WriteHeader(400)
				_ = json.NewEncoder(w).Encode(map[string]any{"error": "authorization_pending"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "device-atk",
				"expires_in":   3600,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	cat := &stubCatalog{svc: &core.Service{
		ID: "gh",
		AuthConfigs: map[core.ProviderKind]core.AuthConfig{
			core.ProviderOAuthDevice: {
				ClientID:       "cli",
				DeviceEndpoint: srv.URL + "/device",
				TokenEndpoint:  srv.URL + "/token",
			},
		},
	}}
	p := NewOAuthDeviceProvider(loopbackTransport(t, srv), cat, fake.NewClock(time.Now()))
	cred, err := p.Login(context.Background(), "gh", "work")
	if err != nil {
		t.Fatal(err)
	}
	if cred.AccessToken.Reveal() != "device-atk" {
		t.Errorf("access_token = %q", cred.AccessToken.Reveal())
	}
	if calls != 2 {
		t.Errorf("expected 2 polls, got %d", calls)
	}
}

func TestTokenPasteProvider_ValidationRejects401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Error("expected Authorization header")
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	cat := &stubCatalog{svc: &core.Service{
		ID: "demo",
		Injection: map[core.ProviderKind]core.AuthInjection{
			core.ProviderPAT: {Header: "Authorization", Format: "Bearer {access_token}"},
		},
		AuthConfigs: map[core.ProviderKind]core.AuthConfig{
			core.ProviderPAT: {ValidateURL: srv.URL + "/me"},
		},
	}}
	stubPrompt(t, "bad-token")
	p := NewTokenPasteProvider().WithValidation(loopbackTransport(t, srv), cat)
	_, err := p.Login(context.Background(), "demo", "default")
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestTokenPasteProvider_ValidationAccepts200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()
	cat := &stubCatalog{svc: &core.Service{
		ID: "demo",
		Injection: map[core.ProviderKind]core.AuthInjection{
			core.ProviderPAT: {Header: "Authorization", Format: "Bearer {access_token}"},
		},
		AuthConfigs: map[core.ProviderKind]core.AuthConfig{
			core.ProviderPAT: {ValidateURL: srv.URL + "/me"},
		},
	}}
	stubPrompt(t, "good-token")
	p := NewTokenPasteProvider().WithValidation(loopbackTransport(t, srv), cat)
	cred, err := p.Login(context.Background(), "demo", "default")
	if err != nil {
		t.Fatal(err)
	}
	if cred.AccessToken.Reveal() != "good-token" {
		t.Errorf("token = %q", cred.AccessToken.Reveal())
	}
}
