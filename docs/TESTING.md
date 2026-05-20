# TESTING.md

> Stratégie de test du binaire `one`. Pour les tests des handlers WASM contribués au catalogue, voir [HANDLERS.md](./HANDLERS.md#tests). Pour les tests sécurité spécifiques, voir [SECURITY.md](./SECURITY.md#tests-de-sécurité).

## Vue d'ensemble

Quatre niveaux de tests, en pyramide :

```
            ┌─────────────────┐
            │   E2E (~5%)     │   le binaire compilé + un catalog réel
            ├─────────────────┤
            │ Integration     │
            │    (~10%)       │   plusieurs adapters câblés ensemble
            ├─────────────────┤
            │  Contract       │
            │    (~15%)       │   un adapter contre les contrats des ports
            ├─────────────────┤
            │                 │
            │  Unit (~70%)    │   domaine et logique pure
            │                 │
            └─────────────────┘
```

**Objectif chiffré** : 80%+ de coverage sur `internal/core/`, 70%+ sur `internal/app/`, suffisant pour ne pas perdre la confiance ailleurs.

**Principe directeur** : pour chaque ligne de code, demande-toi "à quel niveau cette ligne devrait être testée". La majorité doit l'être au niveau le plus bas possible.

## Niveau 1 : tests unitaires

Tests purs du domaine et de la logique applicative. Pas d'I/O, pas de network, pas de filesystem.

### Pattern : table-driven

Go a une convention forte pour les tests de fonctions pures. Toujours table-driven, jamais une assertion par fonction.

```go
// internal/core/scope_test.go
func TestScope_Allows(t *testing.T) {
    tests := []struct {
        name     string
        scope    Scope
        perm     Permission
        wantAllow bool
    }{
        {
            name: "exact allow matches",
            scope: NewScope(map[ServiceID]ServiceScope{
                "github": {Allow: []PermissionPattern{"issues.read"}},
            }),
            perm:      mustPerm(t, "github", "issues.read"),
            wantAllow: true,
        },
        {
            name: "glob allow matches",
            scope: NewScope(map[ServiceID]ServiceScope{
                "github": {Allow: []PermissionPattern{"issues.*"}},
            }),
            perm:      mustPerm(t, "github", "issues.create"),
            wantAllow: true,
        },
        {
            name: "deny exact beats allow glob",
            scope: NewScope(map[ServiceID]ServiceScope{
                "github": {
                    Allow: []PermissionPattern{"issues.*"},
                    Deny:  []PermissionPattern{"issues.delete"},
                },
            }),
            perm:      mustPerm(t, "github", "issues.delete"),
            wantAllow: false,
        },
        {
            name:      "default deny on unknown service",
            scope:     NewScope(map[ServiceID]ServiceScope{}),
            perm:      mustPerm(t, "github", "issues.read"),
            wantAllow: false,
        },
        // ... 20+ cas
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := tt.scope.Allows(tt.perm)
            if got != tt.wantAllow {
                t.Errorf("Allows(%v) = %v, want %v", tt.perm, got, tt.wantAllow)
            }
        })
    }
}
```

**Règle** : si une logique a 5+ branches, table-driven. Si 1-2 branches, fonctions séparées simples ok.

### Helpers de construction

Pour réduire le bruit :

```go
func mustPerm(t *testing.T, svc, path string) Permission {
    t.Helper()
    p, err := NewPermission(ServiceID(svc), path)
    if err != nil { t.Fatal(err) }
    return p
}
```

`t.Helper()` indique au framework que les erreurs viennent du caller. Confortable.

### Property-based tests pour les parsers

Pour les parsers (YAML scope, glob patterns) : property-based avec [gopter](https://github.com/leanovate/gopter).

```go
func TestScopeRoundtrip(t *testing.T) {
    properties := gopter.NewProperties(nil)
    properties.Property("parse(serialize(scope)) == scope", prop.ForAll(
        func(scope Scope) bool {
            yaml := scope.MarshalYAML()
            parsed, err := ParseScope(yaml)
            return err == nil && scope.Equal(parsed)
        },
        genScope(),
    ))
    properties.TestingRun(t)
}
```

Coverage indirecte de cas que tu n'aurais pas pensé à écrire à la main.

### Test des erreurs typées

```go
func TestExecute_NotInScope_ReturnsTypedError(t *testing.T) {
    uc := newExecuteWithEmptyScope(t)
    _, err := uc.Run(ctx, ExecuteInput{
        Service: "github", Action: "issues.create",
    })

    var notInScope core.ErrNotInScope
    if !errors.As(err, &notInScope) {
        t.Fatalf("expected ErrNotInScope, got %T", err)
    }
    assert.Equal(t, "issues.create", notInScope.Permission.Path.String())
}
```

Vérifier le **type** d'erreur, pas le message. Le message peut changer, le type est l'API.

## Niveau 2 : contract tests

Pattern critique en architecture hexagonale : un même paquet de tests qu'on exécute contre **chaque implémentation d'un port**.

### Structure

```
internal/testing/portstest/
├── catalog.go            # définit RunCatalogTests(t, factory func() ports.Catalog)
├── vault.go              # définit RunVaultTests(t, factory func() ports.Vault)
├── runtime.go
├── scopestore.go
└── ...
```

### Exemple : contract tests pour Catalog

```go
// internal/testing/portstest/catalog.go
package portstest

import (
    "context"
    "testing"
    "one/internal/core"
    "one/internal/ports"
)

func RunCatalogTests(t *testing.T, name string, factory func(t *testing.T) ports.Catalog) {
    t.Run(name, func(t *testing.T) {
        t.Run("GetService returns known service", func(t *testing.T) {
            c := factory(t)
            svc, err := c.GetService(context.Background(), "github")
            require.NoError(t, err)
            assert.Equal(t, "github", svc.Name)
        })

        t.Run("GetService returns ErrUnknownService for missing", func(t *testing.T) {
            c := factory(t)
            _, err := c.GetService(context.Background(), "doesnotexist")
            var unk core.ErrUnknownService
            assert.ErrorAs(t, err, &unk)
        })

        t.Run("GetAction resolves nested action", func(t *testing.T) {
            c := factory(t)
            action, err := c.GetAction(context.Background(), "github", "issues.create")
            require.NoError(t, err)
            assert.Equal(t, "issues.create", string(action.ID))
        })

        t.Run("ListServices returns at least known", func(t *testing.T) {
            c := factory(t)
            services, err := c.ListServices(context.Background())
            require.NoError(t, err)
            names := serviceNames(services)
            assert.Contains(t, names, "github")
        })

        // ... 15-20 cas de contrat
    })
}
```

### Application

Chaque adapter du port `Catalog` fait tourner ces tests :

```go
// internal/adapters/catalog/fs_test.go
func TestCatalogFS_Contract(t *testing.T) {
    portstest.RunCatalogTests(t, "FS", func(t *testing.T) ports.Catalog {
        return NewFS(filepath.Join("testdata", "catalog-v1-minimal"))
    })
}

// internal/adapters/catalog/http_test.go
func TestCatalogHTTP_Contract(t *testing.T) {
    portstest.RunCatalogTests(t, "HTTP", func(t *testing.T) ports.Catalog {
        srv := startFakeCatalogServer(t)
        return NewHTTP(srv.URL, srv.Client())
    })
}

// internal/adapters/catalog/cached_test.go
func TestCatalogCached_Contract(t *testing.T) {
    portstest.RunCatalogTests(t, "Cached", func(t *testing.T) ports.Catalog {
        inner := NewFS(filepath.Join("testdata", "catalog-v1-minimal"))
        clk := fake.NewClock()
        return NewCached(inner, 5*time.Minute, clk)
    })
}
```

**Bénéfice** : tu peux ajouter un nouvel adapter Catalog (genre `CatalogGitTagged` qui fetch depuis Git tags), tu fais tourner les contract tests, tu sais que tout le reste du système marchera avec.

### Fakes contre mocks : règle stricte

**Préférence forte aux fakes** sur les mocks. Un fake est une implémentation in-memory du port. Un mock vérifie des appels.

```go
// internal/testing/fake/vault.go
type Vault struct {
    creds map[core.AccountRef]core.Credential
    mu    sync.Mutex
}

func NewVault() *Vault {
    return &Vault{creds: map[core.AccountRef]core.Credential{}}
}

func (v *Vault) Store(_ context.Context, ref core.AccountRef, cred core.Credential) error {
    v.mu.Lock()
    defer v.mu.Unlock()
    v.creds[ref] = cred
    return nil
}

func (v *Vault) Fetch(_ context.Context, ref core.AccountRef) (core.Credential, error) {
    v.mu.Lock()
    defer v.mu.Unlock()
    if cred, ok := v.creds[ref]; ok { return cred, nil }
    return core.Credential{}, core.ErrNotAuthenticated{Service: ref.Service}
}
```

**Pourquoi** :

- Le fake fait tourner les contract tests, donc son comportement est garanti correct.
- Pas de mock framework qui casse à chaque refactor de signature.
- Lisible, debuggable.
- Réutilisable entre tests.

Les mocks ne sont acceptables que pour vérifier des side effects difficilement observables (genre "ce port a-t-il bien été appelé avec ces arguments précis ?"). Et même là, préférer un fake qui enregistre les appels via `RecordedCalls()`.

## Niveau 3 : tests d'intégration

Plusieurs adapters câblés ensemble, mais sans réseau réel. Tests dans `internal/app/*_test.go`.

### Exemple : test du use case Execute

```go
// internal/app/execute_test.go
func TestExecuteAction_HappyPath(t *testing.T) {
    // Câblage
    catalog := fake.NewCatalog(fixtures.MinimalCatalog())
    vlt := fake.NewVault()
    rt := fake.NewRuntime()
    scope := fake.NewScopeStore()
    clk := fake.NewClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
    log := slog.New(slog.NewTextHandler(io.Discard, nil))

    // État
    vlt.Store(ctx, core.AccountRef{Service: "github", Alias: "default"}, core.Credential{
        AccessToken: "valid-token",
        ExpiresAt:   ptr(clk.Now().Add(1 * time.Hour)),
    })
    scope.SetScope(".", `version: 1
services:
  github:
    allow: [issues.read]`)
    rt.SetResponse("github.issues.read", core.Inputs{"issue_number": 42},
        json.RawMessage(`{"id":42,"title":"Test issue"}`))

    // Exécution
    uc := app.NewExecuteAction(catalog, vlt, rt, scope, fakeAuthProviders(), log, clk)
    out, err := uc.Run(ctx, app.ExecuteInput{
        Service: "github", Action: "issues.read",
        Inputs: core.Inputs{"issue_number": 42},
        ProjectDir: ".",
    })

    // Assertions
    require.NoError(t, err)
    assert.JSONEq(t, `{"id":42,"title":"Test issue"}`, string(out.Result))
}

func TestExecuteAction_NotInScope_ReturnsError3(t *testing.T) {
    // ... même setup mais scope vide
    _, err := uc.Run(ctx, app.ExecuteInput{...})

    var nsc core.ErrNotInScope
    assert.ErrorAs(t, err, &nsc)
}
```

### Tests des use cases : matrice de cas

Pour `ExecuteAction`, la matrice minimum :

| Cas | Attendu |
|---|---|
| Service inconnu | ErrUnknownService |
| Action inconnue | ErrUnknownAction |
| Pas authentifié | ErrNotAuthenticated |
| Pas dans le scope | ErrNotInScope |
| Inputs invalides | ErrInputValidation |
| Token expiré, refresh OK | succès |
| Token expiré, refresh KO | ErrReAuthRequired |
| Action déclarative OK | succès |
| Action WASM OK | succès |
| Setup requis, action fail | ErrSetupRequired avec hint |
| Action dry-run | succès, pas de side effect |
| Trace ID propagé | TraceID dans output |
| Audit log écrit | log contient l'event |

13 cas. Tous tests via fakes. <1s total.

## Niveau 4 : tests E2E

Le binaire compilé contre un catalog réel. Mais **sans réseau** : on lance un serveur HTTP local qui mock les APIs.

### Structure

```
tests/e2e/
├── e2e_test.go            # tests Go qui invoquent le binaire
├── fixtures/
│   ├── catalog/           # un catalog minimal complet
│   │   └── services/
│   │       └── ...
│   └── projects/
│       ├── ok/
│       │   └── .onerc.yaml
│       └── ...
└── fake_api/              # serveur HTTP qui mime GitHub, Notion, etc.
    ├── server.go
    ├── github.go
    └── ...
```

### Exemple

```go
// tests/e2e/e2e_test.go
func TestE2E_GitHub_IssuesList(t *testing.T) {
    fakeAPI := startFakeAPI(t)
    fakeAPI.GitHub.SetIssuesResponse([]Issue{
        {ID: 1, Title: "First"},
        {ID: 2, Title: "Second"},
    })

    dir := t.TempDir()
    writeFile(t, dir+"/.onerc.yaml", `version: 1
services:
  github:
    allow: [issues.read]`)

    cmd := exec.Command(binaryPath(t),
        "github", "issues.list",
        "--repo", "elydelva/test",
    )
    cmd.Dir = dir
    cmd.Env = append(os.Environ(),
        "ONE_CREDS_GITHUB_DEFAULT={\"access_token\":\"fake-token\",\"provider\":\"pat\",\"service\":\"github\",\"account\":\"default\"}",
        "ONE_CATALOG_DIR="+fixturesCatalogDir(),
        "ONE_GITHUB_API_BASE="+fakeAPI.URL,
    )

    out, err := cmd.CombinedOutput()
    require.NoError(t, err, "output: %s", out)

    var result struct {
        Data []Issue `json:"data"`
    }
    require.NoError(t, json.Unmarshal(out, &result))
    assert.Len(t, result.Data, 2)
}
```

### Cas couverts

5-10 tests E2E maximum, focalisés sur :

- Happy path d'une commande critique (`one <service> <action>`)
- Exit codes attendus (0, 1, 2, 3, 4, 5)
- Output JSON parseable
- Output TTY lisible (snapshot)
- `one capabilities` retourne du JSON valide
- `one info` retourne du markdown valide
- Workflow complet : init → login → scope add → exec

Pas plus. Les E2E sont lents et fragiles, on les garde minimaux.

## Tests cross-platform

Matrice CI sur trois plateformes : Linux, macOS, Windows.

```yaml
# .github/workflows/test.yml
strategy:
  matrix:
    os: [ubuntu-latest, macos-latest, windows-latest]
    go: ['1.23']
```

**Cas spécifiques par OS** :

| Test | Pourquoi |
|---|---|
| Keychain store/fetch | API différente par OS (Keychain/Secret Service/CredMgr) |
| File locks | `flock` Linux/macOS vs `LockFileEx` Windows |
| Path resolution | `/` vs `\`, home dir, XDG |
| TTY detection | comportement `isatty` |
| Exec browser | `open` vs `xdg-open` vs `start` |
| Permissions fichiers | `0600` sous Unix, ACL Windows |

Le keychain Windows en CI est non trivial (besoin d'une session interactive). Solution : tests `--short` qui skippent les tests requérant un keychain réel, plus un test manuel hebdomadaire.

## Tests de sécurité

Suite dédiée taggée `security`, à exécuter en CI sur chaque push :

```bash
go test -tags=security ./tests/security/...
```

### Canary token leak

```go
//go:build security
package security

func TestNoCredentialLeak(t *testing.T) {
    canary := "CANARY_DO_NOT_LEAK_" + uuid.NewString()

    var stdout, stderr, logs bytes.Buffer
    runOne(t, ExecOptions{
        Stdout: &stdout, Stderr: &stderr,
        Env: map[string]string{
            "ONE_CREDS_GITHUB_DEFAULT": fmt.Sprintf(`{"access_token":%q,...}`, canary),
        },
        LogCapture: &logs,
        Args: []string{"github", "issues.read", "--issue", "1"},
    })

    assert.NotContains(t, stdout.String(), canary)
    assert.NotContains(t, stderr.String(), canary)
    assert.NotContains(t, logs.String(), canary)
}
```

### Handler sandbox evasion

```go
func TestSandbox_NoFilesystemAccess(t *testing.T) {
    out := runHandler(t, "tests/security/handlers/try_fs_read.wasm")
    assert.Contains(t, out.Error, "fd_read not allowed")
}

func TestSandbox_NoEnvAccess(t *testing.T) {
    out := runHandler(t, "tests/security/handlers/try_env_get.wasm")
    assert.Empty(t, out.EnvVarsSeen)
}

func TestSandbox_URLAllowlist(t *testing.T) {
    // handler qui try `https://evil.com/exfil`
    out := runHandler(t, "tests/security/handlers/try_external_call.wasm")
    assert.Contains(t, out.Error, "url_not_allowed")
}
```

### Refresh race

```go
func TestRefresh_ConcurrentInvocations(t *testing.T) {
    vlt := fake.NewVault()
    vlt.Store(ctx, ref, core.Credential{
        AccessToken: "expired",
        ExpiresAt:   ptr(time.Now().Add(-1 * time.Hour)),
    })

    provider := &fake.AuthProvider{
        RefreshCount: 0,
        RefreshFunc: func(c core.Credential) core.Credential {
            time.Sleep(50 * time.Millisecond)
            return core.Credential{AccessToken: "new", ExpiresAt: ptr(time.Now().Add(1 * time.Hour))}
        },
    }

    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            uc.Run(ctx, ...)
        }()
    }
    wg.Wait()

    assert.Equal(t, 1, provider.RefreshCount, "only one refresh should happen")
    finalCred, _ := vlt.Fetch(ctx, ref)
    assert.Equal(t, "new", finalCred.AccessToken.Reveal())
}
```

## Tests de performance

Critique pour un CLI : la latence et la mémoire au démarrage. CI fail si dégradation.

### Cold start benchmark

```go
// internal/cli/perf_test.go
func BenchmarkColdStart_Version(b *testing.B) {
    binary := buildBinary(b)
    for i := 0; i < b.N; i++ {
        cmd := exec.Command(binary, "--version")
        if err := cmd.Run(); err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkColdStart_DeclarativeAction(b *testing.B) {
    fakeAPI := startFakeAPI(b)
    binary := buildBinary(b)
    setupProject(b)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        cmd := exec.Command(binary, "github", "issues.read", "--issue", "1")
        cmd.Env = []string{"ONE_GITHUB_API_BASE=" + fakeAPI.URL}
        if err := cmd.Run(); err != nil {
            b.Fatal(err)
        }
    }
}
```

### Budgets

Définis dans `.benchmarks.json` :

```json
{
  "BenchmarkColdStart_Version": { "max_ns_per_op": 30000000 },
  "BenchmarkColdStart_DeclarativeAction": { "max_ns_per_op": 50000000 },
  "BenchmarkColdStart_WASMHandler": { "max_ns_per_op": 80000000 }
}
```

CI exécute les benchmarks, parse le JSON output, compare. Si dégradation >10% : warning. Si dépassement du max absolu : fail.

### Memory footprint

```go
func TestMemoryAtStartup(t *testing.T) {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    initial := m.Alloc

    // Setup standard
    setupFullBinary(t)

    runtime.ReadMemStats(&m)
    delta := m.Alloc - initial
    assert.Less(t, delta, uint64(30*1024*1024), "startup should use less than 30MB")
}
```

## Snapshot tests

Pour les outputs structurés (capabilities JSON, info markdown), snapshot tests avec [go-snaps](https://github.com/gkampitakis/go-snaps).

```go
func TestCapabilities_Output_Snapshot(t *testing.T) {
    catalog := fake.NewCatalog(fixtures.MinimalCatalog())
    out := buildCapabilities(t, catalog, scope)
    snaps.MatchJSON(t, out)
}
```

Premier run : sauvegarde le snapshot. Runs suivants : compare. Diff visible → update via flag `-update`.

Garde tes outputs stables sans écrire des assertions champ par champ. Idéal pour les sorties JSON qui changent peu mais où chaque champ compte.

## Fixtures

Centralisées dans `internal/testing/fixture/` :

```
internal/testing/fixture/
├── catalog/
│   ├── v1-minimal/          # 2-3 services basiques
│   │   ├── github/
│   │   ├── notion/
│   │   └── _index.json
│   ├── v1-full/             # tous les services + handlers WASM mock
│   └── v1-broken/           # cas dégénérés pour tester le validateur
├── scopes/
│   ├── empty.yaml
│   ├── readonly.yaml
│   ├── full.yaml
│   └── with-deny.yaml
└── projects/
    └── ok/
        ├── .onerc.yaml
        └── .onerc.lock
```

Chargement via helpers :

```go
func FixtureCatalog(t *testing.T, name string) string {
    t.Helper()
    path := filepath.Join("internal/testing/fixture/catalog", name)
    if _, err := os.Stat(path); err != nil {
        t.Fatalf("fixture %q not found: %v", name, err)
    }
    return path
}
```

## Tests des install guides

Les guides sont des markdown avec frontmatter. Validation automatique en CI catalog :

```go
func TestGuide_FrontmatterValid(t *testing.T) {
    for _, g := range walkGuides("catalog/services") {
        t.Run(g.Service+"/"+g.ID, func(t *testing.T) {
            fm, err := ParseFrontmatter(g.Content)
            require.NoError(t, err)
            require.NotEmpty(t, fm.ID)
            require.NotEmpty(t, fm.Title)

            if fm.Verify != nil {
                actionExists := catalog.HasAction(g.Service, fm.Verify.Action)
                assert.True(t, actionExists, "verify references unknown action")
            }
        })
    }
}
```

## Tests UI/UX (TTY)

Pour le renderer TTY, snapshot des outputs colorés (avec `lipgloss`) :

```go
func TestRenderer_TTY_ExecuteSuccess(t *testing.T) {
    var buf bytes.Buffer
    r := renderer.NewTTY(&buf, &buf)
    r.RenderResult(app.ExecuteOutput{
        Result:  json.RawMessage(`{"id":42}`),
        TraceID: "01HXYZ",
    })
    snaps.MatchSnapshot(t, buf.String())
}
```

Capture l'ANSI escape codes. Diff visible si on change un emoji ou une couleur.

## Tests des erreurs et hints

```go
func TestSetupRequired_RendersHintWithGuide(t *testing.T) {
    err := core.ErrSetupRequired{
        Service: "notion",
        Guide:   "share-page",
        Reason:  "Page not accessible",
        Human:   true,
    }

    var buf bytes.Buffer
    r := renderer.NewJSON(&buf, &buf)
    r.RenderError(err)

    var parsed map[string]interface{}
    json.Unmarshal(buf.Bytes(), &parsed)
    assert.Equal(t, "share-page", parsed["error"].(map[string]interface{})["install"].(map[string]interface{})["guide"])
    assert.Equal(t, "one install notion share-page", parsed["error"].(map[string]interface{})["install"].(map[string]interface{})["command"])
}
```

## Tests de la sortie JSON

Le format JSON émis par le binaire est une API publique pour les agents. Tout changement est breaking.

```go
func TestJSONOutput_Schema(t *testing.T) {
    schema, _ := ioutil.ReadFile("schemas/execute-output-v1.json")
    output := runExecution(t)

    err := jsonschema.Validate(schema, output)
    require.NoError(t, err, "output must match the v1 schema")
}
```

Schéma maintenu dans `schemas/`, versionné, jamais cassé entre minors.

## Tests des handlers WASM (côté binaire)

Pas les tests des handlers eux-mêmes (ça c'est côté catalogue), mais les tests du **runtime** qui les exécute.

### Fixtures de handlers

```
internal/testing/fixture/handlers/
├── echo.wasm                # retourne ses inputs
├── http_get.wasm            # fait un GET sur une URL fixée
├── http_evil.wasm           # tente un GET hors allowlist
├── creds_get.wasm           # tente host.creds.get
├── fail.wasm                # appelle host.fail.withCode
├── timeout.wasm             # boucle infinie (test timeout)
└── memory_hog.wasm          # alloue 200MB (test OOM)
```

Compilés en CI une fois, pas commités (build cache).

```go
func TestRuntime_WASM_Echo(t *testing.T) {
    rt := runtime.NewWazero(fakeHTTP, fakeCrypto, fakeClock, log)
    out, err := rt.Execute(ctx, ports.ExecuteRequest{
        Action: actionWithHandler("echo.wasm"),
        Inputs: core.Inputs{"hello": "world"},
    })
    require.NoError(t, err)
    assert.JSONEq(t, `{"hello":"world"}`, string(out.Output))
}

func TestRuntime_WASM_URLAllowlist(t *testing.T) {
    rt := runtime.NewWazero(fakeHTTP, fakeCrypto, fakeClock, log)
    _, err := rt.Execute(ctx, ports.ExecuteRequest{
        Action: actionWithHandlerAndCalls("http_evil.wasm", []string{"https://api.allowed.com/*"}),
    })
    assert.ErrorContains(t, err, "url_not_allowed")
}
```

## Coverage

`go test -coverprofile=coverage.out ./...` en CI. Objectifs :

- `internal/core/` : >85%
- `internal/app/` : >75%
- `internal/adapters/` : >70%
- `internal/cli/` : >60% (CLI a beaucoup de boilerplate, ok pour moins)
- Global : >70%

Coverage en dessous des objectifs : CI warning, pas fail. Le but est de garder la confiance, pas de jouer à un jeu de chiffres.

## Organisation des tests

### Conventions de nommage

- `Test<Type>_<Method>_<Condition>` : `TestScope_Allows_ExactMatch`, `TestExecute_NotInScope`
- `Benchmark<Operation>` : `BenchmarkColdStart_Version`
- `TestE2E_<Scenario>` : `TestE2E_GitHub_IssuesList`
- Helpers : `func helperName(t *testing.T, ...)` avec `t.Helper()`

### Setup et teardown

Préférer `t.TempDir()` pour les fichiers temporaires (cleanup automatique). Préférer `t.Cleanup(func)` pour le teardown explicite (s'exécute même en cas de panic).

### Pas de tests parallèles tant que pas nécessaire

`t.Parallel()` est tentant mais ajoute de la complexité. Le binaire est assez rapide pour que la suite tourne en <30s même séquentielle. Activer `t.Parallel()` uniquement pour les tests E2E lents (>1s).

## Anti-patterns

### Mocks à 50 expectations

```go
mockVault.EXPECT().Fetch(gomock.Any(), gomock.Eq(ref1)).Return(cred1, nil)
mockVault.EXPECT().Store(gomock.Any(), gomock.Eq(ref1), gomock.AssignableToTypeOf(core.Credential{})).Return(nil)
// ... 48 lignes
```

Casse à chaque refactor. Préfère un fake `Vault` in-memory réutilisable.

### Sleeps pour synchroniser

```go
go startBackground()
time.Sleep(100 * time.Millisecond)  // attend que ça démarre
```

Flaky en CI. Utilise des channels ou des callbacks.

### Tests qui n'assertent rien

```go
func TestFooDoesNotPanic(t *testing.T) {
    foo()  // si ça crash, le test fail
}
```

Acceptable si l'objectif est "le code compile et exécute", mais la plupart du temps tu veux assert l'output.

### Tests dépendants de l'ordre

```go
func TestStep1_CreatesFile(t *testing.T) { ... }
func TestStep2_ReadsFile(t *testing.T) { ... }  // dépend que Step1 ait tourné
```

Chaque test doit être indépendant. Setup ce dont tu as besoin, cleanup à la fin.

### Tests qui vont sur Internet

Sauf E2E explicites, **aucun test ne doit faire d'appel réseau réel**. Toujours mocker via fakeAPI local. Sinon les tests fail offline et en CI sans accès Internet.

### Tests avec dépendance de temps réel

```go
time.Sleep(1 * time.Second)
assert.True(t, cred.NeedsRefresh(time.Now()))
```

Utilise `fake.Clock` pour contrôler le temps déterministiquement.

---

*Pour exécuter la suite complète : `go test ./...`. Pour les tests sécurité : `go test -tags=security ./tests/security/...`. Pour les benchmarks : `go test -bench=. ./internal/cli/`. CI fait tourner les trois sur chaque PR.*
