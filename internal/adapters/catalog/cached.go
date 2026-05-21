package catalog

import (
	"context"
	"sync"
	"time"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// CachedCatalog wraps a Catalog with an in-memory TTL cache.
// Each method has its own cache keyspace; entries expire when the clock advances
// past their recorded fetch time + TTL.
type CachedCatalog struct {
	inner ports.Catalog
	ttl   time.Duration
	clock ports.Clock

	mu      sync.Mutex
	entries map[string]cacheEntry
}

type cacheEntry struct {
	at    time.Time
	value any
	err   error
}

// NewCachedCatalog wraps inner with a TTL cache. ttl <= 0 disables expiry.
func NewCachedCatalog(inner ports.Catalog, ttl time.Duration, clock ports.Clock) *CachedCatalog {
	return &CachedCatalog{inner: inner, ttl: ttl, clock: clock, entries: map[string]cacheEntry{}}
}

func (c *CachedCatalog) get(key string) (cacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok {
		return cacheEntry{}, false
	}
	if c.ttl > 0 && c.clock.Now().Sub(e.at) > c.ttl {
		delete(c.entries, key)
		return cacheEntry{}, false
	}
	return e, true
}

func (c *CachedCatalog) put(key string, value any, err error) {
	c.mu.Lock()
	c.entries[key] = cacheEntry{at: c.clock.Now(), value: value, err: err}
	c.mu.Unlock()
}

func (c *CachedCatalog) GetService(ctx context.Context, id core.ServiceID) (*core.Service, error) {
	key := "service:" + string(id)
	if e, ok := c.get(key); ok {
		if e.err != nil {
			return nil, e.err
		}
		svc := *e.value.(*core.Service)
		return &svc, nil
	}
	v, err := c.inner.GetService(ctx, id)
	c.put(key, v, err)
	return v, err
}

func (c *CachedCatalog) GetAction(ctx context.Context, svc core.ServiceID, action core.ActionID) (*core.Action, error) {
	key := "action:" + string(svc) + ":" + string(action)
	if e, ok := c.get(key); ok {
		if e.err != nil {
			return nil, e.err
		}
		a := *e.value.(*core.Action)
		return &a, nil
	}
	v, err := c.inner.GetAction(ctx, svc, action)
	c.put(key, v, err)
	return v, err
}

func (c *CachedCatalog) ListServices(ctx context.Context) ([]core.Service, error) {
	key := "list"
	if e, ok := c.get(key); ok {
		if e.err != nil {
			return nil, e.err
		}
		return e.value.([]core.Service), nil
	}
	v, err := c.inner.ListServices(ctx)
	c.put(key, v, err)
	return v, err
}

func (c *CachedCatalog) GetSkill(ctx context.Context, svc core.ServiceID) (string, error) {
	key := "skill:" + string(svc)
	if e, ok := c.get(key); ok {
		if e.err != nil {
			return "", e.err
		}
		return e.value.(string), nil
	}
	v, err := c.inner.GetSkill(ctx, svc)
	c.put(key, v, err)
	return v, err
}

func (c *CachedCatalog) GetGuide(ctx context.Context, svc core.ServiceID, id string) (*core.InstallGuide, error) {
	key := "guide:" + string(svc) + ":" + id
	if e, ok := c.get(key); ok {
		if e.err != nil {
			return nil, e.err
		}
		g := *e.value.(*core.InstallGuide)
		return &g, nil
	}
	v, err := c.inner.GetGuide(ctx, svc, id)
	c.put(key, v, err)
	return v, err
}

func (c *CachedCatalog) ListGuides(ctx context.Context, svc core.ServiceID) ([]core.InstallGuide, error) {
	key := "guides:" + string(svc)
	if e, ok := c.get(key); ok {
		if e.err != nil {
			return nil, e.err
		}
		return e.value.([]core.InstallGuide), nil
	}
	v, err := c.inner.ListGuides(ctx, svc)
	c.put(key, v, err)
	return v, err
}

var _ ports.Catalog = (*CachedCatalog)(nil)
