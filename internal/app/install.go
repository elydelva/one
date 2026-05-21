package app

import (
	"context"
	"fmt"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// InstallInput holds parameters for the ShowGuide use case.
type InstallInput struct {
	Service string
	Guide   string
	List    bool
}

// InstallOutput holds the rendered guide(s).
type InstallOutput struct {
	Service string
	Guide   *core.InstallGuide   // populated for single-guide mode
	All     []core.InstallGuide  // populated when List is true
}

// ShowGuide renders an install guide for a service.
type ShowGuide struct{ catalog ports.Catalog }

// NewShowGuide creates a ShowGuide use case.
func NewShowGuide(catalog ports.Catalog) *ShowGuide { return &ShowGuide{catalog: catalog} }

// Run returns the requested install guide(s).
func (uc *ShowGuide) Run(ctx context.Context, in InstallInput) (InstallOutput, error) {
	if in.Service == "" {
		return InstallOutput{}, core.ErrInputValidation{Field: "service", Reason: "required"}
	}
	out := InstallOutput{Service: in.Service}
	if in.List {
		gs, err := uc.catalog.ListGuides(ctx, core.ServiceID(in.Service))
		if err != nil {
			return out, err
		}
		out.All = gs
		return out, nil
	}
	if in.Guide == "" {
		return out, core.ErrInputValidation{Field: "guide", Reason: "required (or pass --list)"}
	}
	g, err := uc.catalog.GetGuide(ctx, core.ServiceID(in.Service), in.Guide)
	if err != nil {
		return out, fmt.Errorf("install: %s/%s: %w", in.Service, in.Guide, err)
	}
	out.Guide = g
	return out, nil
}
