# ROADMAP

> Phases d'implémentation de One CLI. Une phase = un milestone GitHub, une PR = un objectif atteignable.
> Les PRs listées par phase sont des **propositions cadrées** (titre conventional commit + scope d'une ligne), pas un découpage rigide. Regrouper deux PRs triviales est OK si elles tiennent dans un même objectif.

## Statut actuel — v0.0.1 (squelette)

- Module Go `elydelva/one`, layout hexagonal (`cmd/`, `internal/{core,ports,adapters,app,cli,testing}`, `pkg/{catalog,handlersdk}`)
- Composition root câblée dans `cmd/one/main.go` (tous les ports résolus vers adapters concrets ou stubs)
- Tooling : Makefile, golangci, lefthook, goreleaser, renovate, codecov, schemas
- `go build ./...` ✅ — `go test ./...` ✅ **zéro test pour l'instant**
- Aucune implémentation réelle : tous les adapters sont des stubs sans logique métier

**Prochaine action** : ouvrir le milestone **v0.1** et commencer Phase 1.

---

## Phase 1 — Core, Vault, Scope (v0.1)

**Objectif mesurable** : `one login github --provider token-paste && one scope add github issues.read && one can github issues.read` → exit 0, credential stocké dans le keychain, scope versionné dans `.onerc.yaml`.

### PRs

1. `feat(core): typed errors + Secret type with redaction` — `internal/core/{errors,secret}.go`, canary leak test (log/print/marshal jamais en clair).
2. `feat(core): value objects ServiceID / AccountAlias / PermissionPath` — validation à la construction, table-driven tests.
3. `feat(core): Permission + glob matching` — `*` single-level, refus de `**`/`?`/brace.
4. `feat(core): Scope.Allows precedence` — deny exact > deny glob > allow exact > allow glob > default-deny, property-based tests.
5. `feat(core): Credential + NeedsRefresh + Account model`.
6. `feat(ports): finaliser les interfaces` — `Catalog`, `Vault`, `ScopeStore`, `AuthProvider`, `Clock`, `Logger`, `Crypto`, `Runtime`, `Transport`, `Renderer`.
7. `feat(testing): fakes in-memory + harness portstest` — vault, scope, clock (contract tests réutilisables par tout adapter).
8. `feat(adapters/clock): system + fake clock`.
9. `feat(adapters/vault): EnvVar source` — `ONE_CREDS_<SERVICE>_<ACCOUNT>` JSON, contract tests.
10. `feat(adapters/vault): Keyring via zalando/go-keyring` — contract tests cross-OS (CI Linux/macOS, Windows en `-short`).
11. `feat(adapters/vault): chain composite env → keyring` avec `ErrNotAuthenticated` fallthrough.
12. `feat(adapters/scopestore): FileScopeStore` — YAML (goccy/go-yaml), validation JSON Schema (schemas/onerc-v1).
13. `feat(adapters/scopestore): merged base + local` — règle anti-extension (local ne peut pas élargir base), warnings.
14. `feat(app): ManageScope use case` — Show / Add / Remove / Check.
15. `feat(app): Login + Logout use cases` — provider `token_paste` uniquement en v0.1.
16. `feat(app): ListCapabilities scope-only + Can precheck`.
17. `feat(cli): cobra root, global flags, mapping exit codes` — 0/1/2/3/4/5 typés depuis `core.Error`.
18. `feat(cli): commandes login, logout, scope show/add/remove/check, can, init`.
19. `feat(adapters/renderer): JSON renderer` — **figer le schema v1 dans `schemas/`** (breaking change interdit après).
20. `chore(testing): security canary leak test sur flow login complet`.

### Critères de sortie

- `make test && make lint && make test-security` verts en CI matrix (Linux/macOS/Windows).
- Coverage `internal/core/` > 85 %.
- Schema JSON output v1 publié dans `schemas/`.
- Cold-start `one --version` < 30 ms.

### Risques à traiter tôt

- **Keychain Windows en CI** : pas de session interactive, prévoir mode `-short`.
- **JSON schema figé** : tout changement après v0.1 = SemVer major.

---

## Phase 2 — Catalog FS, Runtime déclaratif, Execute (v0.2)

**Objectif mesurable** : `one github issues.list --repo x/y` exécute un GET HTTP (mocké en E2E), valide les inputs, applique le scope, render JSON v1 conforme.

### PRs

1. `feat(pkg/catalog): JSON Schema v1` — `service.yaml`, `action.yaml`, `onerc-v1` finalisés.
2. `feat(pkg/catalog): structs Go + loader YAML→struct` avec validation au load.
3. `feat(adapters/catalog): FS adapter` — layout `services/<id>/{service.yaml,actions/,guides/,SKILL.md}` + contract tests.
4. `feat(adapters/transport): nethttp` — timeout, cap redirects, refus `http://`, **SSRF block** (RFC1918, link-local, loopback).
5. `feat(adapters/runtime): declarative runtime` — interpolation path/query/body/header, injection auth, passthrough réponse.
6. `feat(adapters/runtime): mapping erreurs HTTP → core.Error typé` — 401/403/404/429/5xx + hints + refs `install_guide`.
7. `feat(adapters/runtime): pagination cursor` (style cursor, safety cap `max_pages`).
8. `feat(adapters/runtime): retry/backoff sur 429 + idempotency-key plumbing`.
9. `feat(adapters/runtime): validation inputs` — pattern/enum/min/max/required/`@file_ref`.
10. `feat(app): ExecuteAction complet` — catalog → scope → validate → vault → runtime → audit hook.
11. `feat(app): ShowInfo (SKILL.md) + ListCapabilities full` (flag `in_scope`).
12. `feat(cli): dispatch one <service> <action>` — `@file`, `--stdin`, `--dry-run`, `--as <alias>`.
13. `feat(cli): completion bash/zsh/fish` (services chargés depuis catalog).
14. `feat(testing/fixture): catalog v1-minimal complet` (github read-only) + fakeAPI réutilisable.
15. `test(e2e): happy path issues.list + couverture des 5 exit codes typés` + validation JSON schema.
16. `feat(cli): confirmation TTY pour actions destructives` (skip si `--confirm` ou pipé).

### Critères de sortie

- E2E happy path + exit codes 1/2/3/5 testés.
- JSON output validé automatiquement contre `schemas/`.
- Coverage `internal/app/` > 75 %.
- Cold-start action déclarative < 50 ms.

---

## Phase 3 — Runtime WASM, Handlers SDK (v0.3)

**Objectif mesurable** : handler `echo.wasm` (Go) + handler `notion.pages.create` (TS via Javy) tournent sandboxés, URL allowlist stricte, suite d'évasion verte.

### PRs

1. `feat(adapters/runtime): wazero adapter` — WASI minimal **sans** fs/env/clock/random/exec/net.
2. `feat(runtime/wasm): host.creds` — allowlist depuis `service.yaml > credentials`.
3. `feat(runtime/wasm): host.http` — allowlist URLs depuis `calls:`, regex validées au load (anti-ReDoS).
4. `feat(runtime/wasm): host.crypto` — sha256/512, hmac, randomBytes, uuidV4, base64, hex.
5. `feat(runtime/wasm): host.time` — `now`, `sleep` capé (30 s par appel, 60 s cumulé).
6. `feat(runtime/wasm): host.log` — debug/info/warn, cap 1000 lignes par run.
7. `feat(runtime/wasm): host.fail.withCode` — mapping vers `errors:` du YAML.
8. `feat(runtime/wasm): caps ressources` — mémoire 64/256 MB, CPU 30/120 s, http calls 50, stack 1 MB.
9. `feat(runtime/wasm): host_api_version check` — refus si mismatch.
10. `feat(adapters/runtime): RoutingRuntime` — déclaratif vs WASM selon `action.RequiresHandler()`.
11. `feat(pkg/handlersdk): SDK TS @one-cli/handler-sdk-ts` + template build Javy.
12. `feat(pkg/handlersdk): SDK Go (tinygo target=wasi)`.
13. `feat(pkg/handlersdk): helper de test fakeHost` (TS + Go).
14. `test(security): suite évasion sandbox` — fs read, env get, exec, OOM, infinite loop, evil URL, redirect vers IP privée.
15. `bench(runtime): cold-start WASM < 80 ms p99, RSS < 30 MB` + cache de précompilation AOT.
16. `feat(pkg/catalog): lint statique` — URLs hit ⊆ `calls:`, `creds.get` keys ⊆ `credentials:`, `fail` codes ⊆ `errors:`.

### Critères de sortie

- Tous tests `-tags=security` verts (incluant les 6 scénarios SECURITY.md).
- Budgets bench WASM respectés.
- Lint catalog refuse toute tentative d'évasion à la publication.

---

## Phase 4 — Catalog HTTP, Auth complet, Lock, Install (v0.4)

**Objectif mesurable** : OAuth user-flow (Notion) + device-flow (GitHub) + AWS keys + lock file reproductible + install guide rendable.

### PRs

1. `feat(adapters/catalog): HTTP adapter` — CDN index JSON + tarballs SHA256 vérifiés.
2. `feat(adapters/catalog): Cached decorator` — TTL LRU, clock-driven.
3. `feat(adapters/catalog): Chain FS → HTTP fallback`.
4. `feat(adapters/auth): oauth2_user` — PKCE, serveur local éphémère, state CSRF, timeout 5 min, ouverture navigateur.
5. `feat(adapters/auth): oauth2_device` — RFC 8628 polling.
6. `feat(adapters/auth): oauth2_client` — machine-to-machine.
7. `feat(adapters/auth): token_paste + api_key` — prompt no-echo, validate endpoint.
8. `feat(adapters/auth): aws_keys` — validation via STS `GetCallerIdentity`.
9. `feat(adapters/auth): certificate (mTLS)` — lecture PEM dans vault.
10. `feat(adapters/vault): age file vault` (filippo.io/age) — passphrase prompt/env.
11. `feat(adapters/vault): chain complet env → keyring → age`.
12. `feat(app): refresh lazy avec file lock` — `~/.one/locks/<svc>:<acc>.lock`, timeout 10 s, rollback si store fail.
13. `test(security): race refresh` — 10 goroutines concurrentes → exactement 1 refresh, aucun token révoqué utilisé.
14. `feat(app): Lock use case` — `.onerc.lock` generate/update/check avec `index_sha256` + `tarball_sha256`.
15. `feat(cli): one lock [--update | --update-all | --check]`.
16. `feat(app+cli): InstallGuide rendering` — parser markdown + frontmatter, modes JSON/TTY, `auto_install` exécutable.
17. `feat(app+cli): hint auto sur erreur → install_guide` (`auto_detect_on_error`).
18. `feat(app+cli): one doctor` — version binaire, fraîcheur catalog, source vault, comptes, scope, lock.
19. `feat(app+cli): commandes accounts / rotate / refresh / vault export-import-status`.
20. `feat(adapters/scopestore): profiles + extends` — `default`/`production`, env `ONE_PROFILE`.

### Critères de sortie

- OAuth flows testés contre fakeAPI (PKCE, state, timeout).
- Race test refresh passe sans flake.
- Lock mismatch → exit 1 avec hint exploitable.
- `one doctor` couvre tous les checks listés dans SECURITY.md > anti-patterns.

---

## Phase 5 — Polish, Trace, Skill, Release (v1.0)

**Objectif mesurable** : binaire releasé via goreleaser sur 3 OS × 2 archi, `install.sh` signé, skill `one` installable dans Claude Code / Cursor / Aider.

### PRs

1. `feat(adapters/renderer): TTY via lipgloss` — couleurs, `NO_COLOR` env, isatty detection.
2. `feat(adapters/renderer): bubbletea pour install guides interactifs` (checklist).
3. `feat(app+cli): one trace` — NDJSON `~/.one/audit.log`, rotation 30 j, filtres `--since`, `--auth`, `--service`, détail par `<trace_id>`.
4. `feat(app+cli): one skill [--install --ide claude-code|cursor|aider]` + détection IDE auto.
5. `feat(app+cli): one catalog search/update/lint/scaffold/test`.
6. `feat(app+cli): one upgrade` — self-update avec vérification SHA256.
7. `feat(cli): warning side_effects: destructive + prompt`.
8. `perf(cli): budgets cold-start enforced en CI` — `.benchmarks.json` (`--version` < 30 ms, déclaratif < 50 ms, WASM < 80 ms, RSS < 30 MB).
9. `chore(release): goreleaser cross-platform` — linux/darwin/windows × amd64/arm64, codesign macOS.
10. `chore(release): install.sh` — vérification SHA256 + URL canonique documentée.
11. `chore(ci): vulncheck hebdomadaire + tuning renovate` (automerge patch).
12. `docs: SKILL.md du binaire (one skill)` + finalisation TOOLING/CONTRIBUTING.
13. `test(e2e): workflow complet init → login → scope → exec → trace` + snapshots TTY.
14. `feat(security): vault rotate + audit log LOGIN/REFRESH/LOGOUT`.
15. `chore(supply-chain): SBOM + signed commits sur main + branch protection enforced`.

### Critères de sortie

- Budgets perf respectés en CI matrix (3 OS).
- Coverage globale > 70 %.
- Artifacts release signés + hashes publiés.
- Skill auto-installable dans les 3 IDE cibles.
- Disclosure policy + `security@` opérationnels.

---

## Dépendances inter-phases

- **Phase 2** dépend de Phase 1 (Vault, Scope, ports, Secret, errors, JSON renderer).
- **Phase 3** dépend de Phase 2 (RoutingRuntime, catalog FS, lint catalog).
- **Phase 4** dépend de Phase 1-2 (vault chain extensible, runtime déclaratif stable). Les ports auth sont posés dès Phase 1, mais une seule impl (`token_paste`) y vit.
- **Phase 5** dépend du reste (trace transverse, skill = doc à jour).

## Risques transverses (à traiter dès la phase indiquée)

- **Phase 1** — Cross-platform keychain (Windows CI sans session interactive → tests `-short`).
- **Phase 1** — JSON output schema v1 figé (breaking change = SemVer major).
- **Phase 2** — SSRF + allowlist URL dans le transport (pas attendre Phase 3 WASM).
- **Phase 1** — File locks sur writes vault (pas seulement refresh Phase 4).
