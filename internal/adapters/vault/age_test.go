package vault

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
	"elydelva/one/internal/testing/portstest"
)

func newAgeFactory(t *testing.T) func(t *testing.T) ports.Vault {
	return func(t *testing.T) ports.Vault {
		dir := t.TempDir()
		return NewAgeVault(filepath.Join(dir, "vault.age"), func() (string, error) {
			return "test-pass", nil
		})
	}
}

func TestAgeVault_Contract(t *testing.T) {
	portstest.RunVaultTests(t, "AgeVault", newAgeFactory(t))
}

func TestAgeVault_RoundtripPersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.age")
	pass := func() (string, error) { return "test-pass", nil }
	v1 := NewAgeVault(path, pass)
	exp := time.Now().Add(time.Hour)
	cred := core.Credential{
		Provider:     core.ProviderPAT,
		Service:      "github",
		Account:      "work",
		AccessToken:  core.NewSecret("atk"),
		RefreshToken: core.NewSecret("rtk"),
		ExpiresAt:    &exp,
	}
	if err := v1.Store(context.Background(), cred.Ref(), cred); err != nil {
		t.Fatal(err)
	}
	v2 := NewAgeVault(path, pass)
	got, err := v2.Fetch(context.Background(), cred.Ref())
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken.Reveal() != "atk" {
		t.Errorf("access_token = %q", got.AccessToken.Reveal())
	}
	if got.RefreshToken.Reveal() != "rtk" {
		t.Errorf("refresh_token = %q", got.RefreshToken.Reveal())
	}
}

func TestAgeVault_WrongPassphraseFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.age")
	good := NewAgeVault(path, func() (string, error) { return "good", nil })
	cred := core.Credential{Provider: core.ProviderPAT, Service: "x", Account: "y", AccessToken: core.NewSecret("t")}
	if err := good.Store(context.Background(), cred.Ref(), cred); err != nil {
		t.Fatal(err)
	}
	bad := NewAgeVault(path, func() (string, error) { return "nope", nil })
	if _, err := bad.Fetch(context.Background(), cred.Ref()); err == nil {
		t.Fatal("expected decryption error")
	}
}
