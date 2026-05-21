# ARCHITECTURE.md

> Technical description of the internal architecture of the `one` binary. For any contributor touching the Go code. For the catalog service format, see [CATALOG.md](./CATALOG.md). For the WASM contract, see [HANDLERS.md](./HANDLERS.md).

## Overview: hexagonal architecture (ports & adapters)

One CLI follows a strict hexagonal architecture. Three concentric layers with a **non-negotiable** dependency rule: arrows always point toward the center.

```
┌────────────────────────────────────────────────────────────────┐
│  Interface utilisateur (cmd/, internal/cli/)                   │
│  • cobra commands                                              │
│  • parsing des flags                                           │
│  • détection TTY                                               │
│  • mapping exit codes                                          │
└────────────────────────────┬───────────────────────────────────┘
                             │
┌────────────────────────────▼───────────────────────────────────┐
│  Application (internal/app/)                                   │
│  Use cases : ExecuteAction, Login, ShowInfo, ListCapabilities, │
│              AddScope, Install, Lock, ...                      │
│  Orchestre le domaine et les ports.                            │
└────┬──────────────────────────────────────────────────┬────────┘
     │                                                  │
┌────▼─────────────────┐  ┌──────────────┐  ┌──────────▼────────┐
│ Domaine              │  │  Ports       │  │  Adapters         │
│ (internal/core/)     │◄─┤(interfaces)  ├─►│ (internal/adapters)
│                      │  │              │  │                   │
│ • Service            │  │ Catalog      │  │ • CatalogFS       │
│ • Action             │  │ Vault        │  │ • CatalogHTTP     │
│ • Scope              │  │ Runtime      │  │ • KeyringVault    │
│ • Permission         │  │ Transport    │  │ • AgeVault        │
│ • Account            │  │ ScopeStore   │  │ • WazeroRuntime   │
│ • Credential         │  │ Renderer     │  │ • DeclarativeRT   │
│ • Skill              │  │ AuthProvider │  │ • OAuthProvider   │
│ • InstallGuide       │  │ Clock        │  │ • PATProvider     │
│ • Errors             │  │ Logger       │  │ • NetHTTP         │
│                      │  │ Crypto       │  │ • TTYRenderer     │
│ (logique pure,       │  │              │  │ • JSONRenderer    │
│  pas de I/O)         │  │              │  │ • FileScopeStore  │
└──────────────────────┘  └──────────────┘  └───────────────────┘
```

**The absolute rule**: `internal/core/` only imports the Go stdlib. No YAML parser, no HTTP client, no external crypto lib, no logger. If a file in `core/` has any dependency other than stdlib, it is an architecture bug — reject it in code review.

## Repo layout

