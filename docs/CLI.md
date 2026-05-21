# CLI.md

> Exhaustive reference for `one` binary commands. For concepts (scope, vault, catalog), see the dedicated docs.

## General conventions

### Command format

```
one [global flags] <command> [args] [command flags]
one <service> <action> [inputs]                # short form for exec
```

### Global flags

Persistent on the cobra root:

| Flag | Description |
|---|---|
| `--json` | Force JSON output even in TTY (default: auto-detected) |
| `--account <alias>` | Account to use (equivalent to `--as` on the exec side) |
| `--dry-run` | Execute without side effects (for mutations) |
| `--project <dir>` | Project directory (default: cwd) |
| `--help`, `-h` | Show help |
| `--version`, `-v` | Show version |

Profile: controlled via the `ONE_PROFILE` environment variable (not a flag). No `--tty`, `--quiet`, `--debug`, `--trace`, `--no-color`, `--catalog-dir` in v0.4.

### TTY detection

By default:

- **stdout is a TTY**: colored and formatted output
- **stdout is piped**: JSON output

Can be overridden via `--json` or `--tty`. Important: if you pipe the output to `jq`, you will get JSON automatically.

## Exit codes

Stable convention, **public API for agents**.

| Code | Meaning | When |
|---|---|---|
| 0 | Success | The action executed and succeeded |
| 1 | Generic error | Internal bug, non-specific network error, input validation |
| 2 | Not authenticated | No credential for this service/account, or refresh not possible |
| 3 | Out of scope | The action is not allowed by `.onerc.yaml` |
| 4 | Setup required | A human step is needed (`one install` suggested) |
| 5 | Unknown service or action | Not in the catalog |

Any exit code >5 is a bug. Agents can rely on these codes without parsing stderr.

## JSON output format

Stable schema for `one <service> <action>` in JSON mode:

```json
{
  "ok": true,
  "data": { ... },                  // action output
  "trace_id": "01HXYZ...",
  "warnings": []
}
```

On error:

```json
{
  "ok": false,
  "error": {
    "code": "not_in_scope",
    "message": "Permission github.repos.delete not allowed",
    "hint": "Run `one scope add github repos.delete` to allow",
    "install": null
  },
  "trace_id": "01HXYZ..."
}
```

With `install` when relevant:

```json
{
  "ok": false,
  "error": {
    "code": "setup_required",
    "message": "Page not accessible",
    "install": {
      "service": "notion",
      "guide": "share-page",
      "command": "one install notion share-page",
      "requires_human": true
    }
  }
}
```

## Commands by category

### Initialization

#### `one init`

Creates a minimal `.onerc.yaml` in the current directory and adds `.onerc.local.yaml` to `.gitignore` (created or appended).

```bash
one init
```

No `--from-template` in v0.4.

#### `one doctor`

Full diagnostic of the installation and config.

```bash
one doctor
```

Output: one line per check, prefixed by `✓` (ok), `!` (warn) or `✗` (fail).

```
✓ scope    2 service(s) in scope
✓ catalog:github
! vault:github   no accounts (run `one login github`)
✓ lock     /path/.onerc.lock
✓ home     /Users/x/.one
```

Exit 0 if no failures, 1 as soon as a `fail` check appears (`warn` checks do not cause failure).

#### `one upgrade`

Deferred to v0.5.

### Authentication

#### `one login <service>`

Authenticates to the service. Default provider: `pat`.

```bash
one login github                              # provider pat
one login github --provider oauth2_user       # OAuth user-flow
one login github --provider oauth2_device     # device flow (headless)
one login github --as perso                   # creates/overwrites the "perso" alias
```

No `--client-id` flag in v0.4 (the `client_id` values are resolved from the catalog / env variables documented by the service). See [AUTH.md](./AUTH.md).

#### `one logout <service>`

Removes an account from the vault.

```bash
one logout github                     # removes "default"
one logout github --as perso          # removes "perso"
```

No `--all` in v0.4.

#### `one accounts <service>`

Lists registered accounts for a service (required argument).

```bash
one accounts github
```

Output: one `<service>:<alias>` line per account, or `no accounts`.

#### `one rotate <service> <account>`

Re-runs the login flow and overwrites the credential in the vault.

```bash
one rotate github work
```

OAuth provider-side revocation is deferred to v0.5.

#### `one refresh <service> <account>`

Forces a manual refresh without waiting for expiration.

```bash
one refresh github work
```

Useful for diagnosing a refresh issue.

#### `one vault export`

