# ROADMAP

> Implementation phases for One CLI. One phase = one GitHub milestone, one PR = one achievable objective.
> The PRs listed per phase are **scoped proposals** (conventional commit title + one-line scope), not a rigid breakdown. Grouping two trivial PRs is OK if they fit within the same objective.

## Current status — v0.0.1 (skeleton)

- Go module `elydelva/one`, hexagonal layout (`cmd/`, `internal/{core,ports,adapters,app,cli,testing}`, `pkg/{catalog,handlersdk}`)
- Composition root wired in `cmd/one/main.go` (all ports resolved to concrete adapters or stubs)
- Tooling: Makefile, golangci, lefthook, goreleaser, renovate, codecov, schemas
- `go build ./...` ✅ — `go test ./...` ✅ **zero tests for now**
- No real implementation: all adapters are stubs with no business logic

**Next action**: open the **v0.1** milestone and start Phase 1.

---

## Phase 1 — Core, Vault, Scope (v0.1)

**Measurable objective**: `one login github --provider token-paste && one scope add github issues.read && one can github issues.read` → exit 0, credential stored in keychain, scope versioned in `.onerc.yaml`.

### PRs

1. `feat(core): typed errors + Secret type with redaction` — `internal/core/{errors,secret}.go`, canary leak test (log/print/marshal never in plaintext).
2. `feat(core): value objects ServiceID / AccountAlias / PermissionPath` — validation at construction, table-driven tests.
3. `feat(core): Permission + glob matching` — `*` single-level, reject `**`/`?`/brace.
4. `feat(core): Scope.Allows precedence` — deny exact > deny glob > allow exact > allow glob > default-deny, property-based tests.
5. `feat(core): Credential + NeedsRefresh + Account model`.
6. `feat(ports): finalize interfaces` — `Catalog`, `Vault`, `ScopeStore`, `AuthProvider`, `Clock`, `Logger`, `Crypto`, `Runtime`, `Transport`, `Renderer`.
7. `feat(testing): fakes in-memory + harness portstest` — vault, scope, clock (contract tests reusable by any adapter).
8. `feat(adapters/clock): system + fake clock`.
9. `feat(adapters/vault): EnvVar source` — `ONE_CREDS_<SERVICE>_<ACCOUNT>` JSON, contract tests.
10. `feat(adapters/vault): Keyring via zalando/go-keyring` — cross-OS contract tests (CI Linux/macOS, Windows with `-short`).
11. `feat(adapters/vault): chain composite env → keyring` with `ErrNotAuthenticated` fallthrough.
12. `feat(adapters/scopestore): FileScopeStore` — YAML (goccy/go-yaml), JSON Schema validation (schemas/onerc-v1).
13. `feat(adapters/scopestore): merged base + local` — anti-extension rule (local cannot widen base), warnings.
14. `feat(app): ManageScope use case` — Show / Add / Remove / Check.
15. `feat(app): Login + Logout use cases` — `token_paste` provider only in v0.1.
16. `feat(app): ListCapabilities scope-only + Can precheck`.
17. `feat(cli): cobra root, global flags, exit code mapping` — 0/1/2/3/4/5 typed from `core.Error`.
18. `feat(cli): commands login, logout, scope show/add/remove/check, can, init`.
19. `feat(adapters/renderer): JSON renderer` — **freeze the v1 schema in `schemas/`** (breaking changes forbidden after this).
20. `chore(testing): security canary leak test on full login flow`.

### Exit criteria

- `make test && make lint && make test-security` green in CI matrix (Linux/macOS/Windows).
- Coverage `internal/core/` > 85%.
- JSON output schema v1 published in `schemas/`.
- Cold-start `one --version` < 30 ms.

### Risks to address early

- **Windows keychain in CI**: no interactive session, plan for `-short` mode.
- **Frozen JSON schema**: any change after v0.1 = SemVer major.

---

## Phase 2 — Catalog FS, Declarative Runtime, Execute (v0.2)

**Measurable objective**: `one github issues.list --repo x/y` executes an HTTP GET (mocked in E2E), validates inputs, enforces scope, renders conformant JSON v1.

### PRs

