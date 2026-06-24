package fake

import (
	"context"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// Catalog is an in-memory implementation of ports.Catalog for testing.
type Catalog struct {
	services map[core.ServiceID]core.Service
	guides   map[string]core.InstallGuide
	skills   map[core.ServiceID]string
}

// NewCatalog creates a catalog pre-loaded with the given services.
func NewCatalog(services []core.Service) *Catalog {
	c := &Catalog{
		services: map[core.ServiceID]core.Service{},
		guides:   map[string]core.InstallGuide{},
		skills:   map[core.ServiceID]string{},
	}
	for _, svc := range services {
		c.services[svc.ID] = svc
	}
	return c
}

// WithSkill registers a per-service SKILL.md body and returns the catalog for
// chaining. Used by tests exercising `one info <service>`.
func (c *Catalog) WithSkill(svc core.ServiceID, skill string) *Catalog {
	c.skills[svc] = skill
	return c
}

func (c *Catalog) GetService(_ context.Context, id core.ServiceID) (*core.Service, error) {
	svc, ok := c.services[id]
	if !ok {
		return nil, core.ErrUnknownService{Service: id}
	}
	return &svc, nil
}

func (c *Catalog) GetAction(_ context.Context, svc core.ServiceID, id core.ActionID) (*core.Action, error) {
	s, ok := c.services[svc]
	if !ok {
		return nil, core.ErrUnknownService{Service: svc}
	}
	for i := range s.Actions {
		if s.Actions[i].ID == id {
			return &s.Actions[i], nil
		}
	}
	return nil, core.ErrUnknownAction{Service: svc, Action: id}
}

func (c *Catalog) ListServices(_ context.Context) ([]core.Service, error) {
	svcs := make([]core.Service, 0, len(c.services))
	for _, s := range c.services {
		svcs = append(svcs, s)
	}
	return svcs, nil
}

func (c *Catalog) GetSkill(_ context.Context, svc core.ServiceID) (string, error) {
	skill, ok := c.skills[svc]
	if !ok {
		return "", core.ErrUnknownService{Service: svc}
	}
	return skill, nil
}

func (c *Catalog) GetGuide(_ context.Context, _ core.ServiceID, id string) (*core.InstallGuide, error) {
	g, ok := c.guides[id]
	if !ok {
		return nil, core.ErrUnknownService{Service: "unknown"}
	}
	return &g, nil
}

func (c *Catalog) ListGuides(_ context.Context, svc core.ServiceID) ([]core.InstallGuide, error) {
	var guides []core.InstallGuide
	for _, g := range c.guides {
		if g.Service == svc {
			guides = append(guides, g)
		}
	}
	return guides, nil
}

var _ ports.Catalog = (*Catalog)(nil)
