package catalog

import (
	"context"
	"errors"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// ChainCatalog tries each catalog left-to-right; first non-"unknown" answer wins.
// `core.ErrUnknownService` and `core.ErrUnknownAction` are treated as "miss" and
// fall through to the next catalog; any other error short-circuits.
type ChainCatalog struct{ catalogs []ports.Catalog }

// NewChainCatalog creates a chain from the given catalogs (tried left to right).
func NewChainCatalog(catalogs ...ports.Catalog) *ChainCatalog {
	return &ChainCatalog{catalogs: catalogs}
}

// isMiss reports whether the error means "not found in this catalog" (try next).
func isMiss(err error) bool {
	if err == nil {
		return false
	}
	var unkS core.ErrUnknownService
	var unkA core.ErrUnknownAction
	return errors.As(err, &unkS) || errors.As(err, &unkA)
}

func (c *ChainCatalog) GetService(ctx context.Context, id core.ServiceID) (*core.Service, error) {
	var lastErr error = core.ErrUnknownService{Service: id}
	for _, cat := range c.catalogs {
		v, err := cat.GetService(ctx, id)
		if err == nil {
			return v, nil
		}
		if isMiss(err) {
			lastErr = err
			continue
		}
		return nil, err
	}
	return nil, lastErr
}

func (c *ChainCatalog) GetAction(ctx context.Context, svc core.ServiceID, action core.ActionID) (*core.Action, error) {
	var lastErr error = core.ErrUnknownAction{Service: svc, Action: action}
	for _, cat := range c.catalogs {
		v, err := cat.GetAction(ctx, svc, action)
		if err == nil {
			return v, nil
		}
		if isMiss(err) {
			lastErr = err
			continue
		}
		return nil, err
	}
	return nil, lastErr
}

// ListServices unions the result across all catalogs, dedupe by ID
// (first catalog wins on conflict).
func (c *ChainCatalog) ListServices(ctx context.Context) ([]core.Service, error) {
	seen := map[core.ServiceID]bool{}
	var out []core.Service
	for _, cat := range c.catalogs {
		svcs, err := cat.ListServices(ctx)
		if err != nil {
			if isMiss(err) {
				continue
			}
			return nil, err
		}
		for _, s := range svcs {
			if seen[s.ID] {
				continue
			}
			seen[s.ID] = true
			out = append(out, s)
		}
	}
	return out, nil
}

func (c *ChainCatalog) GetSkill(ctx context.Context, svc core.ServiceID) (string, error) {
	for _, cat := range c.catalogs {
		v, err := cat.GetSkill(ctx, svc)
		if err == nil && v != "" {
			return v, nil
		}
		if err != nil && !isMiss(err) {
			return "", err
		}
	}
	return "", nil
}

func (c *ChainCatalog) GetGuide(ctx context.Context, svc core.ServiceID, id string) (*core.InstallGuide, error) {
	var lastErr error = core.ErrNotSupported{Source: "catalog/chain", Op: "GetGuide " + id}
	for _, cat := range c.catalogs {
		v, err := cat.GetGuide(ctx, svc, id)
		if err == nil {
			return v, nil
		}
		var notSup core.ErrNotSupported
		if isMiss(err) || errors.As(err, &notSup) {
			lastErr = err
			continue
		}
		return nil, err
	}
	return nil, lastErr
}

func (c *ChainCatalog) ListGuides(ctx context.Context, svc core.ServiceID) ([]core.InstallGuide, error) {
	seen := map[string]bool{}
	var out []core.InstallGuide
	for _, cat := range c.catalogs {
		gs, err := cat.ListGuides(ctx, svc)
		if err != nil {
			var notSup core.ErrNotSupported
			if isMiss(err) || errors.As(err, &notSup) {
				continue
			}
			return nil, err
		}
		for _, g := range gs {
			if seen[g.ID] {
				continue
			}
			seen[g.ID] = true
			out = append(out, g)
		}
	}
	return out, nil
}

var _ ports.Catalog = (*ChainCatalog)(nil)
