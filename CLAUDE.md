# CLAUDE.md

> Operational instructions for AI agents (Claude Code, Cursor, Aider) contributing to the **One CLI** binary. Read this file first. It points to other docs for details. If you have 5 minutes before coding, spend them here.

## The project in 30 seconds

**One CLI** is a Go binary that unifies access to third-party APIs for AI agents. Three structural pillars:

1. A **local vault** multi-account encrypted (never SaaS)
2. A **scope file** `.onerc.yaml` versioned that makes explicit what an agent can do
3. An **open source catalog** of services, distributed via Git + CDN

Four verbs on the agent side: `one <service> <action>`, `one capabilities`, `one info`, `one can`.

**Current status**: v1.0.

## What to do before coding

### 1. Read the right docs

Depending on your task, read in this order:

| Task | Docs to read |
|---|---|
| Understand the project | DESIGN.md (10 min) |
| Touch the Go binary | ARCHITECTURE.md (20 min) then TESTING.md (15 min) |
| Add a service to the catalog | CATALOG.md (15 min) |
| Write a WASM handler | HANDLERS.md (15 min) then SECURITY.md (sandbox section) |
| Modify the scope file format | SCOPE.md (10 min) then open an RFC first |
| Touch auth | AUTH.md (15 min) then SECURITY.md |
| Modify the CLI | CLI.md (10 min) |

**Don't read everything systematically.** Target the doc that matches your task.

### 2. Local setup

```bash
go version                    # must be 1.23+
make build                    # produces ./bin/one
make test                     # must pass (otherwise stop, report it)
./bin/one --version
```

If `make test` fails on main, don't start your task. Report it, it's probably a fix that needs to land first.

### 3. Identify the scope

Before writing code, **clearly state which use case you're touching** and **which adapter or port**. If you can't precisely locate your change in the ARCHITECTURE.md layout, you're not ready to code.

## Architecture at a glance

```
cmd/one/main.go                  composition root (≤100 lignes)
    │
    ▼
internal/cli/                    cobra commands, parsing, exit codes
    │
    ▼
internal/app/                    use cases (ExecuteAction, Login, ShowInfo, ...)
    │
    ▼
internal/core/  ◄── internal/ports/  ◄──  internal/adapters/<port>/
(domaine pur,        (interfaces)         (implémentations concrètes)
 zéro deps)
```

**The golden rule**: `internal/core/` only imports the Go stdlib. No exceptions. If your code in `core/` needs YAML, HTTP, or external crypto, it belongs elsewhere.

## Common workflows

### Adding a new CLI command

1. Define the use case in `internal/app/<usecase>.go`
2. Define the cobra command in `internal/cli/<command>.go`
3. Wire in `cmd/one/main.go` (composition root)
4. Tests: unit test the use case with fakes, integration via `internal/app/<usecase>_test.go`, and an E2E case if critical
5. Update `CLI.md` with the new command
6. If the command is accessible to agents: update the `one skill`

### Adding a new adapter (e.g. a 2nd Vault implementation)

1. Create the file in `internal/adapters/<port>/<impl>.go`
2. Implement the port interface
3. **Run the contract tests**: `portstest.Run<Port>Tests(t, "<Impl>", factory)`
4. Wire in the composition root if you want it to be used
5. Document briefly in the relevant doc (AUTH, or other)

Contract tests are the key element. If you don't run them, you're not done.

### Adding a new port (new domain need)

**Stop.** This is probably a sign that a discussion is needed first. Open an issue, propose the port and its interface, wait for feedback. A new port = new responsibility in the domain, it deserves thought.

### Fixing a bug

1. Reproduce the bug with a failing test
2. Fix the code
3. Verify the test passes
4. Verify the full suite passes: `make test`
5. If the bug came from a gap in the docs, update the docs

### Changing behavior described in a doc

**Update the doc in the same PR.** Not in a separate PR "later". A doc that lies is worse than no doc at all.

