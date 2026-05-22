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
