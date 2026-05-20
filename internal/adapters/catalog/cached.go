package catalog

import (
	"context"
	"errors"
	"time"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// CachedCatalog wraps a Catalog with an in-memory TTL cache.
type CachedCatalog struct {
	inner ports.Catalog
	ttl   time.Duration
	clock ports.Clock
}

// NewCachedCatalog wraps inner with a TTL cache.
func NewCachedCatalog(inner ports.Catalog, ttl time.Duration, clock ports.Clock) *CachedCatalog {
	return &CachedCatalog{inner: inner, ttl: ttl, clock: clock}
}

func (c *CachedCatalog) GetService(_ context.Context, _ core.ServiceID) (*core.Service, error) {
	return nil, errors.New("not implemented")
}

func (c *CachedCatalog) GetAction(_ context.Context, _ core.ServiceID, _ core.ActionID) (*core.Action, error) {
	return nil, errors.New("not implemented")
}

func (c *CachedCatalog) ListServices(_ context.Context) ([]core.Service, error) {
	return nil, errors.New("not implemented")
}

func (c *CachedCatalog) GetSkill(_ context.Context, _ core.ServiceID) (string, error) {
	return "", errors.New("not implemented")
}

func (c *CachedCatalog) GetGuide(_ context.Context, _ core.ServiceID, _ string) (*core.InstallGuide, error) {
	return nil, errors.New("not implemented")
}

func (c *CachedCatalog) ListGuides(_ context.Context, _ core.ServiceID) ([]core.InstallGuide, error) {
	return nil, errors.New("not implemented")
}

var _ ports.Catalog = (*CachedCatalog)(nil)