```
one/
├── cmd/
│   └── one/
│       └── main.go              # entry point, composition root (≤80 lignes)
├── internal/
│   ├── core/                    # domaine pur, zéro deps externes
│   │   ├── action.go
│   │   ├── service.go
│   │   ├── scope.go
│   │   ├── permission.go
│   │   ├── account.go
│   │   ├── credential.go
│   │   ├── secret.go            # type Secret avec redaction
│   │   ├── installguide.go
│   │   ├── errors.go
│   │   └── *_test.go            # unit tests table-driven
│   ├── ports/                   # interfaces définies par le domaine
│   │   ├── catalog.go
│   │   ├── vault.go
│   │   ├── runtime.go
│   │   ├── transport.go
│   │   ├── scopestore.go
│   │   ├── renderer.go
│   │   ├── authprovider.go
│   │   ├── clock.go
│   │   ├── logger.go
│   │   └── crypto.go
│   ├── app/                     # use cases
│   │   ├── execute.go           # ExecuteAction (le plus important)
│   │   ├── login.go
│   │   ├── logout.go
│   │   ├── capabilities.go
│   │   ├── info.go
│   │   ├── scope.go             # Show/Add/Remove/Check/Explain
│   │   ├── install.go
│   │   ├── lock.go
│   │   ├── trace.go
│   │   ├── doctor.go
│   │   └── *_test.go            # tests avec fakes
│   ├── adapters/
│   │   ├── catalog/
│   │   │   ├── fs.go
│   │   │   ├── http.go
│   │   │   ├── cached.go        # decorator
│   │   │   └── chain.go         # fallback FS → HTTP
│   │   ├── vault/
│   │   │   ├── keyring.go
│   │   │   ├── age.go
│   │   │   ├── envvar.go
│   │   │   └── chain.go         # composite envvar → keyring → age
│   │   ├── runtime/
│   │   │   ├── declarative.go
│   │   │   ├── wazero.go
│   │   │   └── router.go        # strategy: WASM si action.Handler != nil
│   │   ├── transport/
│   │   │   └── nethttp.go
│   │   ├── renderer/
│   │   │   ├── json.go
│   │   │   └── tty.go
│   │   ├── auth/
│   │   │   ├── oauth_user.go
│   │   │   ├── oauth_device.go
│   │   │   ├── oauth_client.go
│   │   │   ├── token_paste.go
│   │   │   ├── api_key.go
│   │   │   └── aws_keys.go
│   │   ├── scopestore/
│   │   │   ├── file.go
│   │   │   └── merged.go        # base + local
│   │   ├── clock/
│   │   │   ├── system.go
│   │   │   └── fake.go          # pour tests
│   │   └── crypto/
│   │       └── std.go
│   ├── cli/                     # adapter UI (cobra)
│   │   ├── root.go
│   │   ├── exec.go              # one <service> <action>
│   │   ├── login.go
│   │   ├── scope.go
│   │   ├── install.go
│   │   ├── info.go
│   │   ├── capabilities.go
│   │   ├── skill.go
│   │   ├── lock.go
│   │   ├── trace.go
│   │   ├── doctor.go
│   │   └── exit.go              # mapping erreurs → exit codes
│   └── testing/                 # helpers de test, fakes, fixtures
│       ├── fake/
│       │   ├── catalog.go
│       │   ├── vault.go
│       │   ├── runtime.go
│       │   ├── clock.go
│       │   └── ...
│       ├── fixture/
│       │   └── catalog/
│       │       └── v1-minimal/  # un catalog complet pour tests E2E
│       └── portstest/           # contract tests, voir TESTING.md
│           ├── catalog.go
│           ├── vault.go
│           └── runtime.go
├── pkg/                         # API publique, exportable par contributeurs
│   ├── catalog/                 # schémas, types YAML
│   │   ├── schema/
│   │   │   └── v1.json          # JSON Schema du service.yaml
│   │   └── types.go             # struct Go pour le YAML
│   └── handlersdk/              # API host pour les auteurs de WASM
│       ├── README.md
│       └── ...
├── catalog-schema/              # symlink vers pkg/catalog/schema/
├── docs/                        # ce que vous lisez
├── scripts/
│   ├── install.sh               # curl one-liner installer
│   └── build-release.sh
├── .github/
│   └── workflows/
│       ├── test.yml
│       ├── release.yml
│       └── security.yml
├── go.mod
├── go.sum
├── README.md
├── CHANGELOG.md
└── LICENSE
```

`internal/` is crucial: Go prevents any other module from importing these packages. It protects the contract of your binary.

`pkg/` exposes **only** what an external contributor needs (catalog format, handler SDK).

## The domain (internal/core/)

### Principles

- **No I/O.** No `os.*`, `http.*`, or direct `time.Now()` calls (use `ports.Clock` instead).
- **No global sentinel errors.** Errors are named types, tagged by context.
- **Immutability by default.** Domain structs are value objects: no mutating methods — return a new instance instead.
- **Validation at construction time.** If a `Permission` can be constructed, it is valid. No "validate at runtime later".

### Examples

```go
// core/permission.go
package core

import "strings"

type Permission struct {
    Service ServiceID
    Path    PermissionPath  // ex: "pages.read"
}

// NewPermission valide à la construction
func NewPermission(service ServiceID, path string) (Permission, error) {
    if err := validatePath(path); err != nil {
        return Permission{}, ErrInvalidPermission{Reason: err.Error()}
    }
    return Permission{Service: service, Path: PermissionPath(path)}, nil
}

// Matches vérifie si la permission correspond à un pattern (glob)
func (p Permission) Matches(pattern PermissionPattern) bool {
    // pure function, testable directement
}

type PermissionPattern string  // type distinct pour éviter les confusions
```

