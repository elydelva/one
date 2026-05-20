# TOOLING.md

> Stack d'outils du projet One CLI : build, test, qualité de code, CI/CD, et contribution. À lire avant de toucher à la configuration.

## Vue d'ensemble

```
┌─────────────────────────────────────────────────────┐
│  Dev local                                          │
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

Version minimale : **1.23**. Vérifier : `go version`.

Raison : support des `range` sur les entiers (Go 1.22+), améliorations des toolchain directives (Go 1.21+), et `slices`/`maps` stdlib (Go 1.21+).

### Make

Toutes les commandes publiques passent par le `Makefile`. Ne jamais mémoriser les flags Go à la main.

| Commande | Action |
|---|---|
| `make build` | Compile `./bin/one` |
| `make install` | Build + installe dans `$GOBIN` |
| `make test` | Unit + integration + contract tests |
| `make test-security` | Suite sécurité (tags `security`) |
| `make test-e2e` | Suite E2E (tags `e2e`, ~2 min) |
| `make bench` | Benchmarks + compare aux budgets `.benchmarks.json` |
| `make lint` | golangci-lint run |
| `make clean` | Supprime `./bin/` et les artefacts de build |
| `make release` | Build cross-platform + tag SemVer (CI fait la release) |

Installation Make : préinstallé sur macOS/Linux. Windows : `choco install make` ou WSL.

---

## Tests

### Framework

Go stdlib `testing` uniquement. Pas de framework externe de tests (pas de Ginkgo, pas de testify/suite).

Exception : helpers d'assertion via **testify** (`github.com/stretchr/testify`) pour `assert.Equal`, `require.NoError`. Pas le runner, juste les assertions.

### Property-based : gopter

`github.com/leanovate/gopter` pour les parsers et les structures complexes (scope YAML, glob patterns). Utilisé dans `internal/core/` pour tester les roundtrips et les cas non évidents.

```bash
go get github.com/leanovate/gopter
```

### Snapshots : go-snaps

`github.com/gkampitakis/go-snaps` pour les sorties structurées (JSON capabilities, markdown info, ANSI TTY).

Premier run crée le snapshot. Runs suivants comparent. Mise à jour : `go test ./... -update`.

```bash
go get github.com/gkampitakis/go-snaps
```

### Coverage

Objectifs par package :

| Package | Seuil |
|---|---|
| `internal/core/` | >85% |
| `internal/app/` | >75% |
| `internal/adapters/` | >70% |
| `internal/cli/` | >60% |
| Global | >70% |

En dessous des seuils : CI warning, pas fail. Voir `codecov.yml` pour la config.

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Exécution rapide

```bash
go test ./...                                    # tout sauf security/e2e
go test -tags=security ./tests/security/...     # suite sécurité
go test -tags=e2e ./tests/e2e/...               # suite E2E
go test -bench=. -benchmem ./internal/cli/...   # benchmarks
go test -run TestMyFunc ./internal/core/...     # un test précis
go test -v -count=1 ./...                       # verbose, no cache
```

---

## Qualité de code

### golangci-lint

Version : dernière stable. Installation : `brew install golangci-lint`.

Config dans `.golangci.yml`. Linters actifs :

| Linter | Rôle |
|---|---|
| `staticcheck` | Analyse statique Go (SA*, S*, QF*) |
| `errcheck` | Toutes les erreurs doivent être vérifiées |
| `govet` | `go vet` standard |
| `unused` | Symboles non utilisés |
| `goimports` | Import order (stdlib / externe / local) |
| `gocritic` | Suggestions idiomatiques |
| `gosec` | Vulnérabilités sécurité (G*) |
| `revive` | Style et conventions |
| `exhaustive` | Exhaustivité des switch sur types enum |
| `bodyclose` | `resp.Body.Close()` obligatoire |
| `contextcheck` | `context.Context` bien propagé |
| `noctx` | Pas de `http.Get` sans context |

```bash
golangci-lint run                           # lint tout
golangci-lint run ./internal/core/...      # lint un package
golangci-lint run --fix                    # auto-fix ce qui peut l'être
```

### govulncheck

Scan des vulnérabilités dans les dépendances. En CI sur chaque push main et PR.

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

### gofmt / goimports

Le code doit être formatté. `goimports` est le superset (format + import order).

```bash
goimports -w ./...
```

Vérification CI : `goimports -l .` — fail si output non vide.

---

## Git hooks : lefthook

`lefthook.yml` à la racine. Installation :

```bash
brew install lefthook
lefthook install
```

Hooks configurés :

- **pre-commit** : `golangci-lint run --fast` + `goimports -l .`
- **commit-msg** : validation conventional commits (format `type(scope): message`)
- **pre-push** : `make test` (unit + integration, pas E2E)

Pour bypasser exceptionnellement (ne pas abuser) : `git commit --no-verify`.

---

## WASM

### Runtime : wazero

`github.com/tetratelabs/wazero` est le runtime WASM embarqué dans le binaire. Pas de dépendance système, sandbox WASI minimal. Voir HANDLERS.md.

### Compilation handlers Go : tinygo

Pour compiler des handlers Go → WASM :

```bash
brew install tinygo
tinygo build -o handler.wasm -target=wasi ./handler/
```

Version minimum : `0.31+`. Vérifier : `tinygo version`.

### Compilation handlers TypeScript : bun

Pour compiler des handlers TypeScript → WASM :

```bash
brew install bun
bun build handler.ts --outfile handler.wasm
```

Alternative : `node` + `@extism/js-pdk`.

### Debug WASM

Pour inspecter un module WASM hors du runtime One :

```bash
brew install wasmtime
wasmtime --dir=. handler.wasm
```

Ou `wasmer` selon préférence. Pas requis pour contribuer au binaire.

Variable d'environnement debug : `ONE_DEBUG=1 one <service> <action>`.

---

## Dépendances Go

Liste des dépendances externes actuellement en place :

| Package | Usage |
|---|---|
| `github.com/spf13/cobra` | CLI (commands, flags, help) |
| `github.com/spf13/viper` | Config (env, fichiers, defaults) |
| `github.com/zalando/go-keyring` | Keychain natif (macOS/Linux/Windows) |
| `github.com/tetratelabs/wazero` | Runtime WASM |
| `filippo.io/age` | Vault chiffré (fichiers) |
| `github.com/goccy/go-yaml` | Parser YAML |
| `github.com/santhosh-tekuri/jsonschema/v6` | Validation JSON Schema |
| `golang.org/x/oauth2` | OAuth 2.0 helpers |
| `github.com/charmbracelet/lipgloss` | Styling TTY |
| `github.com/charmbracelet/bubbletea` | TUI flows interactifs |
| `github.com/stretchr/testify` | Assertions de tests (assert/require) |
| `github.com/leanovate/gopter` | Property-based testing |
| `github.com/gkampitakis/go-snaps` | Snapshot testing |

Critères pour ajouter une dépendance : voir CONTRIBUTING.md > "Ajouter une nouvelle dépendance externe".

Gestion des mises à jour : **Renovate** (voir `renovate.json`). PRs automatiques sur minor/patch, review manuelle sur major.

---

## CI/CD : GitHub Actions

Workflows dans `.github/workflows/` :

| Fichier | Déclencheur | Contenu |
|---|---|---|
| `ci.yml` | PR + push main | lint, test (matrix 3 OS), security scan |
| `e2e.yml` | push main + schedule | suite E2E |
| `bench.yml` | push main | benchmarks + comparaison budgets |
| `release.yml` | push tag `v*` | goreleaser cross-platform |
| `vulncheck.yml` | schedule hebdo | govulncheck |

Matrix OS : `ubuntu-latest`, `macos-latest`, `windows-latest`. Go : `1.23`.

---

## Release : goreleaser

`goreleaser` gère les builds cross-platform et les assets de release GitHub.

Config dans `.goreleaser.yml`. Targets : `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`.

```bash
brew install goreleaser
goreleaser build --snapshot --clean    # build local sans push
make release                           # via CI, tag SemVer d'abord
```

---

## Mises à jour de dépendances : Renovate

`renovate.json` à la racine. Renovate ouvre des PRs automatiques pour :

- Dépendances Go (`go.mod`)
- GitHub Actions (versions des actions)
- Outils CLI (golangci-lint, goreleaser, tinygo)

Stratégie : automerge sur patch, review manuelle sur minor/major.

---

## Coverage : Codecov

`codecov.yml` à la racine. Coverage uploadé après chaque run CI (`ubuntu-latest`). Seuils configurés en miroir de TESTING.md :

- Patch coverage : >70% requis (sinon fail PR)
- Project coverage : warning si régression >2%

Badge dans README.md.

---

## Variables d'environnement dev

| Variable | Usage |
|---|---|
| `ONE_DEBUG=1` | Logs verbeux + trace WASM |
| `ONE_CATALOG_DIR=<path>` | Override le répertoire catalog |
| `ONE_GITHUB_API_BASE=<url>` | Override l'URL GitHub (tests E2E) |
| `ONE_CREDS_<SVC>_<ACCOUNT>=<json>` | Inject credentials sans keychain |
| `ONE_VAULT_KEY=<hex>` | Override clé vault pour les tests |

---

## Fichiers de configuration à la racine

| Fichier | Outil | Rôle |
|---|---|---|
| `Makefile` | Make | Toutes les commandes de build/test |
| `.golangci.yml` | golangci-lint | Linters et règles |
| `lefthook.yml` | lefthook | Git hooks |
| `renovate.json` | Renovate | Mises à jour de dépendances |
| `codecov.yml` | Codecov | Seuils de coverage |
| `.goreleaser.yml` | goreleaser | Build cross-platform |
| `.benchmarks.json` | CI bench | Budgets de performance |
| `schemas/` | jsonschema | Schémas JSON output publics |

---

*Document à maintenir en sync avec les fichiers config. Si tu ajoutes un outil, documente-le ici.*
