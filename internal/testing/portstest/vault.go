package portstest

import (
	"context"
	"testing"
	"time"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// RunVaultTests executes the ports.Vault contract against any implementation.
func RunVaultTests(t *testing.T, name string, factory func(t *testing.T) ports.Vault) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		ref := core.AccountRef{Service: "test", Alias: "default"}
		cred := core.Credential{
			Provider:    core.ProviderPAT,
			Service:     "test",
			Account:     "default",
			AccessToken: core.NewSecret("test-token"),
			ExpiresAt:   ptrTime(time.Now().Add(1 * time.Hour)),
		}

		t.Run("Fetch returns ErrNotAuthenticated for missing", func(t *testing.T) {
			v := factory(t)
			_, err := v.Fetch(context.Background(), ref)
			var notAuth core.ErrNotAuthenticated
			if !isErrorAs(err, &notAuth) {
				t.Errorf("expected ErrNotAuthenticated, got %T: %v", err, err)
			}
		})

		t.Run("Store then Fetch returns credential", func(t *testing.T) {
			v := factory(t)
			if err := v.Store(context.Background(), ref, cred); err != nil {
				t.Fatalf("store: %v", err)
			}
			got, err := v.Fetch(context.Background(), ref)
			if err != nil {
				t.Fatalf("fetch: %v", err)
			}
			if got.AccessToken.Reveal() != cred.AccessToken.Reveal() {
				t.Errorf("access token mismatch: got %q, want %q", got.AccessToken.Reveal(), cred.AccessToken.Reveal())
			}
		})

		t.Run("Delete removes credential", func(t *testing.T) {
			v := factory(t)
			_ = v.Store(context.Background(), ref, cred)
			if err := v.Delete(context.Background(), ref); err != nil {
				t.Fatalf("delete: %v", err)
			}
			_, err := v.Fetch(context.Background(), ref)
			var notAuth core.ErrNotAuthenticated
			if !isErrorAs(err, &notAuth) {
				t.Errorf("expected ErrNotAuthenticated after delete, got %T: %v", err, err)
			}
		})
	})
}

func ptrTime(t time.Time) *time.Time { return &t }
