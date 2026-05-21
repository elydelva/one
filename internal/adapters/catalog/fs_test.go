package catalog

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
	"elydelva/one/internal/testing/portstest"
)

// fixtureRoot returns the v1-minimal fixture path relative to this test file.
func fixtureRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "testing", "fixture", "catalog", "v1-minimal")
}

func TestCatalogFS_Contract(t *testing.T) {
	root := fixtureRoot(t)
	portstest.RunCatalogTests(t, "CatalogFS", func(_ *testing.T) ports.Catalog {
		return NewCatalogFS(root)
	})
}

func TestCatalogFS_LoadsActionsFromFiles(t *testing.T) {
	c := NewCatalogFS(fixtureRoot(t))
	svc, err := c.GetService(context.Background(), "github")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if svc.BaseURL != "https://api.github.com" {
		t.Errorf("base_url = %q", svc.BaseURL)
	}
	if len(svc.Actions) < 3 {
		t.Errorf("expected at least 3 actions, got %d", len(svc.Actions))
	}
	want := map[core.ActionID]bool{"issues.read": false, "issues.list": false, "issues.create": false}
	for _, a := range svc.Actions {
		if _, ok := want[a.ID]; ok {
			want[a.ID] = true
		}
	}
	for id, found := range want {
		if !found {
			t.Errorf("missing action %s", id)
		}
	}
}

func TestCatalogFS_ActionDecodesRequest(t *testing.T) {
	c := NewCatalogFS(fixtureRoot(t))
	act, err := c.GetAction(context.Background(), "github", "issues.read")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if act.Request == nil {
		t.Fatalf("request nil")
	}
	if act.Request.Method != "GET" {
		t.Errorf("method = %q", act.Request.Method)
	}
	if act.Request.Path != "/repos/{owner}/{repo}/issues/{issue_number}" {
		t.Errorf("path = %q", act.Request.Path)
	}
	if act.Errors[404].Code != "not_found" {
		t.Errorf("error mapping for 404 missing: %+v", act.Errors)
	}
}

func TestCatalogFS_ActionDecodesPagination(t *testing.T) {
	c := NewCatalogFS(fixtureRoot(t))
	act, _ := c.GetAction(context.Background(), "github", "issues.list")
	if act.Pagination == nil {
		t.Fatalf("pagination nil")
	}
	if act.Pagination.Style != "cursor" || act.Pagination.RequestParam != "page" {
		t.Errorf("pagination = %+v", act.Pagination)
	}
}

func TestCatalogFS_AuthInjection(t *testing.T) {
	c := NewCatalogFS(fixtureRoot(t))
	svc, _ := c.GetService(context.Background(), "github")
	inj := svc.Injection[core.ProviderPAT]
	if inj.Header != "Authorization" || inj.Format != "Bearer {access_token}" {
		t.Errorf("injection = %+v", inj)
	}
}

func TestCatalogFS_InputSchemaParseable(t *testing.T) {
	c := NewCatalogFS(fixtureRoot(t))
	act, _ := c.GetAction(context.Background(), "github", "issues.read")
	sch, err := core.ParseInputSchema(act.InputSchema)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := sch.Validate(core.Inputs{}); err == nil {
		t.Errorf("expected required-missing error")
	}
	err = sch.Validate(core.Inputs{"owner": "x", "repo": "y", "issue_number": 1})
	if err != nil {
		t.Errorf("valid inputs rejected: %v", err)
	}
}

// e2eFixtureRoot points at v1-minimal-e2e (used by tests/e2e and by the
// Notion service fixture below).
func e2eFixtureRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "testing", "fixture", "catalog", "v1-minimal-e2e")
}

func TestCatalogFS_NotionFixture(t *testing.T) {
	c := NewCatalogFS(e2eFixtureRoot(t))
	svc, err := c.GetService(context.Background(), "notion")
	if err != nil {
		t.Fatalf("get notion: %v", err)
	}
	if svc.BaseURL != "https://api.notion.com" {
		t.Errorf("base_url = %q", svc.BaseURL)
	}
	if inj := svc.Injection[core.ProviderPAT]; inj.Header != "Authorization" || inj.Format != "Bearer {access_token}" {
		t.Errorf("pat injection = %+v", inj)
	}
	want := map[core.ActionID]string{
		"pages.read":           "2026-03-11", // markdown endpoint, newer version
		"pages.meta":           "2022-06-28",
		"pages.create":         "2022-06-28",
		"pages.append":         "2026-03-11",
		"pages.prepend":        "2026-03-11",
		"pages.replace":        "2026-03-11",
		"pages.find_replace":   "2026-03-11",
		"pages.update":         "2026-03-11",
		"blocks.children.list": "2022-06-28",
		"search":               "2022-06-28",
	}
	seen := map[core.ActionID]bool{}
	for _, a := range svc.Actions {
		wantVer, ok := want[a.ID]
		if !ok {
			continue
		}
		seen[a.ID] = true
		if a.Request == nil {
			t.Errorf("action %s has no request spec", a.ID)
			continue
		}
		if got := a.Request.Headers["Notion-Version"]; got != wantVer {
			t.Errorf("action %s Notion-Version = %q want %q", a.ID, got, wantVer)
		}
	}
	for id := range want {
		if !seen[id] {
			t.Errorf("missing action %s", id)
		}
	}
}