```go
// core/scope.go
package core

type Scope struct {
    services map[ServiceID]ServiceScope
}

type ServiceScope struct {
    Account AccountAlias
    Allow   []PermissionPattern
    Deny    []PermissionPattern
}

// Allows applique la précédence: deny exact > deny glob > allow exact > allow glob > default deny
func (s Scope) Allows(p Permission) bool {
    svc, ok := s.services[p.Service]
    if !ok { return false }                                  // default deny

    for _, pat := range svc.Deny {
        if p.Matches(pat) && !pat.IsGlob() { return false }  // deny exact
    }
    for _, pat := range svc.Deny {
        if p.Matches(pat) { return false }                   // deny glob
    }
    for _, pat := range svc.Allow {
        if p.Matches(pat) && !pat.IsGlob() { return true }   // allow exact
    }
    for _, pat := range svc.Allow {
        if p.Matches(pat) { return true }                    // allow glob
    }
    return false                                              // default deny
}

// MergedWith retourne un nouveau scope, ne mute pas
func (s Scope) MergedWith(other Scope) Scope { ... }
```

### Typed errors

Business errors are types that implement `error`. No scattered `errors.New("foo")`.

```go
// core/errors.go
package core

type ErrNotAuthenticated struct {
    Service ServiceID
}
func (e ErrNotAuthenticated) Error() string {
    return fmt.Sprintf("not authenticated for service %q", e.Service)
}

type ErrNotInScope struct {
    Permission Permission
}

type ErrSetupRequired struct {
    Service ServiceID
    Guide   string
    Reason  string
    Human   bool
}

type ErrUnknownService struct {
    Service ServiceID
}

type ErrInputValidation struct {
    Field  string
    Reason string
}

type ErrReAuthRequired struct {
    Service ServiceID
}
```

The mapping to exit codes happens at the `cli/exit.go` layer:

```go
// internal/cli/exit.go
package cli

func ExitCodeFor(err error) int {
    if err == nil { return 0 }
    var nauth core.ErrNotAuthenticated
    if errors.As(err, &nauth) { return 2 }
    var nscope core.ErrNotInScope
    if errors.As(err, &nscope) { return 3 }
    var setup core.ErrSetupRequired
    if errors.As(err, &setup) { return 4 }
    var unk core.ErrUnknownService
    if errors.As(err, &unk) { return 5 }
    return 1
}
```

The domain is completely unaware of exit codes. That is an interface-layer concern.

### The `Secret` type

Critical for security. A custom type that masks the value in all logging contexts.

```go
// core/secret.go
package core

import "encoding/json"

type Secret string

func (s Secret) String() string { return "[REDACTED]" }
func (s Secret) GoString() string { return "[REDACTED]" }
func (s Secret) MarshalJSON() ([]byte, error) {
    return []byte(`"[REDACTED]"`), nil
}

// Reveal renvoie la valeur en clair. À utiliser UNIQUEMENT au moment d'injecter
// dans un header HTTP, et nulle part ailleurs.
func (s Secret) Reveal() string { return string(s) }
```

All tokens, refresh tokens, secrets, and passwords go through this type. Audit: `grep -r "string" core/credential.go` must return zero secret fields typed as plain string.

## Ports (internal/ports/)

Interfaces that the domain *expects* and that adapters *implement*. Defined in domain terms, never in terms of external libraries.

### Example: Catalog

```go
// internal/ports/catalog.go
package ports

import (
    "context"
    "one/internal/core"
)

type Catalog interface {
    GetService(ctx context.Context, id core.ServiceID) (*core.Service, error)
    GetAction(ctx context.Context, svc core.ServiceID, action core.ActionID) (*core.Action, error)
    ListServices(ctx context.Context) ([]core.Service, error)
    GetSkill(ctx context.Context, svc core.ServiceID) (string, error)  // markdown
    GetGuide(ctx context.Context, svc core.ServiceID, id string) (*core.InstallGuide, error)
    ListGuides(ctx context.Context, svc core.ServiceID) ([]core.InstallGuide, error)
}
```

**One method = one responsibility.** If you want to add `GetSkillFor()` and `GetSkillBytes()`, you are probably mixing two concerns.

