package main

import (
	"log/slog"
	"os"

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
	http := transport.NewNetHTTP()
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
	cat := catalog.NewCatalogFS(os.ExpandEnv("$HOME/.one/catalog"))
	vlt := vault.NewChainVault(vault.NewEnvVarVault(), vault.NewKeyringVault("one"))
	rt := runtime.NewRouter(
		runtime.NewDeclarativeRuntime(http, clk),
		runtime.NewWazeroRuntime(http, crp, clk, log),
	)
	scp := scopestore.NewFileScopeStore()

	// Renderer (auto-detect TTY)
	var rnd ports.Renderer
	if isatty() {
		rnd = renderer.NewTTYRendererStd()
	} else {
		rnd = renderer.NewJSONRendererStd()
	}
	_ = rnd

	// Use cases
	deps := cli.Deps{
		Execute:      app.NewExecuteAction(cat, vlt, rt, scp, authProviders, log, clk),
		Login:        app.NewLogin(vlt, cat, authProviders, log),
		Logout:       app.NewLogout(vlt, log),
		Capabilities: app.NewListCapabilities(cat, scp),
		Info:         app.NewShowInfo(cat),
		ShowScope:    app.NewShowScope(scp),
		AddScope:     app.NewAddScope(scp),
		RemoveScope:  app.NewRemoveScope(scp),
		CheckScope:   app.NewCheckScope(scp),
		ShowGuide:    app.NewShowGuide(cat),
		LockScope:    app.NewLockScope(cat, scp),
		ShowTrace:    app.NewShowTrace(),
		RunDoctor:    app.NewRunDoctor(vlt, cat, scp),
	}

	root := cli.BuildRoot(deps)
	root.Version = version

	if err := root.Execute(); err != nil {
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

func isatty() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
