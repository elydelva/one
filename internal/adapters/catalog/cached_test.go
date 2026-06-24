package catalog

import (
	"context"
	"errors"
	"testing"
	"time"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
	"elydelva/one/internal/testing/fake"
)

// countingCatalog records call counts and returns a static service or error.
type countingCatalog struct {
	svc   *core.Service
	err   error
	calls int
}

func (c *countingCatalog) GetService(_ context.Context, _ core.ServiceID) (*core.Service, error) {
	c.calls++
	if c.err != nil {
		return nil, c.err
	}
	cp := *c.svc
	return &cp, nil
}
func (c *countingCatalog) GetAction(_ context.Context, _ core.ServiceID, _ core.ActionID) (*core.Action, error) {
	return nil, errors.New("nope")
}
func (c *countingCatalog) ListServices(_ context.Context) ([]core.Service, error) { return nil, nil }
func (c *countingCatalog) GetSkill(_ context.Context, _ core.ServiceID) (string, error) {
	return "", nil
}
func (c *countingCatalog) GetGuide(_ context.Context, _ core.ServiceID, _ string) (*core.InstallGuide, error) {
	return nil, nil
}
func (c *countingCatalog) ListGuides(_ context.Context, _ core.ServiceID) ([]core.InstallGuide, error) {
	return nil, nil
}

var _ ports.Catalog = (*countingCatalog)(nil)

func TestCachedCatalog_HitsCache(t *testing.T) {
	inner := &countingCatalog{svc: &core.Service{ID: "github", Name: "GitHub"}}
	clk := fake.NewClock(time.Unix(1_700_000_000, 0))
	c := NewCachedCatalog(inner, time.Minute, clk)
	for i := 0; i < 3; i++ {
		if _, err := c.GetService(context.Background(), "github"); err != nil {
			t.Fatal(err)
		}
	}
	if inner.calls != 1 {
		t.Errorf("expected 1 underlying call, got %d", inner.calls)
	}
}

func TestCachedCatalog_ExpiresAfterTTL(t *testing.T) {
	inner := &countingCatalog{svc: &core.Service{ID: "github"}}
	clk := fake.NewClock(time.Unix(1_700_000_000, 0))
	c := NewCachedCatalog(inner, time.Minute, clk)

	_, _ = c.GetService(context.Background(), "github")
	clk.Advance(2 * time.Minute)
	_, _ = c.GetService(context.Background(), "github")

	if inner.calls != 2 {
		t.Errorf("expected 2 calls after expiry, got %d", inner.calls)
	}
}

func TestCachedCatalog_PropagatesError(t *testing.T) {
	inner := &countingCatalog{err: core.ErrUnknownService{Service: "nope"}}
	clk := fake.NewClock(time.Unix(1_700_000_000, 0))
	c := NewCachedCatalog(inner, time.Minute, clk)
	_, err := c.GetService(context.Background(), "nope")
	var unk core.ErrUnknownService
	if !errors.As(err, &unk) {
		t.Fatalf("expected ErrUnknownService, got %v", err)
	}
}
