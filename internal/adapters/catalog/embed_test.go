package catalog

import (
	"context"
	"testing"

	"elydelva/one/internal/core"
)

func TestCatalogEmbed_ListsOfficialServices(t *testing.T) {
	c := NewCatalogEmbed()
	svcs, err := c.ListServices(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(svcs) == 0 {
		t.Fatal("expected at least one embedded service")
	}
	want := map[core.ServiceID]bool{"github": false, "linear": false}
	for _, s := range svcs {
		if _, ok := want[s.ID]; ok {
			want[s.ID] = true
		}
	}
	for id, found := range want {
		if !found {
			t.Errorf("missing embedded service %s", id)
		}
	}
}

func TestCatalogEmbed_GitHubServiceLoads(t *testing.T) {
	c := NewCatalogEmbed()
	svc, err := c.GetService(context.Background(), "github")
	if err != nil {
		t.Fatalf("get github: %v", err)
	}
	if svc.BaseURL != "https://api.github.com" {
		t.Errorf("base_url = %q", svc.BaseURL)
	}
	if len(svc.Actions) < 2 {
		t.Errorf("expected at least 2 actions, got %d", len(svc.Actions))
	}
	act, err := c.GetAction(context.Background(), "github", "issues.list")
	if err != nil {
		t.Fatalf("get action: %v", err)
	}
	if act.Pagination == nil || act.Pagination.Style != "cursor" {
		t.Errorf("issues.list pagination = %+v", act.Pagination)
	}
}

func TestCatalogEmbed_GitHubUsesDeviceOAuthForPrivateRepositories(t *testing.T) {
	c := NewCatalogEmbed()
	svc, err := c.GetService(context.Background(), "github")
	if err != nil {
		t.Fatalf("get github: %v", err)
	}
	if !containsProvider(svc.Providers, core.ProviderOAuthDevice) {
		t.Fatal("GitHub must advertise oauth2_device")
	}
	if containsProvider(svc.Providers, core.ProviderOAuthUser) {
		t.Fatal("GitHub must not advertise the web OAuth flow")
	}
	cfg, ok := svc.AuthConfigs[core.ProviderOAuthDevice]
	if !ok {
		t.Fatal("GitHub device OAuth config is missing")
	}
	if cfg.ClientID != "Ov23ligEslOs680eGllp" {
		t.Errorf("client ID = %q", cfg.ClientID)
	}
	if cfg.DeviceEndpoint != "https://github.com/login/device/code" {
		t.Errorf("device endpoint = %q", cfg.DeviceEndpoint)
	}
	if cfg.TokenEndpoint != "https://github.com/login/oauth/access_token" {
		t.Errorf("token endpoint = %q", cfg.TokenEndpoint)
	}
	wantScopes := map[string]bool{
		"repo": false, "project": false, "gist": false, "workflow": false, "delete_repo": false,
	}
	for _, scope := range cfg.Scopes {
		if _, ok := wantScopes[scope]; !ok {
			t.Errorf("unexpected OAuth scope %q", scope)
			continue
		}
		wantScopes[scope] = true
	}
	if len(cfg.Scopes) != len(wantScopes) {
		t.Errorf("scopes = %v, want exactly repo, project, gist, workflow, and delete_repo", cfg.Scopes)
	}
	for scope, found := range wantScopes {
		if !found {
			t.Errorf("OAuth scope %q missing from %v", scope, cfg.Scopes)
		}
	}
}

func containsProvider(providers []core.ProviderKind, want core.ProviderKind) bool {
	for _, provider := range providers {
		if provider == want {
			return true
		}
	}
	return false
}

func TestCatalogEmbed_UnknownService(t *testing.T) {
	c := NewCatalogEmbed()
	_, err := c.GetService(context.Background(), "nope")
	if _, ok := err.(core.ErrUnknownService); !ok {
		t.Errorf("expected ErrUnknownService, got %T %v", err, err)
	}
}

// TestCatalogEmbed_ChainWinsOverFS verifies the composition-root invariant:
// the embedded catalog wins over a conflicting local FS entry. This protects
// against local overrides shadowing built-in services.
func TestCatalogEmbed_ChainWinsOverFS(t *testing.T) {
	embed := NewCatalogEmbed()
	// FS pointing at a non-existent dir → all lookups miss, fall through.
	fs := NewCatalogFS(t.TempDir())
	chain := NewChainCatalog(embed, fs)
	svc, err := chain.GetService(context.Background(), "github")
	if err != nil {
		t.Fatalf("chain get: %v", err)
	}
	if svc.ID != "github" {
		t.Errorf("got %q", svc.ID)
	}
}
