package main

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"elydelva/one/internal/adapters/audit"
	"elydelva/one/internal/adapters/auth"
	"elydelva/one/internal/adapters/catalog"
	adapterclock "elydelva/one/internal/adapters/clock"
	"elydelva/one/internal/adapters/crypto"
	"elydelva/one/internal/adapters/renderer"
	"elydelva/one/internal/adapters/runtime"
	"elydelva/one/internal/adapters/scopestore"
	"elydelva/one/internal/adapters/tap"
	"elydelva/one/internal/adapters/transport"
	"elydelva/one/internal/adapters/vault"
	"elydelva/one/internal/app"
	"elydelva/one/internal/cli"
	"elydelva/one/internal/ports"
)

var version = "dev"

func main() {
	// Infrastructure
	clk := adapterclock.NewSystemClock()
	http := transport.NewNetHTTP(transportOpts()...)
	crp := crypto.NewStdCrypto()
	log := newLogger()

	// Adapters
	tapOps := app.NewTapOps(tapRoot(), tap.New(), tapAllowedHosts()...).WithVerifier(tap.NewVerifier())
	cat := buildCatalog(clk, http, tapOps)

	// Auth providers (built after catalog: OAuth providers resolve endpoints from it).
	authProviders := []ports.AuthProvider{
		auth.NewOAuthUserProvider(http, cat),
		auth.NewOAuthDeviceProvider(http, cat, clk),
		auth.NewOAuthClientProvider(http, cat),
		auth.NewTokenPasteProvider().WithValidation(http, cat),
		auth.NewAPIKeyProvider().WithValidation(http, cat),
		auth.NewAWSKeysProvider().WithValidation(http),
		auth.NewCertificateProvider(),
	}
	vlt := buildVault()
	wasmCache := os.ExpandEnv("$HOME/.one/cache/wasm")
	rt := runtime.NewRouter(
		runtime.NewDeclarativeRuntime(http, clk, runtime.NewCatalogResolver(cat)),
		runtime.NewWazeroRuntime(http, vlt, clk, log, runtime.NewFSHandlerResolver(catalogRoot()), wasmCache),
	)
	scp := scopestore.NewProfileScopeStore(scopestore.NewMergedScopeStore(scopestore.NewFileScopeStore()))
	scpWriter := scopestore.NewFileScopeStore()

	// Renderer: JSON when piped (auto-detect) or when --json is passed
	// explicitly. The root/exec commands disable flag parsing, so the flag is
	// read here directly from os.Args rather than via cobra.
	var rnd ports.Renderer
	if hasJSONFlag(os.Args[1:]) || !isatty() {
		rnd = renderer.NewJSONRendererStd()
	} else {
		rnd = renderer.NewTTYRendererStd()
	}

	// Audit (NDJSON) — wired into all credentialed use cases.
	aud := audit.NewNDJSONAudit(os.ExpandEnv("$HOME/.one/audit"), 0)

	// Use cases
	refresh := app.NewRefreshIfNeeded(vlt, authProviders, clk, log, os.ExpandEnv("$HOME/.one/locks")).WithAudit(aud)

	deps := cli.Deps{
		Execute:      app.NewExecuteAction(cat, vlt, rt, scp, authProviders, log, clk, crp).WithRefresh(refresh).WithAudit(aud),
		Login:        app.NewLogin(vlt, cat, authProviders, log).WithAudit(aud),
		Logout:       app.NewLogout(vlt, log).WithAudit(aud),
		Capabilities: app.NewListCapabilities(cat, scp),
		Info:         app.NewShowInfo(cat),
		ShowScope:    app.NewShowScope(scp),
		AddScope:     app.NewAddScope(scp, scpWriter),
		RemoveScope:  app.NewRemoveScope(scp, scpWriter),
		CheckScope:   app.NewCheckScope(scp),
		ShowGuide:    app.NewShowGuide(cat),
		LockScope:    app.NewLockScope(cat, scp),
		ShowTrace:    app.NewShowTrace(aud, clk),
		RunDoctor:    app.NewRunDoctor(vlt, cat, scp),
		Init:         app.NewInit(scpWriter),
		Accounts:     app.NewListAccounts(vlt),
		Rotate:       app.NewRotate(app.NewLogin(vlt, cat, authProviders, log)),
		Refresh:      app.NewForceRefresh(vlt, refresh),
		VaultStatus:  app.NewVaultStatus(vlt, cat, scp),
		VaultExport:  app.NewVaultExport(vlt, scp),
		VaultImport:  app.NewVaultImport(vlt),
		VaultRotate:  app.NewVaultRotate(vlt, scp, log).WithAudit(aud),
		Skill:        app.NewSkill(),
		CatalogOps:   app.NewCatalogOps(cat, catalogRoot()),
		TapOps:       tapOps,
		Upgrade:      app.NewUpgrade(version),
		Renderer:     rnd,
		Catalog:      cat,
	}

	root := cli.BuildRoot(deps)
	root.Version = version

	if err := root.Execute(); err != nil {
		rnd.RenderError(err)
		os.Exit(cli.ExitCode(err))
	}
}

func newLogger() ports.Logger {
	return &slogLogger{slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))}
}

