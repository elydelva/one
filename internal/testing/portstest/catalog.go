package portstest

import (
	"context"
	"testing"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// RunCatalogTests executes the ports.Catalog contract against any implementation.
// The factory must return a Catalog containing at least the "github" service with
// an "issues.read" action whose permission is "issues.read".
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

		t.Run("GetService returns populated service", func(t *testing.T) {
			c := factory(t)
			svc, err := c.GetService(context.Background(), "github")
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			if svc.ID != "github" {
				t.Errorf("id = %q", svc.ID)
			}
			if len(svc.Actions) == 0 {
				t.Errorf("expected at least one action")
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

		t.Run("GetAction returns ErrUnknownAction for missing action", func(t *testing.T) {
			c := factory(t)
			_, err := c.GetAction(context.Background(), "github", "nope.xxx")
			var unk core.ErrUnknownAction
			if !isErrorAs(err, &unk) {
				t.Errorf("expected ErrUnknownAction, got %T: %v", err, err)
			}
		})

		t.Run("GetAction returns action with permission set", func(t *testing.T) {
			c := factory(t)
			act, err := c.GetAction(context.Background(), "github", "issues.read")
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			if act.Permission == "" {
				t.Errorf("permission empty")
			}
			if act.Service != "github" {
				t.Errorf("service = %q", act.Service)
			}
		})

		t.Run("ListServices returns the github service", func(t *testing.T) {
			c := factory(t)
			svcs, err := c.ListServices(context.Background())
			if err != nil {
				t.Fatalf("list: %v", err)
			}
			found := false
			for _, s := range svcs {
				if s.ID == "github" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("github not in ListServices output: %+v", svcs)
			}
		})
	})
}
