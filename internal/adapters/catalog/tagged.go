package catalog

import (
	"context"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// TaggedCatalog wraps a catalog and stamps every returned Service / Action
// with a Source tag. Used to mark tap-sourced services so the runtime can
// apply lower-trust rules to them.
type TaggedCatalog struct {
	inner  ports.Catalog
	source string
}

// NewTaggedCatalog wraps inner so that all returned services/actions carry
// the given source tag (e.g. "tap:user/repo").
func NewTaggedCatalog(inner ports.Catalog, source string) *TaggedCatalog {
	return &TaggedCatalog{inner: inner, source: source}
}

func (c *TaggedCatalog) tag(svc *core.Service) *core.Service {
	if svc == nil {
		return nil
	}
	svc.Source = c.source
	for i := range svc.Actions {
		svc.Actions[i].Source = c.source
	}
	return svc
}

func (c *TaggedCatalog) GetService(ctx context.Context, id core.ServiceID) (*core.Service, error) {
	svc, err := c.inner.GetService(ctx, id)
	return c.tag(svc), err
}
func (c *TaggedCatalog) GetAction(ctx context.Context, s core.ServiceID, a core.ActionID) (*core.Action, error) {
	act, err := c.inner.GetAction(ctx, s, a)
	if err != nil {
		return nil, err
	}
	act.Source = c.source
	return act, nil
}
func (c *TaggedCatalog) ListServices(ctx context.Context) ([]core.Service, error) {
	svcs, err := c.inner.ListServices(ctx)
	if err != nil {
		return nil, err
	}
	for i := range svcs {
		svcs[i].Source = c.source
		for j := range svcs[i].Actions {
			svcs[i].Actions[j].Source = c.source
		}
	}
	return svcs, nil
}
func (c *TaggedCatalog) GetSkill(ctx context.Context, s core.ServiceID) (string, error) {
	return c.inner.GetSkill(ctx, s)
}
func (c *TaggedCatalog) GetGuide(ctx context.Context, s core.ServiceID, id string) (*core.InstallGuide, error) {
	return c.inner.GetGuide(ctx, s, id)
}
func (c *TaggedCatalog) ListGuides(ctx context.Context, s core.ServiceID) ([]core.InstallGuide, error) {
	return c.inner.ListGuides(ctx, s)
}

var _ ports.Catalog = (*TaggedCatalog)(nil)
