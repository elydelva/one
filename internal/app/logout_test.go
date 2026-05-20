package app

import (
	"context"
	"testing"

	"elydelva/one/internal/core"
	"elydelva/one/internal/testing/fake"
)

func TestLogout_RemovesCredential(t *testing.T) {
	vault := fake.NewVault()
	ref := core.AccountRef{Service: "github", Alias: "default"}
	_ = vault.Store(context.Background(), ref, core.Credential{Service: "github", Account: "default", AccessToken: core.NewSecret("t")})

	uc := NewLogout(vault, fake.NewLogger())
	if err := uc.Run(context.Background(), LogoutInput{Service: "github"}); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := vault.Fetch(context.Background(), ref); err == nil {
		t.Errorf("expected credential gone")
	}
}

func TestLogout_IdempotentWhenMissing(t *testing.T) {
	uc := NewLogout(fake.NewVault(), fake.NewLogger())
	if err := uc.Run(context.Background(), LogoutInput{Service: "github"}); err != nil {
		t.Errorf("expected logout to be idempotent, got %v", err)
	}
}