1. `feat(pkg/catalog): JSON Schema v1` — `service.yaml`, `action.yaml`, `onerc-v1` finalized.
2. `feat(pkg/catalog): Go structs + YAML→struct loader` with validation on load.
3. `feat(adapters/catalog): FS adapter` — layout `services/<id>/{service.yaml,actions/,guides/,SKILL.md}` + contract tests.
4. `feat(adapters/transport): nethttp` — timeout, redirect cap, reject `http://`, **SSRF block** (RFC1918, link-local, loopback).
5. `feat(adapters/runtime): declarative runtime` — path/query/body/header interpolation, auth injection, response passthrough.
6. `feat(adapters/runtime): HTTP error → typed core.Error mapping` — 401/403/404/429/5xx + hints + `install_guide` refs.
7. `feat(adapters/runtime): cursor pagination` (cursor style, safety cap `max_pages`).
8. `feat(adapters/runtime): retry/backoff on 429 + idempotency-key plumbing`.
9. `feat(adapters/runtime): input validation` — pattern/enum/min/max/required/`@file_ref`.
10. `feat(app): ExecuteAction complete` — catalog → scope → validate → vault → runtime → audit hook.
11. `feat(app): ShowInfo (SKILL.md) + ListCapabilities full` (flag `in_scope`).
12. `feat(cli): dispatch one <service> <action>` — `@file`, `--stdin`, `--dry-run`, `--as <alias>`.
13. `feat(cli): completion bash/zsh/fish` (services loaded from catalog).
14. `feat(testing/fixture): complete v1-minimal catalog fixture` (github read-only) + reusable fakeAPI.
15. `test(e2e): happy path issues.list + coverage of 5 typed exit codes` + JSON schema validation.
16. `feat(cli): TTY confirmation for destructive actions` (skip if `--confirm` or piped).

### Exit criteria

- E2E happy path + exit codes 1/2/3/5 tested.
- JSON output automatically validated against `schemas/`.
- Coverage `internal/app/` > 75%.
- Cold-start declarative action < 50 ms.

---

## Phase 3 — WASM Runtime, Handler SDK (v0.3)

**Measurable objective**: handler `echo.wasm` (Go) + handler `notion.pages.create` (TS via Javy) run sandboxed, strict URL allowlist, escape test suite green.

### PRs

1. `feat(adapters/runtime): wazero adapter` — minimal WASI **without** fs/env/clock/random/exec/net.
2. `feat(runtime/wasm): host.creds` — allowlist from `service.yaml > credentials`.
3. `feat(runtime/wasm): host.http` — URL allowlist from `calls:`, regexes validated at load (anti-ReDoS).
4. `feat(runtime/wasm): host.crypto` — sha256/512, hmac, randomBytes, uuidV4, base64, hex.
5. `feat(runtime/wasm): host.time` — `now`, `sleep` capped (30 s per call, 60 s cumulative).
6. `feat(runtime/wasm): host.log` — debug/info/warn, cap 1000 lines per run.
7. `feat(runtime/wasm): host.fail.withCode` — mapping to `errors:` in YAML.
8. `feat(runtime/wasm): resource caps` — memory 64/256 MB, CPU 30/120 s, http calls 50, stack 1 MB.
9. `feat(runtime/wasm): host_api_version check` — reject on mismatch.
10. `feat(adapters/runtime): RoutingRuntime` — declarative vs WASM based on `action.RequiresHandler()`.
11. `feat(pkg/handlersdk): TS SDK @one-cli/handler-sdk-ts` + Javy build template.
12. `feat(pkg/handlersdk): Go SDK (tinygo target=wasi)`.
13. `feat(pkg/handlersdk): fakeHost test helper` (TS + Go).
14. `test(security): sandbox escape suite` — fs read, env get, exec, OOM, infinite loop, evil URL, redirect to private IP.
15. `bench(runtime): cold-start WASM < 80 ms p99, RSS < 30 MB` + AOT precompilation cache.
16. `feat(pkg/catalog): static lint` — URLs hit ⊆ `calls:`, `creds.get` keys ⊆ `credentials:`, `fail` codes ⊆ `errors:`.

### Exit criteria

- All `-tags=security` tests green (including the 6 SECURITY.md scenarios).
- WASM bench budgets met.
- Catalog lint rejects any escape attempt at publish time.

---

## Phase 4 — HTTP Catalog, Full Auth, Lock, Install (v0.4)

**Measurable objective**: OAuth user-flow (Notion) + device-flow (GitHub) + AWS keys + reproducible lock file + renderable install guide.

### PRs

