package main

import (
	"log/slog"
	"os"
	"strings"

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

	// Auth providers
	authProviders := []ports.AuthProvider{
		auth.NewOAuthUserProvider(http),
		auth.NewOAuthDeviceProvider(http),
		auth.NewOAuthClientProvider(http),
		auth.NewTokenPasteProvider(),
		auth.NewAPIKeyProvider(),
		auth.NewAWSKeysProvider(),
	}

	// Adapters
	cat := catalog.NewCatalogFS(catalogRoot())
	vlt := vault.NewChainVault(vault.NewEnvVarVault(), vault.NewKeyringVault("one"))
	wasmCache := os.ExpandEnv("$HOME/.one/cache/wasm")
	rt := runtime.NewRouter(
		runtime.NewDeclarativeRuntime(http, clk, runtime.NewCatalogResolver(cat)),
		runtime.NewWazeroRuntime(http, vlt, clk, log, runtime.NewFSHandlerResolver(catalogRoot()), wasmCache),
	)
	scp := scopestore.NewFileScopeStore()

	// Renderer (auto-detect TTY)
	var rnd ports.Renderer
	if isatty() {
		rnd = renderer.NewTTYRendererStd()
	} else {
		rnd = renderer.NewJSONRendererStd()
	}

	// Use cases
	deps := cli.Deps{
		Execute:      app.NewExecuteAction(cat, vlt, rt, scp, authProviders, log, clk, crp),
		Login:        app.NewLogin(vlt, cat, authProviders, log),
		Logout:       app.NewLogout(vlt, log),
		Capabilities: app.NewListCapabilities(cat, scp),
		Info:         app.NewShowInfo(cat),
		ShowScope:    app.NewShowScope(scp),
		AddScope:     app.NewAddScope(scp, scp),
		RemoveScope:  app.NewRemoveScope(scp, scp),
		CheckScope:   app.NewCheckScope(scp),
		ShowGuide:    app.NewShowGuide(cat),
		LockScope:    app.NewLockScope(cat, scp),
		ShowTrace:    app.NewShowTrace(),
		RunDoctor:    app.NewRunDoctor(vlt, cat, scp),
		Init:         app.NewInit(scp),
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

func isatty() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
