# Changelog

All notable changes to One CLI are documented here.
Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versioning: [SemVer](https://semver.org/).

## [Unreleased]

## [1.0.0] — Phase 5: Polish, Trace, Skill, Release

### Added
- **TTY renderer** : `internal/adapters/renderer/tty.go` utilise lipgloss + glamour. `NO_COLOR` honoré, auto-détection TTY.
- **Bubbletea install guides** : checklists interactives navigables. `--non-interactive` pour mode texte.
- **Audit log NDJSON** : port `ports.Audit` + adapter `internal/adapters/audit/ndjson.go` (rotation mensuelle, retention 30 j, fsync). Émission auto de `LOGIN`/`LOGOUT`/`REFRESH`/`EXEC`/`ROTATE` (zéro secret dans les events).
- **`one trace`** : filtres `--since`, `--service`, `--kind`, `--trace-id`, `--limit`, `--json`.
- **`one skill [--install --ide claude-code|cursor|aider]`** : auto-détection IDE.
- **`one catalog`** : `search` / `update` / `lint` / `scaffold <id>` / `test <service>`.
- **`one upgrade --to vX.Y.Z`** : self-update avec SHA256 verify + atomic replace. Refuse `go install` path. `--dry-run`.
- **`one vault rotate`** : re-encrypt tous les credentials in-scope, audit `ROTATE` par cred.
- **Destructive prompts** : actions `side_effects: destructive` → confirmation TTY. Bypass via `--confirm`/`--dry-run`.
- **Cold-start bench CI** : `.github/workflows/bench.yml` enforce les budgets de `.benchmarks.json`.
- **E2E workflow** : `tests/e2e/workflow_test.go` couvre exec → trace → audit NDJSON.
- **Goreleaser polish** : SBOM CycloneDX, signature checksums via cosign keyless, macOS notarize gated sur secrets.
- **`install.sh` SHA256** : verify contre `checksums.txt` avant install.
- **SKILL.md binaire** : `docs/SKILL.md` + `SkillContent` inline.

### Changed
- `core.Action.SideEffects` (read | write | destructive).
- Composition root wire audit NDJSON dans toutes les use cases credentialled.
- `Login`, `Logout`, `RefreshIfNeeded`, `ExecuteAction`, `VaultRotate` gagnent `WithAudit(ports.Audit)`.

### Security
- `install.sh` vérifie SHA256 (gap critique fermé).
- Release artifacts signés via Sigstore keyless.
- SBOM publié par release.

## [0.2.0] — Phase 2: Catalog FS + Runtime déclaratif + Execute

### Added
- **Catalog FS** : `internal/adapters/catalog/fs.go` lit `service.yaml` + `actions/<id>.yaml` (action par fichier ou inline), valide version=1, transforme vers `core.Service` (BaseURL, providers, injection rules, actions avec Request/Pagination/Errors/InputSchema).
- **Core extension** : `core.Action` enrichi (`RequestSpec`, `PaginationSpec`, `ErrorSpec`), `core.Service` enrichi (`BaseURL`, `Providers`, `Injection`). Nouvelles erreurs typées : `ErrForbidden`, `ErrNotFound`, `ErrRateLimited`, `ErrAPIError`, `ErrUnsupportedRuntime`.
- **Input validator maison** : `core.InputSchema` parser + validator zéro-dep (~250 LOC), couvre type/required/pattern/enum/min/max/items + helpers `ApplyDefaults` + `ByLocation`.
- **Transport durci** : SSRF block via dialer custom (RFC1918/loopback/link-local/IMDS), cap redirects 5, refus `http://` par défaut, option pattern (`WithAllowHTTP`, `WithAllowedHosts`, `WithTimeout`).
- **DeclarativeRuntime** : exécution HTTP complète (interpolation path/query/body/headers, `body: $inputs`, auth injection via service.Injection, audit Calls, dry-run preview avec secrets redactés).
- **Pagination cursor** : loop `executePaginated` avec extraction items+token via `ResponseItems`/`ResponseToken`, safety cap `MaxPages`, concat des arrays.
- **Router** : dispatch declarative vs WASM. WASM stub Phase 3 → `ErrUnsupportedRuntime` clair.
- **HTTP error mapping** : 401→ErrNotAuthenticated, 403→ErrForbidden, 404→ErrNotFound, 429→ErrRateLimited, 5xx/4xx unmapped→ErrAPIError. Override via YAML `errors:` (hint, install_guide).
- **ExecuteAction orchestrateur** : catalog → scope → vault → refresh lazy → validate → runtime → audit, avec génération trace_id hex via Crypto port.
- **ShowInfo** : list services / service détail (+ SKILL.md) / action détail (Request + inputs).
- **CLI dispatch custom** : `one <service> <action> [--flag value ...]` route vers ExecuteAction si args[0] est dans le catalog ; sinon cobra normal. Support `--flag=value`, `--flag @file`, coercion type-aware depuis InputSchema. Renderer.RenderError appelé en sortie main.
- **Env escapes** : `ONE_CATALOG_ROOT`, `ONE_TRANSPORT_ALLOW_HTTP=1`, `ONE_TRANSPORT_ALLOWED_HOSTS=h1,h2` (dev/test only).
- **Testing** : `fakeapi` httptest helper (routes table-driven + recording), contract harness `portstest/{catalog,runtime}` enrichi, contract test enforcé via factory pattern.
- **Fixture** : `v1-minimal/github` restructuré en `service.yaml` + `actions/{issues.read,issues.list,issues.create}.yaml` couvrant GET path-param, GET cursor-paginated, POST body.
- **E2E** : `tests/e2e/execute_test.go` (build tag `e2e`) couvre happy path + 5 exit codes (0, 1 input_validation, 2 not_authenticated, 3 not_in_scope, 5 unknown_service/not_found).

### Changed
- `pkg/catalog/types.go` étendu : `ServiceDef.BaseURL`, `AuthDef.Injection`, `ActionDef.{Request,Pagination,Errors}`, `InputDef` enrichi avec Location/Pattern/Enum/Default/Min/Max/MinLen/MaxLen/Items.
- `core.Action.IsDeclarative()` reste l'unique critère de routing du Router.

### Hors-scope (déféré)
- WASM runtime : Router retourne erreur typée.
- OAuth flows, Catalog HTTP/Cached/Chain, install guides : stubs Phase 4+.
- Completion shell : cobra built-in déjà fonctionnel (`one completion bash`).

## [0.1.0] — Phase 1: Core + Vault + Scope

### Added
- **Core domain** : value objects (`ServiceID`, `AccountAlias`, `PermissionPath/Pattern`), typed errors (`ErrNotAuthenticated`, `ErrNotInScope`, `ErrReadOnly`, `ErrNotSupported`, `ErrInvalidScopeFile`, `ErrInvalidPattern`, …), `Secret` with redacting marshal + plaintext storage marshal, `Scope.AllowsWithReason`, `ParsePermissionPattern`/`ParsePermissionPath`.
- **Vault chain** : `EnvVarVault` (`ONE_CREDS_<SVC>_<ACCT>` read-only), `KeyringVault` (OS keychain via zalando/go-keyring), `ChainVault` (env → keyring fallthrough; writes routed to first writable; deletes ignore read-only).
- **Scope store** : `FileScopeStore` Load + atomic Save (.onerc.yaml, mode 0644), default-deny when file is missing, validation of `version` and patterns (refuses `**`, `?`, brace, multi-`*`).
- **Auth providers** : `TokenPasteProvider` (PAT) and `APIKeyProvider` with no-echo prompt via `golang.org/x/term` (graceful fallback for piped stdin).
- **Use cases** : `Login`, `Logout` (idempotent), `ShowScope`, `AddScope`, `RemoveScope`, `CheckScope`, `ListCapabilities` (scope-only mode), `Init` (bootstraps `.onerc.yaml` + `.gitignore`).
- **CLI commands** : `one init`, `one login <svc> [--as <alias>] [--provider pat|api_key]`, `one logout <svc> [--as <alias>]`, `one scope show|add|remove|check|explain`, `one can <svc> <action>`, `one capabilities [--scope-only]`. Persistent flag `--project <dir>`.
- **Testing** : `portstest` harness for `Vault`, `ScopeStore`, `AuthProvider`; fakes for vault/auth/clock/logger; canary leak end-to-end test under `-tags=security`.
- **Schemas** : `schemas/onerc-v1.json` published (frozen v1).
- **Dependencies** : `github.com/zalando/go-keyring`, `golang.org/x/term`, `go.yaml.in/yaml/v3` (promoted from transitive).

### Project boilerplate (v0.0.1)
- Module layout, ports, adapter stubs (Phase 2+ slots), CI, golangci, lefthook, goreleaser config.

[Unreleased]: https://github.com/elydelva/one/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/elydelva/one/releases/tag/v0.2.0
[0.1.0]: https://github.com/elydelva/one/releases/tag/v0.1.0