**Always `context.Context` as the first argument.** Enables cancellation, timeout, and trace ID propagation.

### Example: Vault

```go
// internal/ports/vault.go
package ports

type Vault interface {
    Store(ctx context.Context, account core.AccountRef, cred core.Credential) error
    Fetch(ctx context.Context, account core.AccountRef) (core.Credential, error)
    Delete(ctx context.Context, account core.AccountRef) error
    List(ctx context.Context, service core.ServiceID) ([]core.AccountRef, error)
}
```

### Example: Runtime

```go
// internal/ports/runtime.go
package ports

type Runtime interface {
    Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error)
}

type ExecuteRequest struct {
    Action      core.Action
    Inputs      core.Inputs
    Credentials core.Credential
    DryRun      bool
    TraceID     string
}

type ExecuteResult struct {
    Output  json.RawMessage
    Calls   []HTTPCall      // pour audit log
}
```

## Adapters (internal/adapters/)

Concrete implementations of the ports. Each adapter is a package that depends on external libraries (`go-keyring`, `wazero`, `net/http`, etc.).

### Recurring patterns

**Decorator** to add behavior without changing the interface:

```go
// adapters/catalog/cached.go
package catalog

type CachedCatalog struct {
    inner ports.Catalog
    cache *lru.Cache
    ttl   time.Duration
    clock ports.Clock
}

func NewCached(inner ports.Catalog, ttl time.Duration, clock ports.Clock) ports.Catalog {
    return &CachedCatalog{inner: inner, cache: lru.New(...), ttl: ttl, clock: clock}
}

func (c *CachedCatalog) GetService(ctx context.Context, id core.ServiceID) (*core.Service, error) {
    if v, ok := c.cache.Get(id); ok && !c.expired(v) {
        return v.service, nil
    }
    svc, err := c.inner.GetService(ctx, id)
    if err == nil {
        c.cache.Add(id, cachedEntry{svc, c.clock.Now()})
    }
    return svc, err
}
```

`CachedCatalog` *is* a `Catalog` and *contains* a `Catalog`. Pure composition.

**Chain of Responsibility** for fallbacks:

```go
// adapters/vault/chain.go
package vault

type ChainVault struct {
    sources []ports.Vault
}

func NewChain(sources ...ports.Vault) ports.Vault {
    return &ChainVault{sources: sources}
}

func (c *ChainVault) Fetch(ctx context.Context, acc core.AccountRef) (core.Credential, error) {
    for _, s := range c.sources {
        cred, err := s.Fetch(ctx, acc)
        if err == nil { return cred, nil }
        var nauth core.ErrNotAuthenticated
        if !errors.As(err, &nauth) {
            return core.Credential{}, err  // erreur autre que "non trouvé", propage
        }
    }
    return core.Credential{}, core.ErrNotAuthenticated{Service: acc.Service}
}
```

Wired at composition time: `NewChain(NewEnvVar(), NewKeyring(clock), NewAge(path))`.

**Strategy** for the runtime, see router.go:

```go
// adapters/runtime/router.go
package runtime

type RoutingRuntime struct {
    declarative ports.Runtime
    wasm        ports.Runtime
}

func (r *RoutingRuntime) Execute(ctx context.Context, req ports.ExecuteRequest) (ports.ExecuteResult, error) {
    if req.Action.RequiresHandler() {
        return r.wasm.Execute(ctx, req)
    }
    return r.declarative.Execute(ctx, req)
}
```

## Application layer (internal/app/)

Use cases. **One file per use case.** They orchestrate the domain and ports, but contain **no business logic**: logic lives in the domain.

### Standard use case structure

