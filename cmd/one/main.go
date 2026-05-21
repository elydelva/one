package main

import (
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
	cat := buildCatalog(clk, http)

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

	// Renderer (auto-detect TTY)
	var rnd ports.Renderer
	if isatty() {
		rnd = renderer.NewTTYRendererStd()
	} else {
		rnd = renderer.NewJSONRendererStd()
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

// buildCatalog wires the catalog chain. Always includes FS (local catalog);
// if ONE_CATALOG_URL is set, appends Cached(HTTP) as fallback.
func buildCatalog(clk ports.Clock, httpc ports.Transport) ports.Catalog {
	fs := catalog.NewCatalogFS(catalogRoot())
	url := os.Getenv("ONE_CATALOG_URL")
	if url == "" {
		return fs
	}
	httpCat := catalog.NewCatalogHTTP(url, httpc)
	cached := catalog.NewCachedCatalog(httpCat, 15*time.Minute, clk)
	return catalog.NewChainCatalog(fs, cached)
}

func isatty() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
