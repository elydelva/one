# CONTRIBUTING.md

> Guide for contributing to the One CLI project. If you're contributing to the catalogue (adding a service), go directly to [CATALOG.md](./CATALOG.md). This document covers contributions to the **Go binary**.

## Initial setup

### Prerequisites

- **Go 1.23+** : `go version`
- **Git** : `git --version`
- **make** : or rebuild directly via `go build`
- **golangci-lint** : for linting (`brew install golangci-lint`)
- **tinygo** : only if you touch Go handler tests (`brew install tinygo`)
- **bun** or **node** : only if you touch TypeScript handler tests

Optional:

- **wasmtime** or **wasmer** CLI : for debugging WASM modules
- **age** : for testing the encrypted vault

### Clone and build

```bash
git clone https://github.com/one-cli/one
cd one
make build                            # produces ./bin/one
./bin/one --version                   # verify
```

Or without make:

```bash
go build -o bin/one ./cmd/one
```

### Running the tests

```bash
make test                             # unit + integration + contract
make test-security                    # security tests (long, ~30s)
make test-e2e                         # E2E tests (slow, ~2min)
make bench                            # benchmarks
make lint                             # golangci-lint
```

Or without make:

```bash
go test ./...
go test -tags=security ./tests/security/...
go test -tags=e2e ./tests/e2e/...
go test -bench=. ./...
golangci-lint run
```

### Running in dev

```bash
# Build and use locally
make install                          # puts the binary in $GOBIN
which one                             # verify

# Or directly
go run ./cmd/one -- --version
```

## Contribution workflow

### 1. Find an issue

Three good ways to get started:

- **Issues tagged `good-first-issue`**: designed to explore the code
- **Issues tagged `help-wanted`**: real need for help, medium scope
- **Open RFCs**: feature under discussion, your input is welcome

Before coding, **comment on the issue to signal your interest**. Avoid duplicating work.

### 2. Fork and branch

```bash
# Fork via GitHub UI, then:
git clone https://github.com/<you>/one
cd one
git remote add upstream https://github.com/one-cli/one
git checkout -b feat/add-bitbucket-provider
```

**Branch name**: `<type>/<short-description>`. Types: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `chore`.

### 3. Code

See the sections below for code style and conventions.

### 4. Test

```bash
make test
make lint
```

CI runs both. If it passes locally, it passes in CI (except cross-platform cases).

### 5. Commit

Format: **conventional commits**.

```
feat(auth): add bitbucket OAuth provider

Implements the OAuth 2.0 user-flow for Bitbucket Cloud, including
PKCE and refresh token rotation. Tested via fakes against the
ports.AuthProvider contract.

Closes #142
```

Accepted types: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `chore`, `ci`, `build`. Scope is optional but recommended: `auth`, `catalog`, `vault`, `runtime`, `cli`, `core`, etc.

**No "WIP" commits** in the final PR (rebase to squash if you have any).

### 6. Push and PR

```bash
git push origin feat/add-bitbucket-provider
```

Open the PR via the GitHub UI. The pre-filled template asks for:

- Description of the change
- Linked issue
- Tests added
- Breaking changes (if applicable)
- Screenshots (if UI/TTY changed)

### 7. Review

A maintainer will review within 7 days. Possible iterations:

- Change requests: push to the same branch, no need to reopen the PR
- Design discussions: resolved in the PR, or a RFC is opened if too broad

### 8. Merge

Merged via **squash and merge** by default. Your commit(s) become a single commit on main. The final message is edited by the maintainer to follow conventional commits.

Once merged, your name appears in the `CHANGELOG.md` of the next release.

## Code style

### Go: standard conventions

We follow [Effective Go](https://go.dev/doc/effective_go) and [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments). A few specific points:

#### Naming

- Exported types: `PascalCase` (`Credential`, `ServiceID`)
- Unexported types: `camelCase` (`vaultState`)
- Constructor functions: `New<Type>(...)` (`NewScope`, `NewCachedCatalog`)
- Interfaces: name without `I` prefix, rather with a descriptive suffix (`Catalog`, `AuthProvider`, not `ICatalog`)
- Short variables for short scope (`ctx`, `err`, `i`), explicit for long scope (`servicesByName`)

#### Imports

Ordered in 3 blocks separated by a blank line:

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/zalando/go-keyring"
    "golang.org/x/oauth2"

    "one/internal/core"
    "one/internal/ports"
)
```

Stdlib | external | local.

#### Errors

- Wrap with context: `fmt.Errorf("fetch credential: %w", err)`
- No "failed to" prefix: `"fetch credential"` is enough
- No capitalisation of the first character of the message
- No trailing period
- Sentinel error types defined in `core/errors.go`, never scattered

#### Comments

- All exported functions/types have a Go doc comment
- Format: `// FuncName does X. Returns Y when Z.`
- No comments for obvious cases (`// increment i`)
- Prefer clear names over explanatory comments

#### Context

- Always first argument: `func Foo(ctx context.Context, ...)`
- Never stored in a struct
- Propagated everywhere, even when "not used right now"

#### Concurrency

- Prefer channels over mutexes when possible
- If mutex, document the invariant it protects
- Always `defer mu.Unlock()` after `mu.Lock()`
- No `time.Sleep` to synchronise; use channels or waitgroups

### Repo layout

