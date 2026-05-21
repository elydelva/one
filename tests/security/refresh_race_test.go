package security_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"elydelva/one/internal/app"
	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
	"elydelva/one/internal/testing/fake"
)

// countingRefresher simulates an OAuth provider whose Refresh increments a counter.
type countingRefresher struct {
	calls atomic.Int32
}

func (c *countingRefresher) Supports(kind core.ProviderKind) bool {
	return kind == core.ProviderOAuthUser
}

func (c *countingRefresher) Login(_ context.Context, _ core.ServiceID, _ core.AccountAlias) (core.Credential, error) {
	return core.Credential{}, nil
}

func (c *countingRefresher) Refresh(_ context.Context, cred core.Credential) (core.Credential, error) {
	c.calls.Add(1)
	// Simulate latency so concurrent workers actually race.
	time.Sleep(10 * time.Millisecond)
	exp := time.Now().Add(time.Hour)
	return core.Credential{
		Provider:     cred.Provider,
		Service:      cred.Service,
		Account:      cred.Account,
		AccessToken:  core.NewSecret("fresh"),
		RefreshToken: cred.RefreshToken,
		ExpiresAt:    &exp,
	}, nil
}

func TestRefresh_RaceProducesExactlyOneRefresh(t *testing.T) {
	vault := fake.NewVault()
	expired := time.Now().Add(-time.Minute)
	cred := core.Credential{
		Provider:     core.ProviderOAuthUser,
		Service:      "github",
		Account:      "work",
		AccessToken:  core.NewSecret("stale"),
		RefreshToken: core.NewSecret("rtk"),
		ExpiresAt:    &expired,
	}
	if err := vault.Store(context.Background(), cred.Ref(), cred); err != nil {
		t.Fatal(err)
	}

	refresher := &countingRefresher{}
	clk := fake.NewClock(time.Now())
	log := fake.NewLogger()
	uc := app.NewRefreshIfNeeded(vault, []ports.AuthProvider{refresher}, clk, log, t.TempDir())

	var wg sync.WaitGroup
	results := make([]core.Credential, 10)
	errs := make([]error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = uc.Run(context.Background(), cred)
		}(i)
	}
	wg.Wait()

	for i, e := range errs {
		if e != nil {
			t.Fatalf("goroutine %d: %v", i, e)
		}
	}
	if got := refresher.calls.Load(); got != 1 {
		t.Errorf("expected exactly 1 underlying Refresh, got %d", got)
	}
	for i, r := range results {
		if r.AccessToken.Reveal() != "fresh" {
			t.Errorf("goroutine %d: stale token %q", i, r.AccessToken.Reveal())
		}
	}
}