```go
// internal/app/execute.go
package app

type ExecuteAction struct {
    catalog ports.Catalog
    vault   ports.Vault
    runtime ports.Runtime
    scope   ports.ScopeStore
    auth    map[string]ports.AuthProvider  // keyé par provider type
    log     ports.Logger
    clock   ports.Clock
}

func NewExecuteAction(
    catalog ports.Catalog,
    vault ports.Vault,
    runtime ports.Runtime,
    scope ports.ScopeStore,
    auth map[string]ports.AuthProvider,
    log ports.Logger,
    clock ports.Clock,
) *ExecuteAction {
    return &ExecuteAction{
        catalog: catalog, vault: vault, runtime: runtime,
        scope: scope, auth: auth, log: log, clock: clock,
    }
}

type ExecuteInput struct {
    Service    core.ServiceID
    Action     core.ActionID
    Inputs     core.Inputs
    AccountAlt string
    DryRun     bool
    ProjectDir string
}

type ExecuteOutput struct {
    Result   json.RawMessage
    Warnings []string
    TraceID  string
}

func (uc *ExecuteAction) Run(ctx context.Context, in ExecuteInput) (ExecuteOutput, error) {
    traceID := generateTraceID()

    // 1. Resolve action
    action, err := uc.catalog.GetAction(ctx, in.Service, in.Action)
    if err != nil { return ExecuteOutput{}, err }

    // 2. Resolve scope
    scope, err := uc.scope.Load(ctx, in.ProjectDir)
    if err != nil { return ExecuteOutput{}, err }

    // 3. Check permission (domain logic)
    if !scope.Allows(action.Permission) {
        return ExecuteOutput{}, core.ErrNotInScope{Permission: action.Permission}
    }

    // 4. Validate inputs (domain logic)
    if err := action.Validate(in.Inputs); err != nil {
        return ExecuteOutput{}, err
    }

    // 5. Resolve account
    account := scope.AccountFor(in.Service)
    if in.AccountAlt != "" { account = core.AccountAlias(in.AccountAlt) }

    // 6. Fetch credentials
    cred, err := uc.vault.Fetch(ctx, core.AccountRef{
        Service: in.Service, Alias: account,
    })
    if err != nil { return ExecuteOutput{}, err }

    // 7. Refresh if needed
    if cred.NeedsRefresh(uc.clock.Now()) {
        cred, err = uc.refresh(ctx, cred)
        if err != nil { return ExecuteOutput{}, err }
    }

    // 8. Execute
    result, err := uc.runtime.Execute(ctx, ports.ExecuteRequest{
        Action: *action,
        Inputs: in.Inputs,
        Credentials: cred,
        DryRun: in.DryRun,
        TraceID: traceID,
    })
    if err != nil { return ExecuteOutput{}, err }

    // 9. Log audit trail
    uc.log.Info("action executed",
        "service", in.Service,
        "action", in.Action,
        "calls", len(result.Calls),
        "trace_id", traceID,
    )

    return ExecuteOutput{
        Result: result.Output,
        TraceID: traceID,
    }, nil
}

func (uc *ExecuteAction) refresh(ctx context.Context, cred core.Credential) (core.Credential, error) {
    provider, ok := uc.auth[cred.Provider]
    if !ok {
        return cred, fmt.Errorf("no auth provider for %q", cred.Provider)
    }
    refreshed, err := provider.Refresh(ctx, cred)
    if err != nil {
        return cred, core.ErrReAuthRequired{Service: cred.Service}
    }
    uc.vault.Store(ctx, core.AccountRef{Service: cred.Service, Alias: cred.Account}, refreshed)
    return refreshed, nil
}
```

**Important note**: this code is **the complete business orchestration of an execution**. No mention of YAML, WASM, HTTP, or keychain. That is the virtue of hexagonal architecture.

## Composition root (cmd/one/main.go)

The only place where everything is wired together. **No DI framework** (wire, fx, dig). Explicit manual composition.

