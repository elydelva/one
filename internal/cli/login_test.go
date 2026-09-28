package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"elydelva/one/internal/app"
	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
	"elydelva/one/internal/testing/fake"
)

func TestLoginCommandConfirmsSuccessfulSignIn(t *testing.T) {
	provider := fake.NewAuthProvider(core.ProviderOAuthDevice)
	provider.LoginFunc = func(_ context.Context, service core.ServiceID, account core.AccountAlias) (core.Credential, error) {
		return core.Credential{Provider: core.ProviderOAuthDevice, Service: service, Account: account, AccessToken: core.NewSecret("token")}, nil
	}
	uc := app.NewLogin(fake.NewVault(), nil, []ports.AuthProvider{provider}, fake.NewLogger())
	cmd := newLoginCommand(uc)
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"github", "--provider", "oauth2_device", "--as", "test"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute login: %v", err)
	}
	if !strings.Contains(stderr.String(), "Signed in to github as test.") {
		t.Errorf("login confirmation = %q", stderr.String())
	}
}
