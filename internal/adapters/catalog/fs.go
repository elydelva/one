package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
	pkgcatalog "elydelva/one/pkg/catalog"
)

// CatalogFS implements ports.Catalog by reading service definitions from an
// io/fs.FS. The default constructor wraps a directory on disk; an alternative
// constructor accepts any fs.FS (used by the embedded official catalog).
//
// Layout:
//
//	<root>/<service>/service.yaml
//	<root>/<service>/actions/<action_id>.yaml   (optional; merged with service.yaml > actions)
//	<root>/<service>/SKILL.md                   (optional)
//	<root>/<service>/guides/<guide_id>.md       (Phase 4)
type CatalogFS struct {
	fsys fs.FS
	// root is kept for error messages; "" when the FS is not a directory.
	root string
}

// NewCatalogFS creates a CatalogFS rooted at the given directory on disk.
func NewCatalogFS(root string) *CatalogFS {
	return &CatalogFS{fsys: os.DirFS(root), root: root}
}

// NewCatalogFromFS creates a CatalogFS backed by an arbitrary fs.FS. Paths are
// interpreted as `<service>/service.yaml` relative to the FS root.
func NewCatalogFromFS(fsys fs.FS) *CatalogFS {
	return &CatalogFS{fsys: fsys}
}

func (c *CatalogFS) GetService(_ context.Context, id core.ServiceID) (*core.Service, error) {
	dir := string(id)
	servicePath := path.Join(dir, "service.yaml")
	raw, err := fs.ReadFile(c.fsys, servicePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, core.ErrUnknownService{Service: id}
		}
		return nil, err
	}
	var def pkgcatalog.ServiceDef
	if err := yaml.Unmarshal(raw, &def); err != nil {
		return nil, fmt.Errorf("parse %s: %w", servicePath, err)
	}
	if def.Version != 1 {
		return nil, fmt.Errorf("%s: unsupported version %d", servicePath, def.Version)
	}
	if def.ID == "" {
		def.ID = string(id)
	}

	actions, err := c.loadActions(dir, def)
	if err != nil {
		return nil, err
	}

	svc := &core.Service{
		ID:          core.ServiceID(def.ID),
		Name:        def.Name,
		Description: def.Description,
		BaseURL:     def.BaseURL,
		Providers:   parseProviders(def.Auth.Providers),
		Injection:   parseInjection(def.Auth.Injection),
		AuthConfigs: parseAuthConfigs(def.Auth.Config),
	}
	for _, action := range actions {
		ca, err := toCoreAction(svc.ID, action)
		if err != nil {
			return nil, fmt.Errorf("action %q: %w", action.ID, err)
		}
		svc.Actions = append(svc.Actions, ca)
	}
	sort.Slice(svc.Actions, func(i, j int) bool { return svc.Actions[i].ID < svc.Actions[j].ID })
	return svc, nil
}

func (c *CatalogFS) GetAction(ctx context.Context, svcID core.ServiceID, actionID core.ActionID) (*core.Action, error) {
	svc, err := c.GetService(ctx, svcID)
	if err != nil {
		return nil, err
	}
	for i := range svc.Actions {
		if svc.Actions[i].ID == actionID {
			a := svc.Actions[i]
			return &a, nil
		}
	}
	return nil, core.ErrUnknownAction{Service: svcID, Action: actionID}
}