```go
// cmd/one/main.go
package main

import (
    "context"
    "log/slog"
    "net/http"
    "os"
    "time"

    "one/internal/adapters/auth"
    "one/internal/adapters/catalog"
    "one/internal/adapters/clock"
    "one/internal/adapters/crypto"
    "one/internal/adapters/renderer"
    "one/internal/adapters/runtime"
    "one/internal/adapters/scopestore"
    "one/internal/adapters/transport"
    "one/internal/adapters/vault"
    "one/internal/app"
    "one/internal/cli"
    "one/internal/ports"
)

func main() {
    ctx := context.Background()

    // Infrastructure de base
    clk := clock.NewSystem()
    log := slog.New(slog.NewJSONHandler(os.Stderr, nil))
    httpClient := transport.NewNetHTTP(&http.Client{Timeout: 30 * time.Second})
    cryp := crypto.NewStd()

    // Catalog
    cat := catalog.NewCached(
        catalog.NewChain(
            catalog.NewFS(catalogDir()),
            catalog.NewHTTP(catalogURL(), httpClient),
        ),
        5*time.Minute, clk,
    )

    // Vault
    vlt := vault.NewChain(
        vault.NewEnvVar(),
        vault.NewKeyring(clk),
        vault.NewAge(ageVaultPath(), os.Getenv("ONE_VAULT_PASSPHRASE")),
    )

    // Runtime
    rt := runtime.NewRouting(
        runtime.NewDeclarative(httpClient, cryp, clk),
        runtime.NewWazero(httpClient, cryp, clk, log),
    )

    // Scope
    scope := scopestore.NewMerged(
        scopestore.NewFile(".onerc.yaml"),
        scopestore.NewFile(".onerc.local.yaml"),
    )

    // Auth providers
    providers := map[string]ports.AuthProvider{
        "oauth2_user":       auth.NewOAuthUser(httpClient, clk),
        "oauth2_device":     auth.NewOAuthDevice(httpClient, clk),
        "oauth2_client":     auth.NewOAuthClient(httpClient, clk),
        "token_paste":       auth.NewTokenPaste(httpClient),
        "api_key":           auth.NewAPIKey(httpClient),
        "aws_keys":          auth.NewAWSKeys(),
    }

    // Use cases
    execUC := app.NewExecuteAction(cat, vlt, rt, scope, providers, log, clk)
    capsUC := app.NewListCapabilities(cat, scope)
    infoUC := app.NewShowInfo(cat)
    loginUC := app.NewLogin(cat, vlt, providers, log)
    scopeUC := app.NewManageScope(scope, cat)
    installUC := app.NewInstall(cat, execUC)
    // ...

    // Renderer
    rndr := pickRenderer()

    // CLI root
    root := cli.NewRoot(cli.Deps{
        Execute:      execUC,
        Capabilities: capsUC,
        Info:         infoUC,
        Login:        loginUC,
        Scope:        scopeUC,
        Install:      installUC,
        Renderer:     rndr,
    })

    if err := root.ExecuteContext(ctx); err != nil {
        rndr.Error(err)
        os.Exit(cli.ExitCodeFor(err))
    }
}

func pickRenderer() ports.Renderer {
    if isatty.IsTerminal(os.Stdout.Fd()) {
        return renderer.NewTTY(os.Stdout, os.Stderr)
    }
    return renderer.NewJSON(os.Stdout, os.Stderr)
}
```

**Criterion**: this file must stay under 100 lines. If you are approaching that, it means an intermediate constructor is missing.

## Pattern summary

| Pattern | Where to use it | Why |
|---|---|---|
| **Hexagonal / Ports & Adapters** | Globally | Isolated business core, swappable adapters, maximum testability |
| **Strategy** | RoutingRuntime (declarative vs WASM), AuthProvider | Multiple ways to perform the same operation |
| **Decorator** | CachedCatalog, AuditLogVault | Add behavior without changing the interface |
| **Chain of Responsibility** | ChainVault, ChainCatalog | Ordered fallback between sources |
| **Composition root** | main.go | Explicit DI, no magic, readable |
| **Value object** | Permission, Scope, Account, ServiceID | Immutable, comparable, validated at construction |
| **Specification** | Scope.Allows | Encapsulates a complex decision rule |
| **Typed errors** | core/errors.go | Clean mapping to exit codes, no string-matching |
| **Functional options** | Public constructors | Idiomatic Go, extensibility without breaking the API |

## Anti-patterns to reject in code review

### "Just for this case, we put HTTP in the domain"

**No.** The moment the domain makes a syscall, the architecture starts to rot. Refactor into port + adapter, even if it takes 30 extra minutes.

### Mocks instead of fakes

**Prefer fakes.** An `InMemoryVault` fake that stores in a map is more useful than 50 mock expectations that break on every refactor. See [TESTING.md](./TESTING.md).

### Custom code generation

**Reject.** `go generate` for structs from JSON Schema is fine if it is documented and reproducible. Generating Go code via templates that a human could not reasonably write by hand is hidden technical debt.

### Excessive reflection

