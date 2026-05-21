package catalog

import (
	"context"
	"errors"
	"testing"

	"elydelva/one/internal/core"
)

// fakeCat returns the given service id (one entry) or ErrUnknownService.
type fakeCat struct {
	svc *core.Service
}

func (f *fakeCat) GetService(_ context.Context, id core.ServiceID) (*core.Service, error) {
	if f.svc == nil || f.svc.ID != id {
		return nil, core.ErrUnknownService{Service: id}
	}
	cp := *f.svc
	return &cp, nil
}
func (f *fakeCat) GetAction(_ context.Context, svc core.ServiceID, action core.ActionID) (*core.Action, error) {
	if f.svc == nil || f.svc.ID != svc {
		return nil, core.ErrUnknownService{Service: svc}
	}
	for _, a := range f.svc.Actions {
		if a.ID == action {
			return &a, nil
		}
	}
	return nil, core.ErrUnknownAction{Service: svc, Action: action}
}
func (f *fakeCat) ListServices(_ context.Context) ([]core.Service, error) {
	if f.svc == nil {
		return nil, nil
	}
	return []core.Service{*f.svc}, nil
}
func (f *fakeCat) GetSkill(_ context.Context, svc core.ServiceID) (string, error) {
	if f.svc != nil && f.svc.ID == svc {
		return "skill-" + string(svc), nil
	}
	return "", nil
}
func (f *fakeCat) GetGuide(_ context.Context, svc core.ServiceID, id string) (*core.InstallGuide, error) {
	if f.svc == nil || f.svc.ID != svc {
		return nil, core.ErrNotSupported{Source: "fake", Op: "GetGuide"}
	}
	return &core.InstallGuide{ID: id, Service: svc, Title: "fake"}, nil
}
func (f *fakeCat) ListGuides(_ context.Context, svc core.ServiceID) ([]core.InstallGuide, error) {
	if f.svc == nil || f.svc.ID != svc {
		return nil, nil
	}
	return []core.InstallGuide{{ID: "g1", Service: svc}}, nil
}

func TestChainCatalog_FirstWins(t *testing.T) {
	a := &fakeCat{svc: &core.Service{ID: "github", Name: "A"}}
	b := &fakeCat{svc: &core.Service{ID: "github", Name: "B"}}
	c := NewChainCatalog(a, b)
	svc, err := c.GetService(context.Background(), "github")
	if err != nil {
		t.Fatal(err)
	}
	if svc.Name != "A" {
		t.Errorf("first catalog should win, got %q", svc.Name)
	}
}

func TestChainCatalog_FallsThroughOnMiss(t *testing.T) {
	a := &fakeCat{} // empty
	b := &fakeCat{svc: &core.Service{ID: "notion", Name: "Notion"}}
	c := NewChainCatalog(a, b)
	svc, err := c.GetService(context.Background(), "notion")
	if err != nil {
		t.Fatal(err)
	}
	if svc.ID != "notion" {
		t.Errorf("got %q", svc.ID)
	}
}

func TestChainCatalog_AllMissReturnsUnknown(t *testing.T) {
	c := NewChainCatalog(&fakeCat{}, &fakeCat{})
	_, err := c.GetService(context.Background(), "void")
	var unk core.ErrUnknownService
	if !errors.As(err, &unk) {
		t.Fatalf("got %v", err)
	}
}

func TestChainCatalog_ListServicesUnions(t *testing.T) {
	a := &fakeCat{svc: &core.Service{ID: "github"}}
	b := &fakeCat{svc: &core.Service{ID: "notion"}}
	c := NewChainCatalog(a, b)
	svcs, err := c.ListServices(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(svcs) != 2 {
		t.Fatalf("want 2, got %d", len(svcs))
	}
}

func TestChainCatalog_GetGuideFallsThroughNotSupported(t *testing.T) {
	a := &fakeCat{}
	b := &fakeCat{svc: &core.Service{ID: "github"}}
	c := NewChainCatalog(a, b)
	g, err := c.GetGuide(context.Background(), "github", "setup")
	if err != nil {
		t.Fatal(err)
	}
	if g.ID != "setup" {
		t.Errorf("got %q", g.ID)
	}
}

func TestChainCatalog_PropagatesHardError(t *testing.T) {
	hard := &errCat{err: errors.New("upstream down")}
	c := NewChainCatalog(hard, &fakeCat{svc: &core.Service{ID: "github"}})
	_, err := c.GetService(context.Background(), "github")
	if err == nil || err.Error() != "upstream down" {
		t.Fatalf("expected hard error, got %v", err)
	}
}

type errCat struct{ err error }

func (e *errCat) GetService(context.Context, core.ServiceID) (*core.Service, error) {
	return nil, e.err
}
func (e *errCat) GetAction(context.Context, core.ServiceID, core.ActionID) (*core.Action, error) {
	return nil, e.err
}
func (e *errCat) ListServices(context.Context) ([]core.Service, error)                  { return nil, e.err }
func (e *errCat) GetSkill(context.Context, core.ServiceID) (string, error)              { return "", e.err }
func (e *errCat) GetGuide(context.Context, core.ServiceID, string) (*core.InstallGuide, error) {
	return nil, e.err
}
func (e *errCat) ListGuides(context.Context, core.ServiceID) ([]core.InstallGuide, error) {
	return nil, e.err
}
