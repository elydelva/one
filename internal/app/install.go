package app

import (
	"context"
	"errors"

	"elydelva/one/internal/ports"
)

// InstallInput holds parameters for the ShowGuide use case.
type InstallInput struct {
	Service string
	Guide   string
}

// ShowGuide renders an install guide for a service.
type ShowGuide struct{ catalog ports.Catalog }

// NewShowGuide creates a ShowGuide use case.
func NewShowGuide(catalog ports.Catalog) *ShowGuide { return &ShowGuide{catalog: catalog} }

// Run returns the guide markdown.
func (uc *ShowGuide) Run(_ context.Context, _ InstallInput) (string, error) {
	return "", errors.New("not implemented")
}
