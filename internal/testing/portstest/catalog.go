package portstest

import (
	"context"
	"testing"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// RunCatalogTests executes the ports.Catalog contract against any implementation.
// Call this in each adapter's test file with a factory that returns a fresh instance.
func RunCatalogTests(t *testing.T, name string, factory func(t *testing.T) ports.Catalog) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Run("GetService returns ErrUnknownService for missing", func(t *testing.T) {
			c := factory(t)
			_, err := c.GetService(context.Background(), "doesnotexist")
			var unk core.ErrUnknownService
			if !isErrorAs(err, &unk) {
				t.Errorf("expected ErrUnknownService, got %T: %v", err, err)
			}
		})

		t.Run("GetAction returns ErrUnknownService for missing service", func(t *testing.T) {
			c := factory(t)
			_, err := c.GetAction(context.Background(), "doesnotexist", "action")
			var unk core.ErrUnknownService
			if !isErrorAs(err, &unk) {
				t.Errorf("expected ErrUnknownService, got %T: %v", err, err)
			}
		})

		t.Run("ListServices returns slice (may be empty)", func(t *testing.T) {
			c := factory(t)
			svcs, err := c.ListServices(context.Background())
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			_ = svcs
		})
	})
}
