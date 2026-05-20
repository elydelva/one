package catalog

import (
	"context"
	"errors"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// ChainCatalog tries each catalog in order, returning the first successful response.
// Used to implement FS-first with HTTP fallback.
type ChainCatalog struct{ catalogs []ports.Catalog }

// NewChainCatalog creates a chain from the given catalogs (tried left to right).
func NewChainCatalog(catalogs ...ports.Catalog) *ChainCatalog {
	return &ChainCatalog{catalogs: catalogs}
}

func (c *ChainCatalog) GetService(_ context.Context, _ core.ServiceID) (*core.Service, error) {
	return nil, errors.New("not implemented")
}

func (c *ChainCatalog) GetAction(_ context.Context, _ core.ServiceID, _ core.ActionID) (*core.Action, error) {
	return nil, errors.New("not implemented")
}

func (c *ChainCatalog) ListServices(_ context.Context) ([]core.Service, error) {
	return nil, errors.New("not implemented")
}

func (c *ChainCatalog) GetSkill(_ context.Context, _ core.ServiceID) (string, error) {
	return "", errors.New("not implemented")
}

func (c *ChainCatalog) GetGuide(_ context.Context, _ core.ServiceID, _ string) (*core.InstallGuide, error) {
	return nil, errors.New("not implemented")
}

func (c *ChainCatalog) ListGuides(_ context.Context, _ core.ServiceID) ([]core.InstallGuide, error) {
	return nil, errors.New("not implemented")
}

var _ ports.Catalog = (*ChainCatalog)(nil)
