package catalog

import (
	"context"
	"sync"
	"time"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// TapLister is the minimal surface LazyTapCatalog needs from app.TapOps;
// declared here to avoid a circular adapter→app import.
type TapLister interface {
	List(ctx context.Context) ([]TapListEntry, error)
}

// TapListEntry is the projection of an app.TapEntry consumed by this adapter.
type TapListEntry struct {
	Name     string
	CloneDir string
}

// Clock is the time source used to expire the snapshot cache. Production
// passes a real clock from ports.Clock; tests can substitute.
type clockFn func() time.Time

// LazyTapCatalog wraps a dynamic list of taps and behaves like a single
// ports.Catalog. It re-reads the registry on every call (with a short TTL
// cache to avoid stat-storms on hot paths). This lets `one tap add` take
// effect in the same process without a restart.
//
// The catalog itself is built fresh per snapshot — taps are CatalogFS instances
// rooted at the clone dir of each registered tap, composed via ChainCatalog
// in registry order.
type LazyTapCatalog struct {
	lister TapLister
	now    clockFn
	ttl    time.Duration

	mu       sync.Mutex
	snapshot ports.Catalog
	expires  time.Time
}

// NewLazyTapCatalog creates a hot-reloading tap catalog.
func NewLazyTapCatalog(lister TapLister, now clockFn, ttl time.Duration) *LazyTapCatalog {
	if now == nil {
		now = time.Now
	}
	if ttl <= 0 {
		ttl = time.Second
	}
	return &LazyTapCatalog{lister: lister, now: now, ttl: ttl}
}

func (c *LazyTapCatalog) current(ctx context.Context) ports.Catalog {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.snapshot != nil && c.now().Before(c.expires) {
		return c.snapshot
	}
	entries, err := c.lister.List(ctx)
	if err != nil || len(entries) == 0 {
		c.snapshot = emptyCatalog{}
	} else {
		cats := make([]ports.Catalog, 0, len(entries))
		for _, e := range entries {
			cats = append(cats, NewTaggedCatalog(NewCatalogFS(e.CloneDir), "tap:"+e.Name))
		}
		c.snapshot = NewChainCatalog(cats...)
	}
	c.expires = c.now().Add(c.ttl)
	return c.snapshot
}

func (c *LazyTapCatalog) GetService(ctx context.Context, id core.ServiceID) (*core.Service, error) {
	return c.current(ctx).GetService(ctx, id)
}
func (c *LazyTapCatalog) GetAction(ctx context.Context, s core.ServiceID, a core.ActionID) (*core.Action, error) {
	return c.current(ctx).GetAction(ctx, s, a)
}
func (c *LazyTapCatalog) ListServices(ctx context.Context) ([]core.Service, error) {
	return c.current(ctx).ListServices(ctx)
}
func (c *LazyTapCatalog) GetSkill(ctx context.Context, s core.ServiceID) (string, error) {
	return c.current(ctx).GetSkill(ctx, s)
}
func (c *LazyTapCatalog) GetGuide(ctx context.Context, s core.ServiceID, id string) (*core.InstallGuide, error) {
	return c.current(ctx).GetGuide(ctx, s, id)
}
func (c *LazyTapCatalog) ListGuides(ctx context.Context, s core.ServiceID) ([]core.InstallGuide, error) {
	return c.current(ctx).ListGuides(ctx, s)
}

// emptyCatalog is a no-op Catalog used when no taps are installed; it answers
// every lookup with the appropriate "unknown" error so ChainCatalog falls
// through cleanly.
type emptyCatalog struct{}

func (emptyCatalog) GetService(_ context.Context, id core.ServiceID) (*core.Service, error) {
	return nil, core.ErrUnknownService{Service: id}
}
func (emptyCatalog) GetAction(_ context.Context, s core.ServiceID, a core.ActionID) (*core.Action, error) {
	return nil, core.ErrUnknownAction{Service: s, Action: a}
}
func (emptyCatalog) ListServices(_ context.Context) ([]core.Service, error) { return nil, nil }
func (emptyCatalog) GetSkill(_ context.Context, _ core.ServiceID) (string, error) {
	return "", nil
}
func (emptyCatalog) GetGuide(_ context.Context, s core.ServiceID, _ string) (*core.InstallGuide, error) {
	return nil, core.ErrUnknownService{Service: s}
}
func (emptyCatalog) ListGuides(_ context.Context, _ core.ServiceID) ([]core.InstallGuide, error) {
	return nil, nil
}

var (
	_ ports.Catalog = (*LazyTapCatalog)(nil)
	_ ports.Catalog = emptyCatalog{}
)
