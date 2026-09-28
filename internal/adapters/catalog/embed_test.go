package catalog

import (
	"context"
	"strings"
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

func TestCatalogEmbed_GitHubProjectV2Actions(t *testing.T) {
	want := map[core.ActionID]core.PermissionPath{
		"projects.viewer.read": "projects.read",
		"projects.list":        "projects.read",
		"projects.read":        "projects.read",
		"projects.create":      "projects.write",
		"projects.update":      "projects.write",
		"projects.items.add":   "projects.write",
		"projects.delete":      "projects.write",
	}
	catalog := NewCatalogEmbed()
	for id, permission := range want {
		action, err := catalog.GetAction(context.Background(), "github", id)
		if err != nil {
			t.Errorf("get %s: %v", id, err)
			continue
		}
		if action.Permission != permission {
			t.Errorf("%s permission = %q, want %q", id, action.Permission, permission)
		}
		if action.Request == nil || action.Request.Method != "POST" || action.Request.Path != "/graphql" {
			t.Errorf("%s request = %+v, want POST /graphql", id, action.Request)
			continue
		}
		if action.Request.ResponseErrorsKey != "errors" {
			t.Errorf("%s response errors key = %q, want errors", id, action.Request.ResponseErrorsKey)
		}
		if !strings.Contains(action.Request.Body, `"query"`) {
			t.Errorf("%s request body has no fixed GraphQL query: %q", id, action.Request.Body)
		}
		if id == "projects.delete" && !action.IsDestructive() {
			t.Errorf("%s must be marked destructive", id)
		}
	}
}

func TestCatalogEmbed_GitHubRepoDelete(t *testing.T) {
	action, err := NewCatalogEmbed().GetAction(context.Background(), "github", "repos.delete")
	if err != nil {
		t.Fatalf("get repos.delete: %v", err)
	}
	if action.Permission != "repo.delete" {
		t.Errorf("permission = %q, want repo.delete", action.Permission)
	}
	if !action.IsDestructive() {
		t.Error("repos.delete must be destructive")
	}
	if action.Request == nil || action.Request.Method != "DELETE" || action.Request.Path != "/repos/{owner}/{repo}" {
		t.Errorf("request = %+v, want DELETE /repos/{owner}/{repo}", action.Request)
	}
	schema, err := core.ParseInputSchema(action.InputSchema)
	if err != nil {
		t.Fatalf("parse input schema: %v", err)
	}
	for _, name := range []string{"owner", "repo"} {
		found := false
		for _, input := range schema.Defs {
			if input.Name == name && input.Required && input.Location == "path" {
				found = true
			}
		}
		if !found {
			t.Errorf("required path input %q missing from %+v", name, schema)
		}
	}
}

func TestCatalogEmbed_GitHubRepoReadRedactsTemporaryCloneToken(t *testing.T) {
	action, err := NewCatalogEmbed().GetAction(context.Background(), "github", "repos.read")
	if err != nil {
		t.Fatalf("get repos.read: %v", err)
	}
	if action.Request == nil || len(action.Request.ResponseRedactedFields) != 1 || action.Request.ResponseRedactedFields[0] != "temp_clone_token" {
		t.Errorf("redacted fields = %+v, want temp_clone_token", action.Request)
	}
}

func TestCatalogEmbed_GitHubGistDelete(t *testing.T) {
	action, err := NewCatalogEmbed().GetAction(context.Background(), "github", "gists.delete")
	if err != nil {
		t.Fatalf("get gists.delete: %v", err)
	}
	if action.Permission != "gist.write" {
		t.Errorf("permission = %q, want gist.write", action.Permission)
	}
	if !action.IsDestructive() {
		t.Error("gists.delete must be destructive")
	}
	if action.Request == nil || action.Request.Method != "DELETE" || action.Request.Path != "/gists/{gist_id}" {
		t.Errorf("request = %+v, want DELETE /gists/{gist_id}", action.Request)
	}
	schema, err := core.ParseInputSchema(action.InputSchema)
	if err != nil {
		t.Fatalf("parse input schema: %v", err)
	}
	if len(schema.Defs) != 1 || schema.Defs[0].Name != "gist_id" || !schema.Defs[0].Required || schema.Defs[0].Location != "path" {
		t.Errorf("inputs = %+v, want required gist_id path input", schema.Defs)
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
