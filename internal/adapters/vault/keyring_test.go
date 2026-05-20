package vault

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"elydelva/one/internal/core"
)

// newMockKeyringVault returns a KeyringVault backed by an in-memory map.
func newMockKeyringVault(t *testing.T) *KeyringVault {
	t.Helper()
	store := map[string]string{}
	key := func(svc, user string) string { return svc + "\x00" + user }
	return &KeyringVault{
		service: "one-test",
		set: func(svc, user, pw string) error {
			store[key(svc, user)] = pw
			return nil
		},
		get: func(svc, user string) (string, error) {
			v, ok := store[key(svc, user)]
			if !ok {
				return "", keyring.ErrNotFound
			}
			return v, nil
		},
		delete: func(svc, user string) error {
			k := key(svc, user)
			if _, ok := store[k]; !ok {
				return keyring.ErrNotFound
			}
			delete(store, k)
			return nil
		},
	}
}

func TestKeyringVault_StoreFetch(t *testing.T) {
	v := newMockKeyringVault(t)
	ref := core.AccountRef{Service: "github", Alias: "work"}
	cred := core.Credential{
		Provider:    core.ProviderPAT,
		Service:     "github",
		Account:     "work",
		AccessToken: core.NewSecret("ghp_abc"),
	}
	if err := v.Store(context.Background(), ref, cred); err != nil {
		t.Fatalf("store: %v", err)
	}
	got, err := v.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if got.AccessToken.Reveal() != "ghp_abc" {
		t.Errorf("token = %q", got.AccessToken.Reveal())
	}
}

func TestKeyringVault_FetchMissing(t *testing.T) {
	v := newMockKeyringVault(t)
	_, err := v.Fetch(context.Background(), core.AccountRef{Service: "x", Alias: "y"})
	var notAuth core.ErrNotAuthenticated
	if !errors.As(err, &notAuth) {
		t.Errorf("expected ErrNotAuthenticated, got %T: %v", err, err)
	}
}

func TestKeyringVault_Delete(t *testing.T) {
	v := newMockKeyringVault(t)
	ref := core.AccountRef{Service: "github", Alias: "work"}
	_ = v.Store(context.Background(), ref, core.Credential{Service: "github", Account: "work", AccessToken: core.NewSecret("t")})
	if err := v.Delete(context.Background(), ref); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err := v.Fetch(context.Background(), ref)
	var notAuth core.ErrNotAuthenticated
	if !errors.As(err, &notAuth) {
		t.Errorf("expected ErrNotAuthenticated after delete, got %T: %v", err, err)
	}
}

func TestKeyringVault_ListNotSupported(t *testing.T) {
	v := newMockKeyringVault(t)
	_, err := v.List(context.Background(), "github")
	var ns core.ErrNotSupported
	if !errors.As(err, &ns) {
		t.Errorf("expected ErrNotSupported, got %T: %v", err, err)
	}
}