## Non-negotiable rules

These rules exist for reasons documented in DESIGN.md. Don't break them without opening an RFC first.

1. **Strict default deny.** No permission, no access, no URL is implicit.
2. **No I/O in `internal/core/`.** The domain is pure.
3. **Every secret is typed `core.Secret`.** Never a plain string for a token.
4. **No panic in application code.** Return a typed error.
5. **No DI framework.** Explicit composition in main.go.
6. **No custom code generation** (beyond `go generate` for structures from JSON Schema).
7. **Strict URL allowlist for WASM handlers.** No escape possible.
8. **Cross-platform tests for anything touching the keychain, paths, or locks.**

If a PR breaks one of these rules, it will be rejected even if the code is elegant.

## Common anti-patterns

To avoid, seen frequently in early contributions:

### Storing a global HTTP client in a package variable
```go
var defaultClient = &http.Client{Timeout: 30 * time.Second}
```
**No.** All config goes through constructors. No mutable globals.

### Logging credentials
```go
log.Info("got token", "token", cred.AccessToken.Reveal())
```
**No.** The `Secret` type returns `[REDACTED]` by default. `Reveal()` only at the HTTP injection point.

### Mocks with 30 expectations
```go
mockVault.EXPECT().Fetch(gomock.Any(), gomock.Eq(ref1)).Return(...)
mockVault.EXPECT().Store(gomock.Any(), gomock.Any(), ...)
// ... 28 lignes
```
**No.** Use the fakes in `internal/testing/fake/`. See TESTING.md.

### Importing an external lib into `core/`
```go
// internal/core/credential.go
import "filippo.io/age"
```
**No.** The domain doesn't know who encrypts. Put that in `adapters/vault/age.go`.

### Opening a PR that touches 20 files
**No.** Break it down. A PR has a precise and achievable goal, ideally under 10 files modified.

### Adding a Go dependency without discussion
**No.** Every dependency has a cost. See CONTRIBUTING.md for the criteria.

### `time.Sleep` to synchronize a test
```go
go startBackground()
time.Sleep(100 * time.Millisecond)
```
**No.** Flaky in CI. Use channels or `t.Cleanup`.

### Tests that depend on execution order
**No.** Each test independent, clean setup + cleanup.

### Evasive answer on a trade-off
If you're thinking "I don't know, I'll do both to be safe" and you code both options, **no**. Pick one. Document the choice. If you're genuinely unsure, ask the maintainer via the issue before coding.

## Commit and PR conventions

**Conventional commits.** Format:

