package app

import (
	"context"
	"errors"

	"elydelva/one/internal/ports"
)

// InfoInput holds parameters for the ShowInfo use case.
type InfoInput struct {
	Service string
	Action  string
}

// InfoOutput holds the markdown documentation.
type InfoOutput struct {
	Markdown string
}

// ShowInfo returns documentation for a service or action.
type ShowInfo struct {
	catalog ports.Catalog
}

// NewShowInfo creates a ShowInfo use case.
func NewShowInfo(catalog ports.Catalog) *ShowInfo {
	return &ShowInfo{catalog: catalog}
}

// Run returns the info.
func (uc *ShowInfo) Run(_ context.Context, _ InfoInput) (InfoOutput, error) {
	return InfoOutput{}, errors.New("not implemented")
}