See [ARCHITECTURE.md](./ARCHITECTURE.md#layout-du-repo) for the full layout. Golden rule:

- **Domain** in `internal/core/`: zero external dependencies
- **Ports** in `internal/ports/`: interfaces only
- **Adapters** in `internal/adapters/<port>/`: implementations
- **Use cases** in `internal/app/`: orchestration
- **CLI** in `internal/cli/`: UI adapter (cobra)

If your PR puts HTTP code in `core/`, immediate rejection.

### Recurring patterns

#### Constructors

Explicit constructors, no "init magic":

```go
// GOOD
func NewExecuteAction(
    catalog ports.Catalog,
    vault ports.Vault,
    runtime ports.Runtime,
    log ports.Logger,
    clock ports.Clock,
) *ExecuteAction {
    return &ExecuteAction{...}
}

// NOT GOOD
type ExecuteAction struct { ... }
func (e *ExecuteAction) Init(...) { ... }
```

#### Options patterns for complex structs

```go
type Server struct { ... }

type ServerOption func(*Server)

func WithTimeout(d time.Duration) ServerOption {
    return func(s *Server) { s.timeout = d }
}

func NewServer(opts ...ServerOption) *Server {
    s := &Server{timeout: 30 * time.Second}  // defaults
    for _, opt := range opts { opt(s) }
    return s
}
```

#### Interface segregation

One interface = one responsibility. If you need 3 methods, consider making 2-3 interfaces and composing them.

```go
// GOOD
type CatalogReader interface {
    GetService(ctx, ServiceID) (*Service, error)
    ListServices(ctx) ([]Service, error)
}

type CatalogWriter interface {
    UpdateService(ctx, Service) error
}

type Catalog interface {
    CatalogReader
    CatalogWriter
}
```

## Dos and don'ts

### Do

- **Write tests for every change.** Appropriate level: unit for business logic, contract for adapters, integration for use cases.
- **Document exported functions.** If they're exported, they're being used; doc + doc test ideally.
- **Update docs in parallel with code.** If you change a behaviour described in CLI.md, update CLI.md in the same PR.
- **Take advantage of existing fakes.** `internal/testing/fake/` already has what you need for 80% of cases.
- **Prefer adding over modifying.** If you can add an adapter without touching the domain, that's better.
- **Discuss before coding for large changes.** Open an issue or a RFC.

### Don't

- **No giant PRs.** One PR = one focused change. If you're touching 30 files, split it up.
- **No silent breaking changes.** If you break a public API, mention it in the PR title and propose a migration path.
- **No "fix unrelated typo in passing".** If you see a typo, open a dedicated PR. It makes review easier.
- **No dependency added without discussion.** Every new external dependency has a cost. If it's justified, mention it explicitly in the PR.
- **No magic.** No custom code generation, no excessive reflection, no panic in application code.

## Adding a new external dependency

Criteria before adding a package:

1. **Not available in stdlib**: the Go stdlib is rich, check there first.
2. **Actively maintained**: last commit < 6 months, or if abandoned, justify why that's OK.
3. **Compatible license**: MIT, Apache 2.0, BSD-3, MPL-2.0. Not GPL, not custom.
4. **No explosive transitive dependencies**: check `go mod graph`.
5. **Minimal API surface**: if you only use one function out of 50, consider writing it yourself.

Indicative list of acceptable dependencies at the time of writing:

- `github.com/spf13/cobra` : CLI (already in place)
- `github.com/spf13/viper` : config (already in place)
- `github.com/zalando/go-keyring` : keychain (already in place)
- `github.com/tetratelabs/wazero` : WASM runtime (already in place)
- `filippo.io/age` : encrypted vault (already in place)
- `github.com/goccy/go-yaml` : YAML parser (already in place)
- `github.com/santhosh-tekuri/jsonschema/v6` : JSON Schema (already in place)
- `golang.org/x/oauth2` : OAuth helpers (already in place)
- `github.com/charmbracelet/lipgloss` : TTY styling (already in place)
- `github.com/charmbracelet/bubbletea` : interactive TUI (already in place for interactive flows)

If you want to add another one, motivate it in the PR.

## RFC: for large changes

If you want to change:

- The `service.yaml` API
- The scope file grammar
- The WASM host functions contract
- A structuring naming convention
- A security mechanism

Open a RFC in the `one-cli/rfcs` repo. Format in `rfcs/0000-template.md`.

Process:

1. Fork `rfcs`, copy the template, rename `0000-` to a free number.
2. Edit, push, open a PR.
3. Open discussion for a minimum of 14 days.
4. Decision: accept, defer, reject. Documented by the maintainer.
5. If accepted, implementation follows normally via a PR on the relevant repo.

## Security

If you find a vulnerability, **do not open a public issue**. See [SECURITY.md > Disclosure policy](./SECURITY.md#disclosure-policy).

## License

By opening a PR, you agree that your code is licensed under the same license as the project (Apache 2.0 or MIT, see LICENSE). No CLA, implicit sign-off via merge.

## Community

- **GitHub Discussions**: general questions, design ideas, usage feedback
- **GitHub Issues**: concrete bugs and features
- **Discord** (coming soon): real-time chat

Code of conduct: standard, [Contributor Covenant 2.1](https://www.contributor-covenant.org/version/2/1/code_of_conduct/). In short: be respectful, careful in disagreements, focused on the project.

## Rewards

The project is open source, no direct payment. But:

- Your name in `CHANGELOG.md` and `CONTRIBUTORS.md`
- Public recognition in release announcements
- Mentorship: if you're new to Go or open source, maintainers take the time to help you in review
- For regular contributors: commit access possible after 3-5 quality PRs

## Suggested first issue

If you want to contribute but don't know where to start:

1. **Read `DESIGN.md` and `ARCHITECTURE.md`**: 20 minutes for the overview.
2. **Run `make build && make test`**: make sure your setup works.
3. **Pick a `good-first-issue`**: usually adding a test, fixing an error message, improving a guide.
4. **Follow the workflow above**: no surprises.

Welcome!

---

*Maintained by [@elydelva](https://github.com/elydelva) and the community. Any suggestion to improve this document is welcome via PR.*
