package catalog

import (
	"context"
	"errors"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// CatalogFS implements ports.Catalog by reading service definitions from the filesystem.
type CatalogFS struct{ root string }

// NewCatalogFS creates a CatalogFS rooted at the given directory.
func NewCatalogFS(root string) *CatalogFS { return &CatalogFS{root: root} }

func (c *CatalogFS) GetService(_ context.Context, _ core.ServiceID) (*core.Service, error) {
	return nil, errors.New("not implemented")
}

func (c *CatalogFS) GetAction(_ context.Context, _ core.ServiceID, _ core.ActionID) (*core.Action, error) {
	return nil, errors.New("not implemented")
}

func (c *CatalogFS) ListServices(_ context.Context) ([]core.Service, error) {
	return nil, errors.New("not implemented")
}

func (c *CatalogFS) GetSkill(_ context.Context, _ core.ServiceID) (string, error) {
	return "", errors.New("not implemented")
}

func (c *CatalogFS) GetGuide(_ context.Context, _ core.ServiceID, _ string) (*core.InstallGuide, error) {
	return nil, errors.New("not implemented")
}

func (c *CatalogFS) ListGuides(_ context.Context, _ core.ServiceID) ([]core.InstallGuide, error) {
	return nil, errors.New("not implemented")
}

var _ ports.Catalog = (*CatalogFS)(nil)
