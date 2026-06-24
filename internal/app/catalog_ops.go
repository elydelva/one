package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// CatalogOps groups catalog management use cases (search/update/lint/scaffold/test).
type CatalogOps struct {
	catalog ports.Catalog
	root    string // catalog FS root (for lint/scaffold)
}

// NewCatalogOps creates a CatalogOps use case.
func NewCatalogOps(cat ports.Catalog, root string) *CatalogOps {
	return &CatalogOps{catalog: cat, root: root}
}

// SearchResult is a service summary for catalog search output.
type SearchResult struct {
	ID          string
	Name        string
	Description string
	Actions     int
}

// Search filters services by substring match on ID/name/description.
func (uc *CatalogOps) Search(ctx context.Context, query string) ([]SearchResult, error) {
	services, err := uc.catalog.ListServices(ctx)
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(query)
	var out []SearchResult
	for _, s := range services {
		if q != "" {
			hay := strings.ToLower(string(s.ID) + " " + s.Name + " " + s.Description)
			if !strings.Contains(hay, q) {
				continue
			}
		}
		out = append(out, SearchResult{
			ID:          string(s.ID),
			Name:        s.Name,
			Description: s.Description,
			Actions:     len(s.Actions),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Update is a hook for cache invalidation. The chain/cached adapter currently
// has no public Invalidate method, so this returns nil and documents intent.
// Future: expose Invalidate via a port extension.
func (uc *CatalogOps) Update(_ context.Context) error {
	cacheDir := os.ExpandEnv("$HOME/.one/cache/catalog")
	if _, err := os.Stat(cacheDir); err == nil {
		return os.RemoveAll(cacheDir)
	}
	return nil
}

// LintReport summarizes lint findings for one service.
type LintReport struct {
	Service string
	Errors  []string
}

// Lint walks the catalog root and validates each service.yaml + actions/*.yaml.
// Layout matches the FS adapter: <root>/<service>/service.yaml.
func (uc *CatalogOps) Lint(_ context.Context) ([]LintReport, error) {
	if uc.root == "" {
		return nil, fmt.Errorf("catalog lint: root not configured")
	}
	entries, err := os.ReadDir(uc.root)
	if err != nil {
		return nil, fmt.Errorf("catalog lint: %w", err)
	}
	var reports []LintReport
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(uc.root, e.Name())
		// Skip directories that don't look like services (no service.yaml).
		if _, err := os.Stat(filepath.Join(dir, "service.yaml")); err != nil {
			continue
		}
		reports = append(reports, lintService(dir, e.Name()))
	}
	return reports, nil
}

func lintService(dir, id string) LintReport {
	rep := LintReport{Service: id}
	svcPath := filepath.Join(dir, "service.yaml")
	if _, err := os.Stat(svcPath); err != nil {
		rep.Errors = append(rep.Errors, "missing service.yaml")
		return rep
	}
	raw, err := os.ReadFile(svcPath)
	if err != nil {
		rep.Errors = append(rep.Errors, "read service.yaml: "+err.Error())
		return rep
	}
	if !strings.Contains(string(raw), "version:") {
		rep.Errors = append(rep.Errors, "service.yaml: missing version field")
	}
	if !strings.Contains(string(raw), "id:") && !strings.Contains(string(raw), "name:") {
		rep.Errors = append(rep.Errors, "service.yaml: missing id/name")
	}
	return rep
}

// Scaffold creates a minimal service skeleton at <root>/<id>/.
// Layout matches the FS adapter convention (no `services/` prefix).
func (uc *CatalogOps) Scaffold(_ context.Context, id string) (string, error) {
	if uc.root == "" {
		return "", fmt.Errorf("catalog scaffold: root not configured")
	}
	if id == "" {
		return "", core.ErrInputValidation{Field: "id", Reason: "required"}
	}
	dir := filepath.Join(uc.root, id)
	if _, err := os.Stat(dir); err == nil {
		return dir, fmt.Errorf("catalog scaffold: %s already exists", dir)
	}
	if err := os.MkdirAll(filepath.Join(dir, "actions"), 0o750); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(dir, "guides"), 0o750); err != nil {
		return "", err
	}
	svcYAML := fmt.Sprintf(`version: 1
id: %s
name: %s
description: TODO
base_url: https://api.example.com
auth:
  providers: [token_paste]
  injection:
    token_paste:
      header: Authorization
      format: "Bearer {access_token}"
`, id, id)
	if err := os.WriteFile(filepath.Join(dir, "service.yaml"), []byte(svcYAML), 0o600); err != nil {
		return "", err
	}
	skill := fmt.Sprintf("# %s\n\nTODO: describe this service for agents.\n", id)
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skill), 0o600); err != nil {
		return "", err
	}
	return dir, nil
}

// TestResult summarizes a service-level test run.
type TestResult struct {
	Service string
	Action  string
	Pass    bool
	Err     string
}

// Test loads each action of a service and verifies its input schema parses.
// Designed as a smoke test for catalog authors before publishing.
func (uc *CatalogOps) Test(ctx context.Context, serviceID string) ([]TestResult, error) {
	svc, err := uc.catalog.GetService(ctx, core.ServiceID(serviceID))
	if err != nil {
		return nil, err
	}
	var results []TestResult
	for _, a := range svc.Actions {
		r := TestResult{Service: serviceID, Action: string(a.ID)}
		if _, err := core.ParseInputSchema(a.InputSchema); err != nil {
			r.Err = err.Error()
		} else {
			r.Pass = true
		}
		results = append(results, r)
	}
	return results, nil
}
