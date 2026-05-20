package app

import (
	"context"
	"errors"

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
type ListCapabilities struct {
	catalog ports.Catalog
	scope   ports.ScopeStore
}

// NewListCapabilities creates a ListCapabilities use case.
func NewListCapabilities(catalog ports.Catalog, scope ports.ScopeStore) *ListCapabilities {
	return &ListCapabilities{catalog: catalog, scope: scope}
}

// Run returns the capabilities.
func (uc *ListCapabilities) Run(_ context.Context, _ CapabilitiesInput) (CapabilitiesOutput, error) {
	return CapabilitiesOutput{}, errors.New("not implemented")
}
