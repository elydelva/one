package ports

import (
	"context"

	"elydelva/one/internal/core"
)

// Vault stores and retrieves credentials for service accounts.
type Vault interface {
	Store(ctx context.Context, ref core.AccountRef, cred core.Credential) error
	Fetch(ctx context.Context, ref core.AccountRef) (core.Credential, error)
	Delete(ctx context.Context, ref core.AccountRef) error
	List(ctx context.Context, svc core.ServiceID) ([]core.AccountRef, error)
}
