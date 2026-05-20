package vault

import (
	"context"
	"errors"
	"testing"

	"elydelva/one/internal/core"
	"elydelva/one/internal/testing/fake"
)

func TestChainVault_FetchFallthrough(t *testing.T) {
	first := fake.NewVault()
	second := fake.NewVault()
	ref := core.AccountRef{Service: "github", Alias: "default"}
	_ = second.Store(context.Background(), ref, core.Credential{
		Provider: core.ProviderPAT, Service: "github", Account: "default", AccessToken: core.NewSecret("from-second"),
	})

	chain := NewChainVault(first, second)
	got, err := chain.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if got.AccessToken.Reveal() != "from-second" {
		t.Errorf("expected fallthrough to second, got %q", got.AccessToken.Reveal())
	}
}

func TestChainVault_FetchAllMiss(t *testing.T) {
	chain := NewChainVault(fake.NewVault(), fake.NewVault())
	_, err := chain.Fetch(context.Background(), core.AccountRef{Service: "x", Alias: "y"})
	var notAuth core.ErrNotAuthenticated
	if !errors.As(err, &notAuth) {
		t.Errorf("expected ErrNotAuthenticated, got %T: %v", err, err)
	}
}

func TestChainVault_StoreSkipsReadOnly(t *testing.T) {
	env := NewEnvVarVault() // read-only
	mem := fake.NewVault()
	chain := NewChainVault(env, mem)

	ref := core.AccountRef{Service: "github", Alias: "default"}
	cred := core.Credential{Service: "github", Account: "default", AccessToken: core.NewSecret("t")}
	if err := chain.Store(context.Background(), ref, cred); err != nil {
		t.Fatalf("store: %v", err)
	}
	got, err := mem.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("fetch from mem: %v", err)
	}
	if got.AccessToken.Reveal() != "t" {
		t.Errorf("token = %q", got.AccessToken.Reveal())
	}
}

func TestChainVault_DeleteIgnoresReadOnly(t *testing.T) {
	env := NewEnvVarVault()
	mem := fake.NewVault()
	ref := core.AccountRef{Service: "github", Alias: "default"}
	_ = mem.Store(context.Background(), ref, core.Credential{Service: "github", Account: "default"})

	chain := NewChainVault(env, mem)
	if err := chain.Delete(context.Background(), ref); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err := mem.Fetch(context.Background(), ref)
	var notAuth core.ErrNotAuthenticated
	if !errors.As(err, &notAuth) {
		t.Errorf("expected mem to be empty after chain delete")
	}
}

func TestChainVault_DeleteNotFound(t *testing.T) {
	chain := NewChainVault(fake.NewVault())
	err := chain.Delete(context.Background(), core.AccountRef{Service: "x", Alias: "y"})
	var notAuth core.ErrNotAuthenticated
	if !errors.As(err, &notAuth) {
		t.Errorf("expected ErrNotAuthenticated, got %T: %v", err, err)
	}
}

func TestChainVault_ListUnion(t *testing.T) {
	a := fake.NewVault()
	b := fake.NewVault()
	_ = a.Store(context.Background(), core.AccountRef{Service: "github", Alias: "default"}, core.Credential{Service: "github", Account: "default"})
	_ = b.Store(context.Background(), core.AccountRef{Service: "github", Alias: "work"}, core.Credential{Service: "github", Account: "work"})
	_ = b.Store(context.Background(), core.AccountRef{Service: "github", Alias: "default"}, core.Credential{Service: "github", Account: "default"}) // dup

	chain := NewChainVault(a, b)
	refs, err := chain.List(context.Background(), "github")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(refs) != 2 {
		t.Errorf("expected 2 deduped refs, got %d: %+v", len(refs), refs)
	}
}