```
feat(auth): add bitbucket OAuth provider

Implements OAuth 2.0 user-flow with PKCE. Tested against the
ports.AuthProvider contract.

Closes #142
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `chore`, `ci`, `build`.

Scopes: `core`, `auth`, `vault`, `catalog`, `runtime`, `cli`, `scope`, `skill`, etc.

**One PR = one goal.** No "catch-all PRs" mixing a fix, a refactor, and a feature.

**Descriptive and imperative PR title**: "Add bitbucket OAuth provider", not "Bitbucket stuff".

**PR description**: what changes, why, what's not covered, potential breaking changes.

## Where to find things

A few quick pointers to avoid wasting time searching:

| Question | Answer |
|---|---|
| How to create a new typed error? | `internal/core/errors.go` |
| How to add an exit code? | `internal/cli/exit.go`, mapping from typed error |
| Format of `service.yaml`? | CATALOG.md + `pkg/catalog/schema/v1.json` |
| How to write a fake? | `internal/testing/fake/` (existing models) |
| How to write a contract test? | `internal/testing/portstest/` (existing models) |
| How to debug a WASM handler? | HANDLERS.md, tests section + `ONE_DEBUG=1` |
| Where are credentials stored locally? | Native keychain (macOS/Linux/Windows), see AUTH.md |
| How to add a catalog fixture for tests? | `internal/testing/fixture/catalog/v1-<name>/` |
| How to generate a benchmark? | `make bench`, see TESTING.md perf section |
| How to release a new version? | `make release`, SemVer tag, CI does the rest |

## How you (agent) use One CLI

When working on the repo, **you have `one` installed**. You can and should use it instead of hand-coding API calls.

```bash
one capabilities --scope-only      # qu'est-ce que je peux faire ?
one info <service>                 # comment utiliser un service
one can <service> <action>         # vérifie avant de tenter
one <service> <action> ...         # exécute
```

If you're wondering how to interact with a third-party service (GitHub, Linear, etc.), **use `one`** rather than generating `curl` or asking the user for their credentials.

If an action returns `setup_required`, the JSON output contains `install.command`. Display it to the user, wait for confirmation, then continue.

If an action returns `not_in_scope` (exit 3), propose `one scope add <service> <perm>` to the user. **Never bypass the scope file by going through something else.**

## Response style to the user

The primary user (Ely) prefers:

- **Direct and pragmatic.** No "Great question!", no "Let me think about this carefully". Get to the point.
- **Anti-theatrical.** No pompous language. No "embark on this journey", no "let's dive deep".
- **Constructive pushback accepted.** If you see an undiscussed trade-off, flag it with your opinion.
- **No em dashes.** Use `:`, `(`, or short sentences.
- **English** (except code, commits, and technical entity names).
- **Concise.** If a 5-line answer is enough, don't write 30 lines.

When proposing a non-trivial change, **anticipate objections**: if you see 2 viable options, name them and pick one with a brief argument.

## Three things to always do

1. **Read the relevant doc before coding.** 15 minutes saved in reading prevents 2 hours of debugging.
2. **Run `make test && make lint` before proposing a commit.** Not after "I'm done", during.
3. **Ask questions when something is ambiguous.** A 30-second question beats a rejected PR after 3 hours of work.

## Three things to never do

1. **Never modify `internal/core/` to add an external dependency.** Refactor into a port + adapter first.
2. **Never bypass a security rule** documented in SECURITY.md, even temporarily, even for "debug".
3. **Never refactor outside your scope.** If you spot 4 things to improve along the way, **note them in an issue**, but don't do them in the current PR. Focus is more valuable than completeness.

## Edge cases to know

### You're working on the `one` repo but a question concerns the catalog

The catalog (services, WASM handlers, install guides) is in a **separate repo**: `one-cli/catalog`. If the question concerns the `service.yaml` format or a concrete handler, redirect to that repo. The binary does not contain service definitions.

### You see a decision in the code that seems to contradict the doc

Ask before acting. Either the doc is wrong (fix it), the code has a bug (fix it), or there's context you're missing. All three cases deserve an issue before a change.

### You're torn between doing something "properly" and "quickly"

Default to properly. The project is young, it's the opposite of a legacy codebase: every shortcut taken now will cost 10x more in 6 months. If the user wants a quick fix, they'll tell you explicitly.

### You find a critical security bug

**Don't open a public issue.** See SECURITY.md > Disclosure policy. Email the maintainer directly.

## Resources

- **Docs** in `/docs`: DESIGN.md, ARCHITECTURE.md, CATALOG.md, HANDLERS.md, SCOPE.md, AUTH.md, SECURITY.md, TESTING.md, CLI.md, CONTRIBUTING.md, ROADMAP.md
- **Code**: `cmd/`, `internal/`, `pkg/`
- **Tests**: `*_test.go` co-located, `tests/e2e/`, `tests/security/`
- **Fixtures**: `internal/testing/fixture/`
- **GitHub Issues**: tagged `good-first-issue`, `help-wanted`, `bug`, `feat`
- **RFC**: separate repo `one-cli/rfcs` for large changes

---

*If something in this file seems outdated or contradicts the rest of the docs, it's probably true. Flag it.*
