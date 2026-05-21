# TOOLING.md

> Tool stack for the One CLI project: build, test, code quality, CI/CD, and contribution. Read before touching any configuration.

## Overview

```
┌─────────────────────────────────────────────────────┐
│  Local dev                                          │
│  Go 1.23+ · Make · golangci-lint · lefthook        │
├─────────────────────────────────────────────────────┤
│  Tests                                              │
│  go test · gopter · go-snaps · testify              │
├─────────────────────────────────────────────────────┤
│  WASM                                               │
│  tinygo · bun/node · wazero · wasmtime              │
├─────────────────────────────────────────────────────┤
│  CI/CD                                              │
│  GitHub Actions · codecov · govulncheck             │
├─────────────────────────────────────────────────────┤
│  Maintenance                                        │
│  renovate · goreleaser                              │
└─────────────────────────────────────────────────────┘
```

---

## Build

### Go

Minimum version: **1.23**. Check: `go version`.

Reason: support for `range` over integers (Go 1.22+), improvements to toolchain directives (Go 1.21+), and `slices`/`maps` stdlib (Go 1.21+).

### Make

All public commands go through the `Makefile`. Never memorize Go flags by hand.

| Command | Action |
|---|---|
| `make build` | Compiles `./bin/one` |
| `make install` | Build + installs into `$GOBIN` |
| `make test` | Unit + integration + contract tests |
| `make test-security` | Security suite (tags `security`) |
| `make test-e2e` | E2E suite (tags `e2e`, ~2 min) |
| `make bench` | Benchmarks + compare against `.benchmarks.json` budgets |
| `make lint` | golangci-lint run |
| `make clean` | Removes `./bin/` and build artifacts |
| `make release` | Cross-platform build + SemVer tag (CI does the actual release) |

Installing Make: pre-installed on macOS/Linux. Windows: `choco install make` or WSL.

---

## Tests

### Framework

Go stdlib `testing` only. No external test framework (no Ginkgo, no testify/suite).

Exception: assertion helpers via **testify** (`github.com/stretchr/testify`) for `assert.Equal`, `require.NoError`. Not the runner, just the assertions.

### Property-based: gopter

`github.com/leanovate/gopter` for parsers and complex structures (scope YAML, glob patterns). Used in `internal/core/` to test roundtrips and non-obvious cases.

```bash
go get github.com/leanovate/gopter
```

### Snapshots: go-snaps

`github.com/gkampitakis/go-snaps` for structured outputs (JSON capabilities, markdown info, ANSI TTY).

First run creates the snapshot. Subsequent runs compare. Update: `go test ./... -update`.

```bash
go get github.com/gkampitakis/go-snaps
```

### Coverage

Targets per package:

| Package | Threshold |
|---|---|
| `internal/core/` | >85% |
| `internal/app/` | >75% |
| `internal/adapters/` | >70% |
| `internal/cli/` | >60% |
| Global | >70% |

Below thresholds: CI warning, not fail. See `codecov.yml` for configuration.

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Quick execution

```bash
go test ./...                                    # everything except security/e2e
go test -tags=security ./tests/security/...     # security suite
go test -tags=e2e ./tests/e2e/...               # E2E suite
go test -bench=. -benchmem ./internal/cli/...   # benchmarks
go test -run TestMyFunc ./internal/core/...     # a specific test
go test -v -count=1 ./...                       # verbose, no cache
```

---

## Code quality

### golangci-lint

Version: latest stable. Install: `brew install golangci-lint`.

Config in `.golangci.yml`. Active linters:

| Linter | Role |
|---|---|
| `staticcheck` | Go static analysis (SA*, S*, QF*) |
| `errcheck` | All errors must be checked |
| `govet` | Standard `go vet` |
| `unused` | Unused symbols |
| `goimports` | Import order (stdlib / external / local) |
| `gocritic` | Idiomatic suggestions |
| `gosec` | Security vulnerabilities (G*) |
| `revive` | Style and conventions |
| `exhaustive` | Switch exhaustiveness on enum types |
| `bodyclose` | `resp.Body.Close()` required |
| `contextcheck` | `context.Context` properly propagated |
| `noctx` | No `http.Get` without context |

```bash
golangci-lint run                           # lint everything
golangci-lint run ./internal/core/...      # lint a package
golangci-lint run --fix                    # auto-fix what can be fixed
```

### govulncheck

Vulnerability scan in dependencies. Run in CI on every push to main and on PRs.

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

### gofmt / goimports

Code must be formatted. `goimports` is the superset (format + import order).

```bash
goimports -w ./...
```

CI check: `goimports -l .` — fails if output is non-empty.

---

## Git hooks: lefthook

`lefthook.yml` at the root. Install:

```bash
brew install lefthook
lefthook install
```

Configured hooks:

- **pre-commit**: `golangci-lint run --fast` + `goimports -l .`
- **commit-msg**: conventional commits validation (format `type(scope): message`)
- **pre-push**: `make test` (unit + integration, not E2E)