1. `feat(adapters/catalog): HTTP adapter` — CDN JSON index + SHA256-verified tarballs.
2. `feat(adapters/catalog): Cached decorator` — LRU TTL, clock-driven.
3. `feat(adapters/catalog): Chain FS → HTTP fallback`.
4. `feat(adapters/auth): oauth2_user` — PKCE, ephemeral local server, CSRF state, 5 min timeout, browser opening.
5. `feat(adapters/auth): oauth2_device` — RFC 8628 polling.
6. `feat(adapters/auth): oauth2_client` — machine-to-machine.
7. `feat(adapters/auth): token_paste + api_key` — no-echo prompt, validate endpoint.
8. `feat(adapters/auth): aws_keys` — validation via STS `GetCallerIdentity`.
9. `feat(adapters/auth): certificate (mTLS)` — PEM read from vault.
10. `feat(adapters/vault): age file vault` (filippo.io/age) — passphrase prompt/env.
11. `feat(adapters/vault): full chain env → keyring → age`.
12. `feat(app): lazy refresh with file lock` — `~/.one/locks/<svc>:<acc>.lock`, 10 s timeout, rollback on store failure.
13. `test(security): refresh race` — 10 concurrent goroutines → exactly 1 refresh, no revoked token used.
14. `feat(app): Lock use case` — `.onerc.lock` generate/update/check with `index_sha256` + `tarball_sha256`.
15. `feat(cli): one lock [--update | --update-all | --check]`.
16. `feat(app+cli): InstallGuide rendering` — markdown + frontmatter parser, JSON/TTY modes, executable `auto_install`.
17. `feat(app+cli): auto hint on error → install_guide` (`auto_detect_on_error`).
18. `feat(app+cli): one doctor` — binary version, catalog freshness, vault source, accounts, scope, lock.
19. `feat(app+cli): commands accounts / rotate / refresh / vault export-import-status`.
20. `feat(adapters/scopestore): profiles + extends` — `default`/`production`, env `ONE_PROFILE`.

### Exit criteria

- OAuth flows tested against fakeAPI (PKCE, state, timeout).
- Refresh race test passes without flakiness.
- Lock mismatch → exit 1 with actionable hint.
- `one doctor` covers all checks listed in SECURITY.md > anti-patterns.

---

## Phase 5 — Polish, Trace, Skill, Release (v1.0)

**Measurable objective**: binary released via goreleaser on 3 OS × 2 architectures, signed `install.sh`, `one` skill installable in Claude Code / Cursor / Aider.

### PRs

1. `feat(adapters/renderer): TTY via lipgloss` — colors, `NO_COLOR` env, isatty detection.
2. `feat(adapters/renderer): bubbletea for interactive install guides` (checklist).
3. `feat(app+cli): one trace` — NDJSON `~/.one/audit.log`, 30-day rotation, `--since`, `--auth`, `--service` filters, detail by `<trace_id>`.
4. `feat(app+cli): one skill [--install --ide claude-code|cursor|aider]` + auto IDE detection.
5. `feat(app+cli): one catalog search/update/lint/scaffold/test`.
6. `feat(app+cli): one upgrade` — self-update with SHA256 verification.
7. `feat(cli): warning side_effects: destructive + prompt`.
8. `perf(cli): cold-start budgets enforced in CI` — `.benchmarks.json` (`--version` < 30 ms, declarative < 50 ms, WASM < 80 ms, RSS < 30 MB).
9. `chore(release): goreleaser cross-platform` — linux/darwin/windows × amd64/arm64, macOS codesign.
10. `chore(release): install.sh` — SHA256 verification + documented canonical URL.
11. `chore(ci): weekly vulncheck + renovate tuning` (automerge patch).
12. `docs: SKILL.md for the binary (one skill)` + finalization of TOOLING/CONTRIBUTING.
13. `test(e2e): complete workflow init → login → scope → exec → trace` + TTY snapshots.
14. `feat(security): vault rotate + audit log LOGIN/REFRESH/LOGOUT`.
15. `chore(supply-chain): SBOM + signed commits on main + enforced branch protection`.

### Exit criteria

- Perf budgets met in CI matrix (3 OS).
- Global coverage > 70%.
- Signed release artifacts + published hashes.
- Skill auto-installable in the 3 target IDEs.
- Disclosure policy + `security@` operational.

---

## Inter-phase dependencies

- **Phase 2** depends on Phase 1 (Vault, Scope, ports, Secret, errors, JSON renderer).
- **Phase 3** depends on Phase 2 (RoutingRuntime, catalog FS, catalog lint).
- **Phase 4** depends on Phases 1-2 (extensible vault chain, stable declarative runtime). Auth ports are defined in Phase 1, but only one impl (`token_paste`) lives there.
- **Phase 5** depends on everything else (trace is cross-cutting, skill = up-to-date docs).

## Cross-cutting risks (address from the indicated phase)

- **Phase 1** — Cross-platform keychain (Windows CI without interactive session → `-short` tests).
- **Phase 1** — Frozen JSON output schema v1 (breaking change = SemVer major).
- **Phase 2** — SSRF + URL allowlist in transport (do not wait for Phase 3 WASM).
- **Phase 1** — File locks on vault writes (not only refresh in Phase 4).