Dumps **plaintext** JSON of in-scope credentials to stdout. Pipe to `age -p` (or equivalent) before persisting.

```bash
one vault export | age -p > backup.age
```

#### `one vault import`

Restores from a JSON bundle read from stdin. Always overwrites existing entries (no `--overwrite` flag: this is the default behavior).

```bash
age -d backup.age | one vault import
```

#### `one vault status`

JSON: credential count per in-scope service.

```bash
$ one vault status
{
  "services": { "github": 2, "notion": 1 },
  "total": 3
}
```

### Scope and permissions

#### `one scope show [service]`

Shows the effective scope (merges `.onerc.yaml` + `.onerc.local.yaml`, and overrides `.onerc.<profile>.yaml` if `ONE_PROFILE` is set), in JSON format.

```bash
one scope show
one scope show github
```

#### `one scope add <service> <permission>`

Adds a permission to `allow` (or to `deny` with `--deny`). Always writes to `.onerc.yaml`.

```bash
one scope add github issues.read
one scope add github "issues.*"
one scope add github issues.delete --deny
```

No `--local` in v0.4.

#### `one scope remove <service> <permission>`

Removes a permission (searches in both `allow` and `deny`).

```bash
one scope remove github issues.read
```

#### `one scope check <service> <action>`

Exit 0 if allowed, exit 3 (`ErrNotInScope`) otherwise.

```bash
one scope check github issues.delete
```

#### `one scope explain <service> <action>`

Outputs `{allowed, reason, service, action}` as JSON, then exit 0 if allowed, non-zero with the reason otherwise.

`one scope use`, `--strict`, `--raw`: deferred to v0.5.

### Catalog and lock

#### `one catalog ...`

`one catalog update`, `search`, `lint`, `scaffold`, `test`: deferred to v0.5. In v0.4, the HTTP catalog is driven by `ONE_CATALOG_URL` (15-minute cache) and the FS → HTTP chain. See [CATALOG.md](./CATALOG.md).

#### `one lock`

Generates or updates `.onerc.lock` (schema v1).

```bash
one lock                              # (re)generates from the current scope
one lock --update notion              # refreshes one service
one lock --update-all                 # refreshes all in-scope services
one lock --check                      # exit 1 (ErrLockDrift) if drift
```

`--check` returns a `lock drift detected: ...` error listing divergent services, with the hint `run \`one lock --update-all\` to refresh`.

### Action execution

#### `one <service> <action>`

Primary form.

```bash
one github issues.read --issue 42
one notion pages.create \
  --parent_page_id abc-123 \
  --properties '{"title":[{"text":{"content":"Hello"}}]}'
one stripe customers.create \
  --email user@example.com \
  --name "Jane Doe" \
  --idempotency_key cust-2026-05-20-001
```

#### Passing inputs

Three possible modes depending on the type:

**1. Simple flags (string, number, bool)**:

```bash
one github issues.read --issue 42 --include_pull_requests true
```

**2. JSON flags for objects**:

```bash
one notion pages.create --properties '{"title":[...]}'
```

**3. `@file` for files**:

```bash
one notion blocks.append --children @blocks.json
one s3 objects.put --bucket mybucket --key file.pdf --body @./file.pdf
```

**4. stdin for piping**: deferred to v0.5.

#### Options for mutations

```bash
--dry-run                             # validation without side effect
```

`--idempotency-key` / `--confirm`: deferred to v0.5.

### Introspection (agent verbs)

#### `one capabilities`

JSON listing available services and actions.

```bash
one capabilities                      # all services + actions
one capabilities github               # just github
one capabilities --scope-only         # only what is in the current scope
```

Typical output:

```json
{
  "services": [
    {
      "name": "github",
      "version": "2.1.4",
      "actions": [
        {
          "id": "issues.read",
          "permission": "issues.read",
          "kind": "query",
          "in_scope": true,
          "summary": "Read an issue by number"
        },
        {
          "id": "issues.create",
          "permission": "issues.write",
          "kind": "mutation",
          "side_effects": "write",
          "in_scope": false,
          "summary": "Create a new issue"
        }
      ]
    }
  ]
}
```

#### `one info`

Markdown documentation.

```bash
one info                              # global "onecli" skill
one info github                       # github service skill
one info github issues.create         # doc for a specific action
```

Output: markdown for humans and agents that like markdown. For structured parsing, use `capabilities`.

#### `one can <service> <action>`

Quick permission precheck, without executing.

```bash
one can github issues.delete
# exit 0 if OK, exit 3 if not in scope, exit 2 if not authenticated
```

