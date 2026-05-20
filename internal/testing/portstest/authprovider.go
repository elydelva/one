package portstest

import (
	"context"
	"testing"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// RunAuthProviderTests verifies invariants every AuthProvider must satisfy.
// The factory must return a provider configured for the given supported kind.
func RunAuthProviderTests(t *testing.T, name string, kind core.ProviderKind, factory func(t *testing.T) ports.AuthProvider) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Run("Supports matches declared kind", func(t *testing.T) {
			p := factory(t)
			if !p.Supports(kind) {
				t.Errorf("expected Supports(%q) = true", kind)
			}
			for _, k := range otherKinds(kind) {
				if p.Supports(k) {
					t.Errorf("expected Supports(%q) = false", k)
				}
			}
		})

		t.Run("Refresh is idempotent for non-expiring credentials", func(t *testing.T) {
			p := factory(t)
			cred := core.Credential{
				Provider:    kind,
				Service:     "test",
				Account:     "default",
				AccessToken: core.NewSecret("token-xyz"),
			}
			got, err := p.Refresh(context.Background(), cred)
			if err != nil {
				t.Fatalf("refresh: %v", err)
			}
			if got.AccessToken.Reveal() != cred.AccessToken.Reveal() {
				t.Errorf("refresh of non-expiring credential changed token")
			}
		})
	})
}

func otherKinds(k core.ProviderKind) []core.ProviderKind {
	all := []core.ProviderKind{
		core.ProviderOAuthUser,
		core.ProviderOAuthDevice,
		core.ProviderOAuthClient,
		core.ProviderPAT,
		core.ProviderAPIKey,
		core.ProviderAWSKeys,
	}
	out := make([]core.ProviderKind, 0, len(all)-1)
	for _, x := range all {
		if x != k {
			out = append(out, x)
		}
	}
	return out
}