To bypass exceptionally (do not abuse): `git commit --no-verify`.

---

## WASM

### Runtime: wazero

`github.com/tetratelabs/wazero` is the WASM runtime embedded in the binary. No system dependency, minimal WASI sandbox. See HANDLERS.md.

### Compiling Go handlers: tinygo

To compile Go handlers → WASM:

```bash
brew install tinygo
tinygo build -o handler.wasm -target=wasi ./handler/
```

Minimum version: `0.31+`. Check: `tinygo version`.

### Compiling TypeScript handlers: bun

To compile TypeScript handlers → WASM:

```bash
brew install bun
bun build handler.ts --outfile handler.wasm
```

Alternative: `node` + `@extism/js-pdk`.

### WASM debug

To inspect a WASM module outside the One runtime:

```bash
brew install wasmtime
wasmtime --dir=. handler.wasm
```

Or `wasmer` depending on preference. Not required to contribute to the binary.

Debug environment variable: `ONE_DEBUG=1 one <service> <action>`.

---

## Go dependencies

Currently active external dependencies:

| Package | Usage |
|---|---|
| `github.com/spf13/cobra` | CLI (commands, flags, help) |
| `github.com/spf13/viper` | Config (env, files, defaults) |
| `github.com/zalando/go-keyring` | Native keychain (macOS/Linux/Windows) |
| `github.com/tetratelabs/wazero` | WASM runtime |
| `filippo.io/age` | Encrypted vault (files) |
| `github.com/goccy/go-yaml` | YAML parser |
| `github.com/santhosh-tekuri/jsonschema/v6` | JSON Schema validation |
| `golang.org/x/oauth2` | OAuth 2.0 helpers |
| `github.com/charmbracelet/lipgloss` | TTY styling |
| `github.com/charmbracelet/bubbletea` | Interactive TUI flows |
| `github.com/stretchr/testify` | Test assertions (assert/require) |
| `github.com/leanovate/gopter` | Property-based testing |
| `github.com/gkampitakis/go-snaps` | Snapshot testing |

Criteria for adding a dependency: see CONTRIBUTING.md > "Adding a new external dependency".

Dependency updates managed by **Renovate** (see `renovate.json`). Automatic PRs on minor/patch, manual review on major.

---

## CI/CD: GitHub Actions

Workflows in `.github/workflows/`:

| File | Trigger | Content |
|---|---|---|
| `ci.yml` | PR + push main | lint, test (3-OS matrix), security scan |
| `e2e.yml` | push main + schedule | E2E suite |
| `bench.yml` | push main | benchmarks + budget comparison |
| `release.yml` | push tag `v*` | goreleaser cross-platform |
| `vulncheck.yml` | weekly schedule | govulncheck |

OS matrix: `ubuntu-latest`, `macos-latest`, `windows-latest`. Go: `1.23`.

---

## Release: goreleaser

`goreleaser` manages cross-platform builds and GitHub release assets.

Config in `.goreleaser.yml`. Targets: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`.

```bash
brew install goreleaser
goreleaser build --snapshot --clean    # local build without push
make release                           # via CI, tag SemVer first
```

---

## Dependency updates: Renovate

`renovate.json` at the root. Renovate opens automatic PRs for:

- Go dependencies (`go.mod`)
- GitHub Actions (action versions)
- CLI tools (golangci-lint, goreleaser, tinygo)

Strategy: automerge on patch, manual review on minor/major.

---

## Coverage: Codecov

`codecov.yml` at the root. Coverage uploaded after each CI run (`ubuntu-latest`). Thresholds configured to mirror TESTING.md:

- Patch coverage: >70% required (otherwise PR fails)
- Project coverage: warning if regression >2%

Badge in README.md.

---

## Dev environment variables

| Variable | Usage |
|---|---|
| `ONE_DEBUG=1` | Verbose logs + WASM trace |
| `ONE_CATALOG_DIR=<path>` | Override the catalog directory |
| `ONE_GITHUB_API_BASE=<url>` | Override the GitHub URL (E2E tests) |
| `ONE_CREDS_<SVC>_<ACCOUNT>=<json>` | Inject credentials without keychain |
| `ONE_VAULT_KEY=<hex>` | Override vault key for tests |

---

## Root configuration files

| File | Tool | Role |
|---|---|---|
| `Makefile` | Make | All build/test commands |
| `.golangci.yml` | golangci-lint | Linters and rules |
| `lefthook.yml` | lefthook | Git hooks |
| `renovate.json` | Renovate | Dependency updates |
| `codecov.yml` | Codecov | Coverage thresholds |
| `.goreleaser.yml` | goreleaser | Cross-platform build |
| `.benchmarks.json` | CI bench | Performance budgets |
| `schemas/` | jsonschema | Public JSON output schemas |

---

*Keep this document in sync with the config files. If you add a tool, document it here.*
