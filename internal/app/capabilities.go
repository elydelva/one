package app

import (
	"context"
	"errors"
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

// ListCapabilities returns the actions available in the current scope.
// In v0.1 only ScopeOnly mode is supported (catalog-backed mode arrives in v0.2).
type ListCapabilities struct {
	catalog ports.Catalog
	scope   ports.ScopeStore
}

// NewListCapabilities creates a ListCapabilities use case.
func NewListCapabilities(catalog ports.Catalog, scope ports.ScopeStore) *ListCapabilities {
	return &ListCapabilities{catalog: catalog, scope: scope}
}

// Run returns the capabilities derived from .onerc.yaml.
func (uc *ListCapabilities) Run(_ context.Context, in CapabilitiesInput) (CapabilitiesOutput, error) {
	if !in.ScopeOnly {
		return CapabilitiesOutput{}, errors.New("catalog-backed capabilities not available in v0.1; pass scope_only=true")
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