Useful for agents that want to check before attempting, or for scripting.

### Install and setup

#### `one install <service> [guide]`

Displays an install guide. Without `[guide]`, requires `--list`.

```bash
one install notion share-page          # displays the guide
one install notion --list              # lists all guides for the service
```

TTY output (simple guide):

```
# <title>

<content markdown>

Verify: one <service> <action>          # if frontmatter has `verify.action`
```

`--list` prints `<id>\t<title>` per line. No `--verify`, nor automatic execution in v0.4.

### Skill and IDE integration

#### `one skill`

Stub in v0.4: returns `not implemented`. The `--install` flag is declared but inert. Deferred to v0.5.

### Audit and debug

#### `one trace`

Wired on the CLI side but the implementation returns `not implemented`. Audit log persistence deferred to v0.5.

#### `--debug`

Deferred to v0.5. The internal logger runs at `warn` level as text output on stderr.

## Environment variables

| Variable | Description |
|---|---|
| `ONE_CATALOG_URL` | Enables the catalog HTTP layer (otherwise FS only) |
| `ONE_CATALOG_ROOT` | Overrides the local catalog directory (default `$HOME/.one/catalog`) |
| `ONE_AGE_VAULT_PATH` | Path to the age vault file (default `$HOME/.one/vault.age`) |
| `ONE_AGE_PASSPHRASE` | Age vault passphrase (required to enable the age layer) |
| `ONE_PROFILE` | Active scope profile (loads `.onerc.<profile>.yaml`) |
| `ONE_CREDS_<SERVICE>_<ACCOUNT>` | Inline credential (JSON storage shape) — read-only vault |
| `ONE_CERT_<SERVICE>_<ACCOUNT>_CERT` | PEM client cert path (provider `certificate`) |
| `ONE_CERT_<SERVICE>_<ACCOUNT>_KEY` | PEM private key path (provider `certificate`) |
| `ONE_TRANSPORT_ALLOW_HTTP` | `1` to tolerate `http://` (tests; refused by default) |
| `ONE_TRANSPORT_ALLOWED_HOSTS` | SSRF bypass for these hosts (CSV) |

`ONE_DEBUG`, `ONE_NO_COLOR`, `ONE_<SERVICE>_API_BASE`, `ONE_PPROF`, `ONE_HOME`, `XDG_CONFIG_HOME`: not wired in v0.4.

### Usage examples

```bash
# CI: credentials via env, no keychain
export ONE_CREDS_GITHUB_DEFAULT='{"access_token":"ghp_xxx","provider":"pat","service":"github","account":"default"}'
one github issues.list --repo me/repo

# CI: age vault from a secret
export ONE_AGE_VAULT_PATH=/tmp/vault.age
export ONE_AGE_PASSPHRASE="$VAULT_PASSPHRASE"

# Restrictive profile
export ONE_PROFILE=production
one stripe customers.create --email ...
```

## Files used

### Per project (cwd)

| File | Description |
|---|---|
| `.onerc.yaml` | Main scope file (committed) |
| `.onerc.local.yaml` | Personal override (gitignored) |
| `.onerc.lock` | Pinned catalog versions (committed) |

### Global (`~/.one/`)

| File/directory | Description |
|---|---|
| `~/.one/catalog/` | Local FS catalog (override via `ONE_CATALOG_ROOT`) |
| `~/.one/locks/<service>:<alias>.lock` | Refresh file locks (gofrs/flock, 10s timeout) |
| `~/.one/vault.age` | Age vault (if the age layer is enabled) |
| `~/.one/cache/wasm/` | Cache of compiled WASM modules |

`audit.log`, `config.yaml`, XDG convention: deferred to v0.5.

## Full workflow

### First installation

```bash
# Install the binary
curl -fsSL https://one-cli.dev/install.sh | sh

# Verify
one --version
one doctor

# (Optional) Install the skill in Claude Code
one skill --install
```

### First project

```bash
cd my-project
one init                              # creates .onerc.yaml

# Login to services
one login github
one login notion --as kaampus

# Define scope
one scope add github issues.*
one scope add github pulls.read
one scope add notion pages.*

# Lock
one lock

# Commit
git add .onerc.yaml .onerc.lock .gitignore
git commit -m "init: setup One CLI"
```

### Agent workflow

