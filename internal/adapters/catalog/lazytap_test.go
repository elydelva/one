package catalog

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"elydelva/one/internal/core"
)

// fakeLister returns whatever list is set, tracking call count.
type fakeLister struct {
	entries []TapListEntry
	calls   int
}

func (f *fakeLister) List(_ context.Context) ([]TapListEntry, error) {
	f.calls++
	return f.entries, nil
}

func writeService(t *testing.T, dir, id string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, id), 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := "version: 1\nid: " + id + "\nname: " + id + "\nbase_url: https://example.com\n"
	if err := os.WriteFile(filepath.Join(dir, id, "service.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLazyTapCatalog_PicksUpNewTaps(t *testing.T) {
	tap1 := t.TempDir()
	writeService(t, tap1, "alpha")
	tap2 := t.TempDir()
	writeService(t, tap2, "beta")

	lister := &fakeLister{entries: []TapListEntry{{Name: "x/1", CloneDir: tap1}}}
	now := time.Now()
	clock := func() time.Time { return now }
	c := NewLazyTapCatalog(lister, clock, 100*time.Millisecond)

	if _, err := c.GetService(context.Background(), "alpha"); err != nil {
		t.Fatalf("first get: %v", err)
	}
	if _, err := c.GetService(context.Background(), "beta"); err == nil {
		t.Errorf("beta should not be visible yet")
	}

	// Add second tap; cache still warm → no visibility.
	lister.entries = append(lister.entries, TapListEntry{Name: "x/2", CloneDir: tap2})
	if _, err := c.GetService(context.Background(), "beta"); err == nil {
		t.Errorf("beta visible before cache expiry")
	}

	// Expire cache.
	now = now.Add(200 * time.Millisecond)
	if _, err := c.GetService(context.Background(), "beta"); err != nil {
		t.Errorf("beta not visible after expiry: %v", err)
	}
}

func TestLazyTapCatalog_EmptyListerReturnsUnknown(t *testing.T) {
	c := NewLazyTapCatalog(&fakeLister{}, time.Now, time.Second)
	_, err := c.GetService(context.Background(), "anything")
	if _, ok := err.(core.ErrUnknownService); !ok {
		t.Errorf("expected ErrUnknownService, got %T %v", err, err)
	}
	svcs, err := c.ListServices(context.Background())
	if err != nil {
		t.Errorf("list: %v", err)
	}
	if len(svcs) != 0 {
		t.Errorf("expected empty list, got %d", len(svcs))
	}
}

func TestLazyTapCatalog_CachesWithinTTL(t *testing.T) {
	lister := &fakeLister{}
	now := time.Now()
	clock := func() time.Time { return now }
	c := NewLazyTapCatalog(lister, clock, time.Second)
	for i := 0; i < 5; i++ {
		_, _ = c.GetService(context.Background(), "anything")
	}
	if lister.calls != 1 {
		t.Errorf("expected 1 List call within TTL, got %d", lister.calls)
	}
}
