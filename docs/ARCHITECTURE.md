# ARCHITECTURE.md

> Description technique de l'architecture interne du binaire `one`. Pour tout contributeur qui touche au code Go. Pour le format des services du catalogue, voir [CATALOG.md](./CATALOG.md). Pour le contrat WASM, voir [HANDLERS.md](./HANDLERS.md).

## Vue d'ensemble : architecture hexagonale (ports & adapters)

One CLI suit une architecture hexagonale stricte. Trois couches concentriques avec une règle de dépendance **non négociable** : les flèches pointent toujours vers le centre.

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

**La règle absolue** : `internal/core/` n'importe que la stdlib Go. Pas de YAML parser, pas de HTTP client, pas de crypto lib externe, pas de logger. Si un fichier de `core/` a une dépendance autre que stdlib, c'est un bug d'architecture, refuse en code review.

## Layout du repo

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

Le `internal/` est crucial : Go empêche tout autre module d'importer ces packages. Tu protèges le contrat de ton binaire.

`pkg/` n'expose **que** ce dont un contributeur externe a besoin (format catalog, SDK handlers).

## Le domaine (internal/core/)

### Principes

- **Pas d'I/O.** Aucun `os.*`, `http.*`, `time.Now()` direct (passer par `ports.Clock`).
- **Pas d'erreurs sentinelles globales.** Les erreurs sont des types nommés, taggés par contexte.
- **Immutabilité par défaut.** Les structures du domaine sont des value objects : pas de méthodes qui mutent, on retourne une nouvelle instance.
- **Validation à la création.** Si un `Permission` peut être construit, alors il est valide. Pas de "validation au runtime plus tard".

### Exemples

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

### Erreurs typées

Les erreurs métier sont des types qui implémentent `error`. Pas de `errors.New("foo")` dispersé.

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

Le mapping vers exit codes se fait à la couche `cli/exit.go` :

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

Le domaine ignore complètement les exit codes. C'est une concern de l'interface.

### Le type `Secret`

Critique pour la sécurité. Un type custom qui mask la valeur dans tous les contextes de logging.

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

Tous les tokens, refresh tokens, secrets, passwords passent par ce type. Audit : `grep -r "string" core/credential.go` doit retourner zéro champ secret typé en plain string.

## Les ports (internal/ports/)

Interfaces que le domaine *attend* et que les adapters *implémentent*. Définies en termes du domaine, jamais en termes de libs externes.

### Exemple : Catalog

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

**Une seule méthode = une seule responsabilité.** Si tu veux ajouter `GetSkillFor()` et `GetSkillBytes()`, c'est probablement que tu mélanges deux préoccupations.

**Toujours `context.Context` en premier argument.** Permet l'annulation, le timeout, la propagation de trace ID.

### Exemple : Vault

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

### Exemple : Runtime

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

## Les adapters (internal/adapters/)

Implémentations concrètes des ports. Chaque adapter est un package qui dépend de libs externes (`go-keyring`, `wazero`, `net/http`, etc.).

### Patterns récurrents

**Decorator** pour ajouter du comportement sans changer l'interface :

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

`CachedCatalog` est *un* `Catalog` et *contient* un `Catalog`. Composition pure.

**Chain of Responsibility** pour les fallbacks :

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

Composition à la wire : `NewChain(NewEnvVar(), NewKeyring(clock), NewAge(path))`.

**Strategy** pour le runtime, voir router.go :

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

## La couche application (internal/app/)

Use cases. **Un fichier par use case**. Ils orchestrent le domaine et les ports, mais ne contiennent **pas de logique métier** : la logique est dans le domaine.

### Structure standard d'un use case

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

**Note importante** : ce code est **l'orchestration métier complète d'une exécution**. Aucune mention de YAML, WASM, HTTP, keychain. C'est la vertu de l'hexagonal.

## La composition root (cmd/one/main.go)

Le seul endroit où tout est câblé ensemble. **Pas de framework DI** (wire, fx, dig). Composition explicite à la main.

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

**Critère** : ce fichier doit faire moins de 100 lignes. Si tu approches, c'est qu'il manque un constructeur intermédiaire.

## Patterns récapitulatifs

| Pattern | Où l'utiliser | Pourquoi |
|---|---|---|
| **Hexagonal / Ports & Adapters** | Globalement | Cœur métier isolé, adapters interchangeables, testabilité maximale |
| **Strategy** | RoutingRuntime (déclaratif vs WASM), AuthProvider | Plusieurs façons de réaliser une même opération |
| **Decorator** | CachedCatalog, AuditLogVault | Ajouter du comportement sans changer l'interface |
| **Chain of Responsibility** | ChainVault, ChainCatalog | Fallback ordonné entre sources |
| **Composition root** | main.go | DI explicite, aucune magie, lisible |
| **Value object** | Permission, Scope, Account, ServiceID | Immutable, comparable, validation à la création |
| **Specification** | Scope.Allows | Encapsule une règle de décision complexe |
| **Typed errors** | core/errors.go | Mapping propre vers exit codes, pas de string-matching |
| **Functional options** | Constructeurs publics | Idiomatique Go, extensibilité sans casser l'API |

## Anti-patterns à refuser en code review

### "Juste pour ce cas-là, on met du HTTP dans le domaine"