```bash
# Discovery
one capabilities --scope-only          # what can I do?
one info github                        # how does it work?

# Pre-check
one can github issues.create          # is this allowed?

# Execution
one github issues.create \
  --repo me/myrepo \
  --title "Bug: foo" \
  --body "Detailed description"
# stdout: {"ok":true,"data":{"id":42,"url":"https://github.com/..."}}

# On setup error
# stdout: {"ok":false,"error":{"code":"setup_required",...,"install":{"command":"one install ..."}}}
# The agent knows what to do: suggest to the human to run the install
```

### CI workflow

```bash
# In the pipeline
export ONE_AGE_VAULT_PATH=/tmp/vault.age
export ONE_AGE_PASSPHRASE="$VAULT_PASSPHRASE"

aws s3 cp s3://my-secrets/vault.age $ONE_AGE_VAULT_PATH

one doctor                            # verify everything is OK
one lock --check                      # exit 1 (ErrLockDrift) if drift

one github issues.list --repo me/repo
```

## Detailed behavior

### Resolution of the `one <service> <action>` command

Algorithm:

1. Parse global flags
2. Identify `<service>`: if it matches a built-in command (login, scope, info, etc.), route to it
3. Otherwise, load `<service>` from the catalog
4. Load `<action>` from that service
5. Parse action flags according to its input schema
6. Load the scope file
7. Verify that `action.permission` is in scope
8. Resolve the account (default → local → --as)
9. Fetch credentials from the vault
10. Refresh if necessary
11. If setup required: return ErrSetupRequired (exit 4)
12. Invoke the runtime (declarative or WASM)
13. Render output

Each step can short-circuit with a specific exit code.

### Behavior on input validation error

```bash
one github issues.read --issue not-a-number
# stderr: Error: invalid input 'issue': expected integer, got "not-a-number"
# exit 1
```

The error is validated **before** any network call. No quota wasted.

### Behavior on unknown action

```bash
one github does.not.exist
# stderr: Error: unknown action 'does.not.exist' in service 'github'
#         Did you mean: 'issues.delete'?
# exit 5
```

Suggestion via Levenshtein distance if a close action exists.

### Behavior on unknown service

```bash
one unknown-service action
# stderr: Error: unknown service 'unknown-service'
#         Run `one catalog search <query>` to find available services.
# exit 5
```

### Piped behavior

```bash
one github issues.read --issue 42 | jq .data.title
```

The binary detects that stdout is piped and switches to JSON automatically. Stderr remains readable for warnings.

### Interactive TTY behavior

```bash
one stripe customers.delete --customer cus_xxx
# Confirm: This will permanently delete customer cus_xxx. Type 'yes' to confirm: _
```

For `side_effects: destructive` actions, a confirmation prompt in TTY. Bypassed via `--confirm` or when piped (where the prompt makes no sense).

## Outgoing error codes (beyond exit codes)

Stable codes in the `error.code` field of JSON output:

| Code | Meaning |
|---|---|
| `not_authenticated` | no credential or refresh not possible |
| `not_in_scope` | permission denied by scope file |
| `setup_required` | human action required (with install hint) |
| `unknown_service` | service not in the catalog |
| `unknown_action` | action not in the service |
| `invalid_input` | inputs do not match the schema |
| `api_error` | error returned by the third-party service |
| `rate_limited` | third-party service rate limit (with retry-after) |
| `not_found` | resource does not exist |
| `forbidden` | credentials are valid but permission denied |
| `network_error` | timeout, DNS failure, connection refused |
| `internal_error` | binary bug (to be reported) |
| `lock_violation` | lock file does not match installed catalog |
| `url_not_allowed` | a handler attempted a call outside the allowlist (catalog bug) |
| `resource_exhausted` | timeout/memory/calls limits hit |

Agents can switch on the code without parsing the message.

## Action naming conventions

Action names follow `<resource>.<verb>`, in lowercase:

| Verb | Semantics |
|---|---|
| `read`, `get`, `retrieve` | read a resource by ID |
| `list` | paginated list of resources |
| `search`, `query` | search with filters |
| `create` | creation |
| `update` | partial update |
| `replace` | full update |
| `delete`, `archive` | deletion (destructive) |
| `enable`, `disable` | state toggle |
| `attach`, `detach` | link management |
| `subscribe`, `unsubscribe` | events |

If you are unsure when contributing a service, look at existing services in the same domain.

## Commands for catalog contributors

`one catalog lint`, `scaffold`, `test`, as well as `one completion` (bash/zsh/fish): deferred to v0.5.

---

*Any command added requires: an integration test in `internal/app/`, an E2E test in `tests/e2e/`, documentation in this file, and an entry in the `one skill` if agents need to know about it.*