func (c *CatalogFS) ListServices(ctx context.Context) ([]core.Service, error) {
	entries, err := fs.ReadDir(c.fsys, ".")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []core.Service
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || strings.HasPrefix(e.Name(), "_") {
			continue
		}
		svc, err := c.GetService(ctx, core.ServiceID(e.Name()))
		if err != nil {
			if _, ok := err.(core.ErrUnknownService); ok {
				continue
			}
			return nil, err
		}
		out = append(out, *svc)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (c *CatalogFS) GetSkill(_ context.Context, id core.ServiceID) (string, error) {
	raw, err := fs.ReadFile(c.fsys, path.Join(string(id), "SKILL.md"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return string(raw), nil
}

func (c *CatalogFS) GetGuide(_ context.Context, svc core.ServiceID, id string) (*core.InstallGuide, error) {
	raw, err := fs.ReadFile(c.fsys, path.Join(string(svc), "guides", id+".md"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, core.ErrUnknownService{Service: svc}
		}
		return nil, err
	}
	return parseGuideMarkdown(raw, id, svc)
}

func (c *CatalogFS) ListGuides(_ context.Context, svc core.ServiceID) ([]core.InstallGuide, error) {
	dir := path.Join(string(svc), "guides")
	entries, err := fs.ReadDir(c.fsys, dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []core.InstallGuide
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".md")
		raw, err := fs.ReadFile(c.fsys, path.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		g, err := parseGuideMarkdown(raw, id, svc)
		if err != nil {
			return nil, fmt.Errorf("guide %s: %w", id, err)
		}
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// loadActions merges actions declared inline in service.yaml with per-file actions/<id>.yaml.
// Per-file definitions win on conflict (later override).
func (c *CatalogFS) loadActions(dir string, def pkgcatalog.ServiceDef) ([]pkgcatalog.ActionDef, error) {
	merged := map[string]pkgcatalog.ActionDef{}
	for id, action := range def.Actions {
		if action.ID == "" {
			action.ID = id
		}
		merged[id] = action
	}

	actionsDir := path.Join(dir, "actions")
	entries, err := fs.ReadDir(c.fsys, actionsDir)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	} else {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
				continue
			}
			raw, err := fs.ReadFile(c.fsys, path.Join(actionsDir, e.Name()))
			if err != nil {
				return nil, err
			}
			var action pkgcatalog.ActionDef
			if err := yaml.Unmarshal(raw, &action); err != nil {
				return nil, fmt.Errorf("parse %s: %w", e.Name(), err)
			}
			if action.ID == "" {
				action.ID = strings.TrimSuffix(e.Name(), ".yaml")
			}
			merged[action.ID] = action
		}
	}

	out := make([]pkgcatalog.ActionDef, 0, len(merged))
	for _, a := range merged {
		out = append(out, a)
	}
	return out, nil
}

func toCoreAction(svcID core.ServiceID, def pkgcatalog.ActionDef) (core.Action, error) {
	out := core.Action{
		ID:          core.ActionID(def.ID),
		Service:     svcID,
		Description: def.Description,
		Permission:  core.PermissionPath(def.Permission),
		SideEffects: core.SideEffect(def.SideEffects),
	}
	if def.Request != nil {
		out.Request = &core.RequestSpec{
			Method:  strings.ToUpper(def.Request.Method),
			Path:    def.Request.Path,
			Headers: def.Request.Headers,
			Body:    def.Request.Body,
		}
	}
	if def.Pagination != nil {
		out.Pagination = &core.PaginationSpec{
			Style:           def.Pagination.Style,
			RequestParam:    def.Pagination.RequestParam,
			RequestLocation: def.Pagination.RequestLocation,
			ResponseToken:   def.Pagination.ResponseToken,
			ResponseItems:   def.Pagination.ResponseItems,
			MaxPages:        def.Pagination.MaxPages,
		}
	}
	if def.Handler != nil {
		out.Handler = &core.HandlerRef{
			File:        def.Handler.File,
			Sha256:      def.Handler.Sha256,
			HostAPI:     def.Handler.HostAPIVersion,
			Entry:       def.Handler.Entry,
			Calls:       append([]string(nil), def.Handler.Calls...),
			Credentials: append([]string(nil), def.Handler.Credentials...),
			FailCodes:   append([]string(nil), def.Handler.FailCodes...),
			MemoryMB:    def.Handler.MemoryMB,
			CPUSeconds:  def.Handler.CPUSeconds,
		}
	}
	if len(def.Errors) > 0 {
		out.Errors = map[int]core.ErrorSpec{}
		for k, v := range def.Errors {
			status, err := parseStatus(k)
			if err != nil {
				return core.Action{}, fmt.Errorf("errors[%s]: %w", k, err)
			}
			out.Errors[status] = core.ErrorSpec{
				Code: v.Code, Hint: v.Hint, InstallGuide: v.InstallGuide, Retry: v.Retry, MaxAttempts: v.MaxAttempts,
			}
		}
	}
	if len(def.Inputs) > 0 {
		core_defs := make([]core.InputDef, len(def.Inputs))
		for i, in := range def.Inputs {
			core_defs[i] = convertInputDef(in)
		}
		raw, err := json.Marshal(core_defs)
		if err != nil {
			return core.Action{}, err
		}
		out.InputSchema = raw
	}
	return out, nil
}

func convertInputDef(in pkgcatalog.InputDef) core.InputDef {
	out := core.InputDef{
		Name:        in.Name,
		Type:        core.InputType(in.Type),
		Required:    in.Required,
		Location:    core.InputLocation(in.Location),
		Description: in.Description,
		Pattern:     in.Pattern,
		Enum:        in.Enum,
		Default:     in.Default,
		Min:         in.Min,
		Max:         in.Max,
		MinLen:      in.MinLen,
		MaxLen:      in.MaxLen,
	}
	if in.Items != nil {
		item := convertInputDef(*in.Items)
		out.Items = &item
	}
	return out
}

func parseStatus(s string) (int, error) {
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return 0, err
	}
	if v < 100 || v > 599 {
		return 0, fmt.Errorf("invalid HTTP status %d", v)
	}
	return v, nil
}

func parseProviders(in []string) []core.ProviderKind {
	out := make([]core.ProviderKind, 0, len(in))
	for _, p := range in {
		out = append(out, core.ProviderKind(p))
	}
	return out
}

func parseInjection(in map[string]pkgcatalog.InjectionDef) map[core.ProviderKind]core.AuthInjection {
	if len(in) == 0 {
		return nil
	}
	out := map[core.ProviderKind]core.AuthInjection{}
	for k, v := range in {
		out[core.ProviderKind(k)] = core.AuthInjection{Header: v.Header, Format: v.Format}
	}
	return out
}

var _ ports.Catalog = (*CatalogFS)(nil)
