package auth

import (
	"context"
	"io"
	"testing"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
	"elydelva/one/internal/testing/portstest"
)

// stubPrompt swaps the prompt function for tests.
func stubPrompt(t *testing.T, secret string) {
	t.Helper()
	prev := promptFn
	promptFn = func(_ string, _ io.Reader, _ io.Writer) (core.Secret, error) {
		return core.NewSecret(secret), nil
	}
	t.Cleanup(func() { promptFn = prev })
}

func TestTokenPasteProvider_Contract(t *testing.T) {
	portstest.RunAuthProviderTests(t, "TokenPasteProvider", core.ProviderPAT, func(_ *testing.T) ports.AuthProvider {
		return NewTokenPasteProvider()
	})
}

func TestAPIKeyProvider_Contract(t *testing.T) {
	portstest.RunAuthProviderTests(t, "APIKeyProvider", core.ProviderAPIKey, func(_ *testing.T) ports.AuthProvider {
		return NewAPIKeyProvider()
	})
}

func TestTokenPasteProvider_Login(t *testing.T) {
	stubPrompt(t, "ghp_abc")
	cred, err := NewTokenPasteProvider().Login(context.Background(), "github", "work")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if cred.Provider != core.ProviderPAT {
		t.Errorf("provider = %v", cred.Provider)
	}
	if cred.Service != "github" || cred.Account != "work" {
		t.Errorf("ref = %s:%s", cred.Service, cred.Account)
	}
	if cred.AccessToken.Reveal() != "ghp_abc" {
		t.Errorf("token = %q", cred.AccessToken.Reveal())
	}
}

func TestAPIKeyProvider_Login(t *testing.T) {
	stubPrompt(t, "sk-xyz")
	cred, err := NewAPIKeyProvider().Login(context.Background(), "openai", "default")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if cred.Provider != core.ProviderAPIKey {
		t.Errorf("provider = %v", cred.Provider)
	}
	if cred.AccessToken.Reveal() != "sk-xyz" {
		t.Errorf("token = %q", cred.AccessToken.Reveal())
	}
}
