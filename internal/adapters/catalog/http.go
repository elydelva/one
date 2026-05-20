package catalog

import (
	"context"
	"errors"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// CatalogHTTP implements ports.Catalog by fetching service definitions from a remote CDN.
type CatalogHTTP struct {
	baseURL   string
	transport ports.Transport
}

// NewCatalogHTTP creates a CatalogHTTP targeting the given base URL.
func NewCatalogHTTP(baseURL string, transport ports.Transport) *CatalogHTTP {
	return &CatalogHTTP{baseURL: baseURL, transport: transport}
}

func (c *CatalogHTTP) GetService(_ context.Context, _ core.ServiceID) (*core.Service, error) {
	return nil, errors.New("not implemented")
}

func (c *CatalogHTTP) GetAction(_ context.Context, _ core.ServiceID, _ core.ActionID) (*core.Action, error) {
	return nil, errors.New("not implemented")
}

func (c *CatalogHTTP) ListServices(_ context.Context) ([]core.Service, error) {
	return nil, errors.New("not implemented")
}

func (c *CatalogHTTP) GetSkill(_ context.Context, _ core.ServiceID) (string, error) {
	return "", errors.New("not implemented")
}

func (c *CatalogHTTP) GetGuide(_ context.Context, _ core.ServiceID, _ string) (*core.InstallGuide, error) {
	return nil, errors.New("not implemented")
}

func (c *CatalogHTTP) ListGuides(_ context.Context, _ core.ServiceID) ([]core.InstallGuide, error) {
	return nil, errors.New("not implemented")
}

var _ ports.Catalog = (*CatalogHTTP)(nil)
