package app

import (
	"context"
	"sort"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// CapabilitiesInput holds parameters for the ListCapabilities use case.
type CapabilitiesInput struct {
	Service    string
	ProjectDir string
	ScopeOnly  bool
}

// CapabilitiesOutput holds the introspection result.
type CapabilitiesOutput struct {
	Services []ports.ServiceCapability
}

// ListCapabilities returns the actions available either in the current scope
// (ScopeOnly) or across the whole catalog (catalog-backed introspection).
type ListCapabilities struct {
	catalog ports.Catalog
	scope   ports.ScopeStore
}

// NewListCapabilities creates a ListCapabilities use case.
func NewListCapabilities(catalog ports.Catalog, scope ports.ScopeStore) *ListCapabilities {
	return &ListCapabilities{catalog: catalog, scope: scope}
}

// Run returns the capabilities. With ScopeOnly it derives them from .onerc.yaml
// (only what the project allows). Otherwise it introspects the whole catalog and
// lists every action a service exposes.
func (uc *ListCapabilities) Run(ctx context.Context, in CapabilitiesInput) (CapabilitiesOutput, error) {
	if !in.ScopeOnly {
		return uc.fromCatalog(ctx, in)
	}
	scope, err := uc.scope.Load(in.ProjectDir)
	if err != nil {
		return CapabilitiesOutput{}, err
	}

	wantSvc := core.ServiceID(in.Service)
	var ids []core.ServiceID
	for id := range scope.Services {
		if wantSvc != "" && id != wantSvc {
			continue
		}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	out := CapabilitiesOutput{Services: make([]ports.ServiceCapability, 0, len(ids))}
	for _, id := range ids {
		ss := scope.Services[id]
		actions := make([]string, 0, len(ss.Allow))
		for _, p := range ss.Allow {
			actions = append(actions, string(p))
		}
		out.Services = append(out.Services, ports.ServiceCapability{
			ID:      string(id),
			Actions: actions,
		})
	}
	return out, nil
}

// fromCatalog lists every action exposed by the catalog, optionally filtered to
// a single service. Unlike ScopeOnly mode (which lists allowed permission
// patterns), this lists concrete action IDs so an agent can discover what
// exists before requesting scope for it.
func (uc *ListCapabilities) fromCatalog(ctx context.Context, in CapabilitiesInput) (CapabilitiesOutput, error) {
	var services []core.Service
	if in.Service != "" {
		svc, err := uc.catalog.GetService(ctx, core.ServiceID(in.Service))
		if err != nil {
			return CapabilitiesOutput{}, err
		}
		services = []core.Service{*svc}
	} else {
		list, err := uc.catalog.ListServices(ctx)
		if err != nil {
			return CapabilitiesOutput{}, err
		}
		services = list
	}

	sort.Slice(services, func(i, j int) bool { return services[i].ID < services[j].ID })

	out := CapabilitiesOutput{Services: make([]ports.ServiceCapability, 0, len(services))}
	for _, svc := range services {
		// ListServices may return services without their actions loaded;
		// fetch the full definition to enumerate them.
		full := &svc
		if len(full.Actions) == 0 {
			if got, err := uc.catalog.GetService(ctx, svc.ID); err == nil {
				full = got
			}
		}
		actions := make([]string, 0, len(full.Actions))
		for _, a := range full.Actions {
			actions = append(actions, string(a.ID))
		}
		sort.Strings(actions)
		out.Services = append(out.Services, ports.ServiceCapability{
			ID:      string(svc.ID),
			Actions: actions,
		})
	}
	return out, nil
}