**No `reflect.*` in application code.** Tolerated in serialization utilities (YAML parser, JSON Schema); forbidden elsewhere. If you need reflection, your type design is probably weak.

### Fat interfaces

An interface with 10+ methods is suspicious. Split into smaller interfaces. `ports.Catalog` has 6 methods — that is already the acceptable maximum.

### Mutable globals

**None.** No `var defaultClient = http.Client{}` modified at initialization. All configuration passes through constructors.

### `panic()` anywhere outside startup

The binary never `panic`s on user-facing code. It returns a typed error. Panics are only acceptable in `main.go` during wiring (e.g., "client_id env var missing").

### Mixing config and runtime state in a struct

```go
// MAUVAIS
type Vault struct {
    Path string
    Key Secret
    cache map[string]Credential  // runtime state
}

// BON: séparer la config injectée du state runtime
type Vault struct {
    cfg   VaultConfig   // immutable après construction
    state vaultState    // mutations contrôlées via lock
}
```

## Concurrency

The binary is globally **sequential per invocation**. One execution = one main thread. But a few concurrency points to handle:

- **Concurrent token refresh**: if two simultaneous invocations refresh the same token, a race is possible. Solution: file lock in `~/.one/locks/<service>:<account>.lock`.
- **Shared catalog cache across invocations**: not a concern, each invocation has its own process with its own cache.
- **HTTP requests inside a WASM handler**: the handler is synchronous, but the host may handle multiple parallel requests if the API supports it. Keep it simple for v0.

Pattern: **explicit lock acquisition** via `flock(2)` on Linux/macOS, `LockFileEx` on Windows. Abstracted in `internal/adapters/fslock/`.

## Logging

**slog** (Go stdlib `log/slog`) for everything. JSON on stderr in normal mode, more verbose with `--debug`.

The `ports.Logger` interface:

```go
type Logger interface {
    Debug(msg string, attrs ...any)
    Info(msg string, attrs ...any)
    Warn(msg string, attrs ...any)
    Error(msg string, attrs ...any)
    With(attrs ...any) Logger
}
```

**No logging in the domain.** The domain returns errors; it is the application layer that decides whether to log.

**No sensitive logs.** `Secret` values are automatically redacted, but never log raw user inputs.

## Binary lifecycle

An invocation of `one` always follows this cycle:

1. **main.go: composition root.** Wire everything, ~50ms.
2. **cobra: parse args.** Identify the command and flags, ~5ms.
3. **Corresponding use case.** Orchestrate. Variable latency (0ms if no network I/O, 100-2000ms for an API request).
4. **Renderer: print output.** JSON or TTY, ~5ms.
5. **Exit with appropriate code.**

**No daemon, no persistent session.** Each invocation is completely independent. That is what makes the binary trivial to reason about and reproducible.

## Code and contract versioning

- **The binary** is versioned with SemVer. One release `vX.Y.Z`.
- **The `service.yaml` format** is versioned via a `version: 1` field at the root. A new major version of the format requires an explicit migration.
- **The WASM host function contract** is versioned via `host_api_version: 1` in service.yaml. The binary checks compatibility at load time.
- **The `.onerc.yaml` format** is also versioned (`version: 1`).

The three versions are **independent**. Binary `v0.7.3` may support `service.yaml v1`, `host_api v1`, and `onerc v1`, and later `v0.8.0` may support `host_api v1 and v2` in transition.

## Performance: budgets and profiling

Three metrics to protect:

| Metric | Budget | Tool |
|---|---|---|
| Cold start (until `--version` output) | <30ms p99 | Go benchmark |
| Memory at startup | <30 MB RSS | `runtime.MemStats` |
| Cold start until execution of a declarative action | <50ms p99 | Go benchmark |

If a PR degrades beyond a threshold, CI fails. Details in [TESTING.md](./TESTING.md#performance).

Profiling: `pprof` can be enabled in dev via `ONE_PPROF=:6060`, which opens a pprof server. Disabled in release builds via build tag.

---

*This architecture is designed to last several years without a rewrite. Evolution happens by adding adapters (new auth type, new runtime), not by modifying the domain. If an evolution requires touching the domain, it means a new business concept has been discovered: that warrants reflection and probably a mini-RFC.*
