# TESTING.md

> Testing strategy for the `one` binary. For tests of WASM handlers contributed to the catalog, see [HANDLERS.md](./HANDLERS.md#tests). For specific security tests, see [SECURITY.md](./SECURITY.md#tests-de-sécurité).

## Overview

Four levels of tests, in a pyramid:

```
            ┌─────────────────┐
            │   E2E (~5%)     │   compiled binary + a real catalog
            ├─────────────────┤
            │ Integration     │
            │    (~10%)       │   multiple adapters wired together
            ├─────────────────┤
            │  Contract       │
            │    (~15%)       │   one adapter against port contracts
            ├─────────────────┤
            │                 │
            │  Unit (~70%)    │   domain and pure logic
            │                 │
            └─────────────────┘
```

**Numeric target**: 80%+ coverage on `internal/core/`, 70%+ on `internal/app/`, sufficient to maintain confidence elsewhere.

**Guiding principle**: for each line of code, ask yourself "at what level should this line be tested". The majority should be tested at the lowest possible level.

## Level 1: unit tests

Pure tests of domain and application logic. No I/O, no network, no filesystem.

### Pattern: table-driven

Go has a strong convention for testing pure functions. Always table-driven, never one assertion per function.

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
        // ... 20+ cases
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

**Rule**: if logic has 5+ branches, use table-driven. If 1-2 branches, separate simple functions are fine.

### Construction helpers

To reduce noise:

```go
func mustPerm(t *testing.T, svc, path string) Permission {
    t.Helper()
    p, err := NewPermission(ServiceID(svc), path)
    if err != nil { t.Fatal(err) }
    return p
}
```

`t.Helper()` tells the framework that errors come from the caller. Comfortable.

### Property-based tests for parsers

For parsers (YAML scope, glob patterns): property-based with [gopter](https://github.com/leanovate/gopter).

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

Indirect coverage of cases you would not have thought to write by hand.

### Testing typed errors

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

Verify the **type** of error, not the message. The message can change; the type is the API.

## Level 2: contract tests

A critical pattern in hexagonal architecture: a single test package executed against **each implementation of a port**.

### Structure

```
internal/testing/portstest/
├── catalog.go            # defines RunCatalogTests(t, factory func() ports.Catalog)
├── vault.go              # defines RunVaultTests(t, factory func() ports.Vault)
├── runtime.go
├── scopestore.go
└── ...
```

### Example: contract tests for Catalog

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

        // ... 15-20 contract cases
    })
}
```

### Application

Each adapter for the `Catalog` port runs these tests:

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

**Benefit**: you can add a new Catalog adapter (e.g. `CatalogGitTagged` that fetches from Git tags), run the contract tests, and know that the rest of the system will work with it.

### Fakes vs mocks: strict rule

**Strong preference for fakes** over mocks. A fake is an in-memory implementation of a port. A mock verifies calls.

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

**Why**:

- The fake runs contract tests, so its behavior is guaranteed correct.
- No mock framework breaking at every signature refactor.
- Readable, debuggable.
- Reusable across tests.

Mocks are only acceptable to verify side effects that are hard to observe directly (e.g. "was this port called with exactly these arguments?"). Even then, prefer a fake that records calls via `RecordedCalls()`.

## Level 3: integration tests

Multiple adapters wired together, but without real network. Tests in `internal/app/*_test.go`.

### Example: test for the Execute use case

```go
// internal/app/execute_test.go
func TestExecuteAction_HappyPath(t *testing.T) {
    // Wiring
    catalog := fake.NewCatalog(fixtures.MinimalCatalog())
    vlt := fake.NewVault()
    rt := fake.NewRuntime()
    scope := fake.NewScopeStore()
    clk := fake.NewClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
    log := slog.New(slog.NewTextHandler(io.Discard, nil))

    // State
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

    // Execution
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
    // ... same setup but empty scope
    _, err := uc.Run(ctx, app.ExecuteInput{...})

    var nsc core.ErrNotInScope
    assert.ErrorAs(t, err, &nsc)
}
```

### Use case tests: case matrix

For `ExecuteAction`, the minimum matrix:

| Case | Expected |
|---|---|
| Unknown service | ErrUnknownService |
| Unknown action | ErrUnknownAction |
| Not authenticated | ErrNotAuthenticated |
| Not in scope | ErrNotInScope |
| Invalid inputs | ErrInputValidation |
| Expired token, refresh OK | success |
| Expired token, refresh KO | ErrReAuthRequired |
| Declarative action OK | success |
| WASM action OK | success |
| Setup required, action fails | ErrSetupRequired with hint |
| Dry-run action | success, no side effect |
| Trace ID propagated | TraceID in output |
| Audit log written | log contains the event |

13 cases. All tested via fakes. <1s total.

## Level 4: E2E tests

The compiled binary against a real catalog. But **without network**: a local HTTP server mocks the APIs.

### Structure

```
tests/e2e/
├── e2e_test.go            # Go tests that invoke the binary
├── fixtures/
│   ├── catalog/           # a complete minimal catalog
│   │   └── services/
│   │       └── ...
│   └── projects/
│       ├── ok/
│       │   └── .onerc.yaml
│       └── ...
└── fake_api/              # HTTP server that mimics GitHub, Notion, etc.
    ├── server.go
    ├── github.go
    └── ...
```

### Example

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

### Cases covered

5-10 E2E tests maximum, focused on:

- Happy path for a critical command (`one <service> <action>`)
- Expected exit codes (0, 1, 2, 3, 4, 5)
- Parseable JSON output
- Readable TTY output (snapshot)
- `one capabilities` returns valid JSON
- `one info` returns valid markdown
- Complete workflow: init → login → scope add → exec

No more than that. E2E tests are slow and brittle; keep them minimal.

## Cross-platform tests

CI matrix on three platforms: Linux, macOS, Windows.

```yaml
# .github/workflows/test.yml
strategy:
  matrix:
    os: [ubuntu-latest, macos-latest, windows-latest]
    go: ['1.23']
```

**OS-specific cases**:

| Test | Why |
|---|---|
| Keychain store/fetch | Different API per OS (Keychain/Secret Service/CredMgr) |
| File locks | `flock` Linux/macOS vs `LockFileEx` Windows |
| Path resolution | `/` vs `\`, home dir, XDG |
| TTY detection | `isatty` behavior |
| Exec browser | `open` vs `xdg-open` vs `start` |
| File permissions | `0600` on Unix, ACL on Windows |

Windows keychain in CI is non-trivial (requires an interactive session). Solution: `--short` tests that skip tests requiring a real keychain, plus a weekly manual test.

## Security tests

Dedicated suite tagged `security`, to be run in CI on every push:

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
    // handler that tries `https://evil.com/exfil`
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

## Performance tests

Critical for a CLI: startup latency and memory. CI fails on regression.

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

Defined in `.benchmarks.json`:

```json
{
  "BenchmarkColdStart_Version": { "max_ns_per_op": 30000000 },
  "BenchmarkColdStart_DeclarativeAction": { "max_ns_per_op": 50000000 },
  "BenchmarkColdStart_WASMHandler": { "max_ns_per_op": 80000000 }
}
```

CI runs benchmarks, parses the JSON output, and compares. If regression >10%: warning. If absolute max exceeded: fail.

### Memory footprint

```go
func TestMemoryAtStartup(t *testing.T) {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    initial := m.Alloc

    // Standard setup
    setupFullBinary(t)

    runtime.ReadMemStats(&m)
    delta := m.Alloc - initial
    assert.Less(t, delta, uint64(30*1024*1024), "startup should use less than 30MB")
}
```

## Snapshot tests

For structured outputs (capabilities JSON, info markdown), snapshot tests with [go-snaps](https://github.com/gkampitakis/go-snaps).

```go
func TestCapabilities_Output_Snapshot(t *testing.T) {
    catalog := fake.NewCatalog(fixtures.MinimalCatalog())
    out := buildCapabilities(t, catalog, scope)
    snaps.MatchJSON(t, out)
}
```

First run: saves the snapshot. Subsequent runs: compare. Visible diff → update via flag `-update`.

Keeps your outputs stable without writing field-by-field assertions. Ideal for JSON outputs that change rarely but where every field matters.

## Fixtures

Centralized in `internal/testing/fixture/`:

```
internal/testing/fixture/
├── catalog/
│   ├── v1-minimal/          # 2-3 basic services
│   │   ├── github/
│   │   ├── notion/
│   │   └── _index.json
│   ├── v1-full/             # all services + mock WASM handlers
│   └── v1-broken/           # degenerate cases for testing the validator
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

Loading via helpers:

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

## Install guide tests

Guides are markdown files with frontmatter. Automatically validated in the catalog CI:

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

## UI/UX tests (TTY)

For the TTY renderer, snapshot colored outputs (with `lipgloss`):

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

Captures ANSI escape codes. Visible diff if you change an emoji or a color.

## Error and hint tests

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

## JSON output tests

The JSON format emitted by the binary is a public API for agents. Any change is breaking.

```go
func TestJSONOutput_Schema(t *testing.T) {
    schema, _ := ioutil.ReadFile("schemas/execute-output-v1.json")
    output := runExecution(t)

    err := jsonschema.Validate(schema, output)
    require.NoError(t, err, "output must match the v1 schema")
}
```

Schema maintained in `schemas/`, versioned, never broken between minors.

## WASM handler tests (binary side)

Not the tests of the handlers themselves (that's on the catalog side), but the tests of the **runtime** that executes them.

### Handler fixtures

```
internal/testing/fixture/handlers/
├── echo.wasm                # returns its inputs
├── http_get.wasm            # makes a GET to a fixed URL
├── http_evil.wasm           # attempts a GET outside the allowlist
├── creds_get.wasm           # attempts host.creds.get
├── fail.wasm                # calls host.fail.withCode
├── timeout.wasm             # infinite loop (timeout test)
└── memory_hog.wasm          # allocates 200MB (OOM test)
```

Compiled in CI once, not committed (build cache).

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

`go test -coverprofile=coverage.out ./...` in CI. Targets:

- `internal/core/`: >85%
- `internal/app/`: >75%
- `internal/adapters/`: >70%
- `internal/cli/`: >60% (CLI has a lot of boilerplate, acceptable to be lower)
- Global: >70%

Coverage below targets: CI warning, not fail. The goal is to maintain confidence, not to play a numbers game.

## Test organization

### Naming conventions

- `Test<Type>_<Method>_<Condition>`: `TestScope_Allows_ExactMatch`, `TestExecute_NotInScope`
- `Benchmark<Operation>`: `BenchmarkColdStart_Version`
- `TestE2E_<Scenario>`: `TestE2E_GitHub_IssuesList`
- Helpers: `func helperName(t *testing.T, ...)` with `t.Helper()`

### Setup and teardown

Prefer `t.TempDir()` for temporary files (automatic cleanup). Prefer `t.Cleanup(func)` for explicit teardown (runs even on panic).

### No parallel tests unless necessary

`t.Parallel()` is tempting but adds complexity. The binary is fast enough for the suite to run in <30s even sequentially. Enable `t.Parallel()` only for slow E2E tests (>1s).

## Anti-patterns

### Mocks with 50 expectations

```go
mockVault.EXPECT().Fetch(gomock.Any(), gomock.Eq(ref1)).Return(cred1, nil)
mockVault.EXPECT().Store(gomock.Any(), gomock.Eq(ref1), gomock.AssignableToTypeOf(core.Credential{})).Return(nil)
// ... 48 lines
```

Breaks at every refactor. Prefer a reusable in-memory `Vault` fake.

### Sleeps for synchronization

```go
go startBackground()
time.Sleep(100 * time.Millisecond)  // wait for it to start
```

Flaky in CI. Use channels or callbacks.

### Tests that assert nothing

```go
func TestFooDoesNotPanic(t *testing.T) {
    foo()  // if it crashes, the test fails
}
```

Acceptable if the goal is "the code compiles and runs", but most of the time you want to assert the output.

### Order-dependent tests

```go
func TestStep1_CreatesFile(t *testing.T) { ... }
func TestStep2_ReadsFile(t *testing.T) { ... }  // depends on Step1 having run
```

Each test must be independent. Set up what you need, clean up at the end.

### Tests that go to the Internet

Except for explicit E2E tests, **no test should make real network calls**. Always mock via a local fakeAPI. Otherwise tests fail offline and in CI without Internet access.

### Tests with real-time dependency

```go
time.Sleep(1 * time.Second)
assert.True(t, cred.NeedsRefresh(time.Now()))
```

Use `fake.Clock` to control time deterministically.

---

*To run the full suite: `go test ./...`. For security tests: `go test -tags=security ./tests/security/...`. For benchmarks: `go test -bench=. ./internal/cli/`. CI runs all three on every PR.*