**Non.** Le moment où le domaine fait un syscall, l'architecture commence à pourrir. Reformule en port + adapter, même si ça prend 30 minutes de plus.

### Mocks à la place de fakes

**Préfère les fakes.** Un fake `InMemoryVault` qui stocke dans une map est plus utile que 50 mock expectations qui cassent à chaque refactor. Voir [TESTING.md](./TESTING.md).

### Génération de code custom

**Refuse.** `go generate` pour des structs depuis JSON Schema est ok si c'est documenté et reproductible. Génération de code Go par templates qu'un humain ne pourrait pas raisonnablement écrire à la main est une dette technique masquée.

### Réflection abusive

**Pas de `reflect.*` dans le code applicatif.** Toléré dans les utilitaires de sérialisation (parser de YAML, JSON Schema), interdit ailleurs. Si tu as besoin de réflection, c'est probablement que ton design type est faible.

### Interfaces "fat"

Une interface avec 10+ méthodes est suspecte. Découpe en interfaces plus petites. `ports.Catalog` a 6 méthodes, c'est déjà le maximum acceptable.

### Globals mutables

**Aucun.** Pas de `var defaultClient = http.Client{}` modifié à l'initialisation. Toute config passe par les constructeurs.

### `panic()` ailleurs qu'au démarrage

Le binaire ne `panic` jamais sur du code utilisateur. Il retourne une erreur typée. Les panics ne sont acceptables que dans `main.go` au moment du wiring (genre "client_id env var manquante").

### Mélange config et runtime dans une struct

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

## Concurrence

Le binaire est globalement **séquentiel par invocation**. Une exécution = un thread principal. Mais quelques points de concurrence à gérer :

- **Refresh tokens concurrents** : si deux invocations simultanées refresh le même token, race possible. Solution : file lock dans `~/.one/locks/<service>:<account>.lock`.
- **Cache catalog partagé entre invocations** : pas concerné, chaque invocation a son propre process avec son propre cache.
- **HTTP requests dans un handler WASM** : le handler est synchrone, mais l'host peut gérer plusieurs requêtes parallèles si l'API le supporte. Garde le simple pour v0.

Pattern : **acquisition explicite de lock** via `flock(2)` sur Linux/macOS, `LockFileEx` sur Windows. Abstraction dans `internal/adapters/fslock/`.

## Logging

**slog** (Go stdlib `log/slog`) pour tout. JSON sur stderr en mode normal, plus verbose si `--debug`.

Le `ports.Logger` interface :

```go
type Logger interface {
    Debug(msg string, attrs ...any)
    Info(msg string, attrs ...any)
    Warn(msg string, attrs ...any)
    Error(msg string, attrs ...any)
    With(attrs ...any) Logger
}
```

**Pas de logging dans le domaine.** Le domaine retourne des erreurs, c'est l'application qui décide de logger ou non.

**Pas de logs sensibles.** Les `Secret` sont redactés automatiquement, mais ne logge jamais des inputs utilisateurs bruts.

## Cycle de vie du binaire

Une invocation de `one` suit toujours ce cycle :

1. **main.go : composition root.** Wire tout, ~50ms.
2. **cobra : parse args.** Identifier la commande, les flags, ~5ms.
3. **Use case correspondant.** Orchestrer. Latence variable (0ms si pas d'I/O réseau, 100-2000ms si requête API).
4. **Renderer : print output.** JSON ou TTY, ~5ms.
5. **Exit avec code approprié.**

**Pas de daemon, pas de session persistante.** Chaque invocation est complètement indépendante. C'est ce qui rend le binaire trivial à raisonner et reproductible.

## Versioning du code et des contrats

- **Le binaire** est versionné selon SemVer. Une release `vX.Y.Z`.
- **Le format `service.yaml`** est versionné par un champ `version: 1` à la racine. Une nouvelle version majeure du format requiert une migration explicite.
- **Le contrat des host functions WASM** est versionné via `host_api_version: 1` dans le service.yaml. Le binaire vérifie la compat au load.
- **Le format `.onerc.yaml`** est versionné aussi (`version: 1`).

Les trois versions sont **indépendantes**. Le binaire `v0.7.3` peut très bien supporter `service.yaml v1`, `host_api v1`, et `onerc v1`, et plus tard `v0.8.0` supportera `host_api v1 et v2` en transition.

## Performance : budgets et profiling

Trois métriques à protéger :

| Métrique | Budget | Outil |
|---|---|---|
| Cold start (jusqu'à output de `--version`) | <30ms p99 | benchmark Go |
| Mémoire au démarrage | <30 MB RSS | `runtime.MemStats` |
| Cold start jusqu'à exécution d'une action déclarative | <50ms p99 | benchmark Go |

Si une PR dégrade au-delà du seuil, CI fail. Détails dans [TESTING.md](./TESTING.md#performance).

Profiling : `pprof` activable en dev via `ONE_PPROF=:6060` qui ouvre un serveur pprof. Désactivé en release builds par build tag.

---

*Cette architecture est conçue pour durer plusieurs années sans réécriture. Les évolutions se font par ajout d'adapters (nouveau type d'auth, nouveau runtime), pas par modification du domaine. Si une évolution requiert de toucher au domaine, c'est qu'on découvre un concept métier nouveau : ça mérite réflexion et probablement un mini-RFC.*
