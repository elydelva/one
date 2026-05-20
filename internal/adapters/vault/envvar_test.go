package vault

import (
	"context"
	"errors"
	"testing"

	"elydelva/one/internal/core"
)

func TestEnvVarVault_Fetch(t *testing.T) {
	env := map[string]string{
		"ONE_CREDS_GITHUB_DEFAULT": `{"provider":"pat","service":"github","account":"default","access_token":"ghp_test"}`,
	}
	v := &EnvVarVault{
		getenv:  func(k string) string { return env[k] },
		environ: func() []string { return []string{"ONE_CREDS_GITHUB_DEFAULT=" + env["ONE_CREDS_GITHUB_DEFAULT"]} },
	}

	ref := core.AccountRef{Service: "github", Alias: "default"}
	cred, err := v.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if cred.AccessToken.Reveal() != "ghp_test" {
		t.Errorf("token = %q", cred.AccessToken.Reveal())
	}
	if cred.Provider != core.ProviderPAT {
		t.Errorf("provider = %v", cred.Provider)
	}
}

func TestEnvVarVault_FetchMissing(t *testing.T) {
	v := &EnvVarVault{
		getenv:  func(string) string { return "" },
		environ: func() []string { return nil },
	}
	_, err := v.Fetch(context.Background(), core.AccountRef{Service: "github", Alias: "default"})
	var notAuth core.ErrNotAuthenticated
	if !errors.As(err, &notAuth) {
		t.Errorf("expected ErrNotAuthenticated, got %T: %v", err, err)
	}
}

func TestEnvVarVault_StoreReadOnly(t *testing.T) {
	v := NewEnvVarVault()
	err := v.Store(context.Background(), core.AccountRef{Service: "x", Alias: "y"}, core.Credential{})
	var ro core.ErrReadOnly
	if !errors.As(err, &ro) {
		t.Errorf("expected ErrReadOnly, got %T: %v", err, err)
	}
}

func TestEnvVarVault_List(t *testing.T) {
	v := &EnvVarVault{
		getenv: func(string) string { return "" },
		environ: func() []string {
			return []string{
				"ONE_CREDS_GITHUB_DEFAULT={}",
				"ONE_CREDS_GITHUB_WORK={}",
				"ONE_CREDS_NOTION_DEFAULT={}",
				"UNRELATED=value",
			}
		},
	}
	refs, err := v.List(context.Background(), "github")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d: %+v", len(refs), refs)
	}
	seen := map[core.AccountAlias]bool{}
	for _, r := range refs {
		seen[r.Alias] = true
	}
	if !seen["default"] || !seen["work"] {
		t.Errorf("expected default+work, got %+v", refs)
	}
}
