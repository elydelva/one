package ports

import (
	"context"

	"elydelva/one/internal/core"
)

// Catalog provides access to service definitions and actions.
type Catalog interface {
	GetService(ctx context.Context, id core.ServiceID) (*core.Service, error)
	GetAction(ctx context.Context, svc core.ServiceID, action core.ActionID) (*core.Action, error)
	ListServices(ctx context.Context) ([]core.Service, error)
	GetSkill(ctx context.Context, svc core.ServiceID) (string, error)
	GetGuide(ctx context.Context, svc core.ServiceID, id string) (*core.InstallGuide, error)
	ListGuides(ctx context.Context, svc core.ServiceID) ([]core.InstallGuide, error)
}
