package app

import (
	"context"
	"errors"
	"testing"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
	"elydelva/one/internal/testing/fake"
)

func TestLogin_StoresCredential(t *testing.T) {
	vault := fake.NewVault()
	provider := fake.NewAuthProvider(core.ProviderPAT)
	provider.LoginFunc = func(_ context.Context, svc core.ServiceID, alias core.AccountAlias) (core.Credential, error) {
		return core.Credential{
			Provider:    core.ProviderPAT,
			Service:     svc,
			Account:     alias,
			AccessToken: core.NewSecret("tok"),
		}, nil
	}
	uc := NewLogin(vault, nil, []ports.AuthProvider{provider}, fake.NewLogger())

	if err := uc.Run(context.Background(), LoginInput{Service: "github", Account: "work"}); err != nil {
		t.Fatalf("login: %v", err)
	}

	got, err := vault.Fetch(context.Background(), core.AccountRef{Service: "github", Alias: "work"})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if got.AccessToken.Reveal() != "tok" {
		t.Errorf("token = %q", got.AccessToken.Reveal())
	}
}

func TestLogin_DefaultsAccountToDefault(t *testing.T) {
	vault := fake.NewVault()
	provider := fake.NewAuthProvider(core.ProviderPAT)
	provider.LoginFunc = func(_ context.Context, svc core.ServiceID, alias core.AccountAlias) (core.Credential, error) {
		if alias != "default" {
			t.Errorf("expected alias=default, got %q", alias)
		}
		return core.Credential{Provider: core.ProviderPAT, Service: svc, Account: alias, AccessToken: core.NewSecret("t")}, nil
	}
	uc := NewLogin(vault, nil, []ports.AuthProvider{provider}, fake.NewLogger())
	if err := uc.Run(context.Background(), LoginInput{Service: "github"}); err != nil {
		t.Fatalf("login: %v", err)
	}
}

func TestLogin_NoProvider(t *testing.T) {
	uc := NewLogin(fake.NewVault(), nil, nil, fake.NewLogger())
	err := uc.Run(context.Background(), LoginInput{Service: "github"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLogin_PropagatesProviderError(t *testing.T) {
	provider := fake.NewAuthProvider(core.ProviderPAT)
	want := errors.New("boom")
	provider.LoginFunc = func(_ context.Context, _ core.ServiceID, _ core.AccountAlias) (core.Credential, error) {
		return core.Credential{}, want
	}
	uc := NewLogin(fake.NewVault(), nil, []ports.AuthProvider{provider}, fake.NewLogger())
	err := uc.Run(context.Background(), LoginInput{Service: "github"})
	if !errors.Is(err, want) {
		t.Errorf("expected %v, got %v", want, err)
	}
}