type slogLogger struct{ l *slog.Logger }

func (s *slogLogger) Debug(msg string, attrs ...any) { s.l.Debug(msg, attrs...) }
func (s *slogLogger) Info(msg string, attrs ...any)  { s.l.Info(msg, attrs...) }
func (s *slogLogger) Warn(msg string, attrs ...any)  { s.l.Warn(msg, attrs...) }
func (s *slogLogger) Error(msg string, attrs ...any) { s.l.Error(msg, attrs...) }
func (s *slogLogger) With(attrs ...any) ports.Logger {
	return &slogLogger{s.l.With(attrs...)}
}

// transportOpts builds NetHTTP options from env vars (test/dev escape hatches).
//
//	ONE_TRANSPORT_ALLOW_HTTP=1        — accept http:// URLs (default: refuse).
//	ONE_TRANSPORT_ALLOWED_HOSTS=h1,h2 — bypass SSRF guard for these hosts (default: none).
//
// Production usage should never set either.
func transportOpts() []transport.Option {
	var opts []transport.Option
	if os.Getenv("ONE_TRANSPORT_ALLOW_HTTP") == "1" {
		opts = append(opts, transport.WithAllowHTTP(true))
	}
	if v := os.Getenv("ONE_TRANSPORT_ALLOWED_HOSTS"); v != "" {
		hosts := splitCSV(v)
		opts = append(opts, transport.WithAllowedHosts(hosts...))
	}
	return opts
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// catalogRoot returns the root directory for the local catalog. Overridable via ONE_CATALOG_ROOT.
func catalogRoot() string {
	if v := os.Getenv("ONE_CATALOG_ROOT"); v != "" {
		return v
	}
	return os.ExpandEnv("$HOME/.one/catalog")
}

// tapRoot returns the root directory for installed taps (registry + clones).
// Overridable via ONE_TAP_ROOT for tests.
func tapRoot() string {
	if v := os.Getenv("ONE_TAP_ROOT"); v != "" {
		return v
	}
	return os.ExpandEnv("$HOME/.one/taps")
}

// tapAllowedHosts returns the git hosts that may host taps. Default: github.com.
// Override with ONE_TAP_ALLOWED_HOSTS=host1,host2 (e.g. gitlab.com,git.acme.corp).
func tapAllowedHosts() []string {
	v := os.Getenv("ONE_TAP_ALLOWED_HOSTS")
	if v == "" {
		return nil
	}
	return splitCSV(v)
}

// buildVault wires the vault chain: EnvVar → Keyring → Age (if ONE_AGE_VAULT_PATH set
// or ONE_AGE_PASSPHRASE present).
func buildVault() ports.Vault {
	layers := []ports.Vault{vault.NewEnvVarVault(), vault.NewKeyringVault("one")}
	path := os.Getenv("ONE_AGE_VAULT_PATH")
	if path == "" {
		path = os.ExpandEnv("$HOME/.one/vault.age")
	}
	if os.Getenv("ONE_AGE_PASSPHRASE") != "" || os.Getenv("ONE_AGE_VAULT_PATH") != "" {
		layers = append(layers, vault.NewAgeVault(path, nil))
	}
	return vault.NewChainVault(layers...)
}

// buildCatalog wires the catalog chain. Resolution order:
//
//  1. Embedded official catalog (compiled into the binary).
//  2. Local FS catalog under $HOME/.one/catalog (user overrides + dev work).
//  3. Installed taps (third-party GitHub repos, TOFU-pinned).
//  4. HTTP catalog (only if ONE_CATALOG_URL is set), wrapped in a 15-min cache.
//
// The official catalog wins on conflict, so neither a local override nor a
// tap can shadow a built-in service.
func buildCatalog(clk ports.Clock, httpc ports.Transport, taps *app.TapOps) ports.Catalog {
	sources := []ports.Catalog{
		catalog.NewCatalogEmbed(),
		catalog.NewCatalogFS(catalogRoot()),
		catalog.NewLazyTapCatalog(tapListerAdapter{ops: taps}, clk.Now, time.Second),
	}
	if url := os.Getenv("ONE_CATALOG_URL"); url != "" {
		httpCat := catalog.NewCatalogHTTP(url, httpc)
		sources = append(sources, catalog.NewCachedCatalog(httpCat, 15*time.Minute, clk))
	}
	return catalog.NewChainCatalog(sources...)
}

// tapListerAdapter projects *app.TapOps onto catalog.TapLister, avoiding a
// circular adapter→app import.
type tapListerAdapter struct{ ops *app.TapOps }

func (a tapListerAdapter) List(ctx context.Context) ([]catalog.TapListEntry, error) {
	taps, err := a.ops.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]catalog.TapListEntry, 0, len(taps))
	for _, t := range taps {
		out = append(out, catalog.TapListEntry{Name: t.Name, CloneDir: a.ops.CloneDir(t.Name)})
	}
	return out, nil
}

func isatty() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// hasJSONFlag reports whether --json (or --json=true) appears anywhere in args.
func hasJSONFlag(args []string) bool {
	for _, a := range args {
		if a == "--json" || a == "--json=true" {
			return true
		}
	}
	return false
}
