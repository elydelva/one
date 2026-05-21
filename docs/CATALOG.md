# CATALOG.md

> Complete reference for adding or modifying a service in the One CLI catalog. For external contributors to the `one-cli/catalog` repo. For the WASM contract, see [HANDLERS.md](./HANDLERS.md).

## Overview

The catalog is a public Git repo (`one-cli/catalog`) that contains the definition of each supported service. Its CI publishes a static JSON index on CDN, which a `one` binary fetches to resolve services.

**Adding a service = opening a PR.** The repo has strict CI that validates the format, runs tests, and checks security constraints. Once merged, the PR triggers automatic publication to the index.

### HTTP Distribution

The binary resolves the catalog in this order:

1. Local FS layer (`$HOME/.one/catalog` or `ONE_CATALOG_ROOT`)
2. If `ONE_CATALOG_URL` is set: HTTP layer as fallback, wrapped by a 15-minute TTL cache (clock-driven, `ports.Clock`).

CDN layout:

```
<baseURL>/index.json
<baseURL>/services/<id>.tar.gz
```

`index.json` (`version: 1`) lists for each service its `version` and `tarball_sha256`. Each gz tarball is fetched and **SHA256-verified** against the index before parsing (`ErrIntegrityCheckFailed` otherwise). Expected contents in the tar: `service.yaml`, `actions/*.yaml`, `SKILL.md`, `guides/*.md` (with an optional `<id>/` wrapper tolerated).

The `FS → HTTP` chain falls through on `ErrUnknownService` / `ErrUnknownAction` / `ErrNotSupported`; any other error short-circuits.

## Service Structure

```
services/
└── notion/
    ├── service.yaml             # metadata + auth declaration + permissions
    ├── SKILL.md                 # markdown for `one info notion`
    ├── actions/
    │   ├── pages.read.yaml
    │   ├── pages.read.md        # optional, otherwise generated from YAML
    │   ├── pages.create.yaml
    │   ├── pages.update.yaml
    │   ├── databases.query.yaml
    │   ├── blocks.append.yaml
    │   └── search.yaml
    ├── guides/
    │   ├── initial-setup.md
    │   └── share-page.md
    └── handlers/                # only if WASM is required
        ├── main.ts              # source
        ├── main.wasm            # compiled by CI, not committed
        ├── package.json
        └── tests/
            └── main.test.ts
```

**Convention**:

- The **folder name** is the service identifier used in commands (`one notion ...`).
- **Action names** follow `<resource>.<verb>` (e.g. `pages.create`, not `createPage` or `create_page`).
- **Guide names** are kebab-case slugs (`share-page`, `iam-setup`).

## The `service.yaml` File

This is the service manifest. Complete field reference.

### Minimal skeleton

```yaml
version: 1
name: notion
display_name: Notion
description: Workspace pages, databases, blocks
homepage: https://notion.so
docs_url: https://developers.notion.com
license: MIT
maintainers:
  - github: elydelva

api:
  base_url: https://api.notion.com/v1
  version: "2025-09-03"            # API version to pass
  headers:
    Notion-Version: "{api.version}"

auth:
  default_provider: oauth
  providers:
    oauth:
      type: oauth2_user
      authorize_url: https://api.notion.com/v1/oauth/authorize
      token_url: https://api.notion.com/v1/oauth/token
      client_id: "{env.ONE_NOTION_CLIENT_ID}"
      pkce: true
      callback:
        mode: local_server
        path: /callback
      injection:
        header: Authorization
        format: "Bearer {access_token}"
      validate:
        method: GET
        url: "{api.base_url}/users/me"
        expect_status: 200

permissions:
  pages.read:
    kind: query
    description: Read page properties and content
  pages.write:
    kind: mutation
    description: Create and update pages
  blocks.read:
    kind: query
  blocks.write:
    kind: mutation
  databases.read:
    kind: query
  search:
    kind: query
```

### Sections in detail

#### `version` (required)

Version of the service.yaml format. Always `1` currently. Allows the format to evolve in the future without breaking existing services.

#### Identity

| Field | Type | Required | Description |
|---|---|:---:|---|
| `name` | string | yes | service identifier, must match the folder name |
| `display_name` | string | yes | human-readable name for display |
| `description` | string | yes | 1-2 sentences, shown in `one info` |
| `homepage` | URL | yes | service's official website |
| `docs_url` | URL | yes | official API documentation (useful for agents) |
| `license` | string | no | license of service.yaml and handlers, defaults to MIT |
| `maintainers` | array | no | list of main contributors |
| `tags` | array | no | categories (productivity, payment, dev, etc.) |

#### `api`

HTTP configuration common to all service actions.

```yaml
api:
  base_url: https://api.notion.com/v1
  version: "2025-09-03"
  headers:
    Notion-Version: "{api.version}"
    User-Agent: "one-cli/{version}"
  timeout_ms: 30000
  rate_limit:
    requests_per_second: 3
    burst: 10
```

- `base_url` is prefixed before each relative `request.url` in actions.
- `version` is referenced via `{api.version}` in headers or paths.
- `headers` are injected on every request.
- `timeout_ms` is the default timeout, overridable per action.
- `rate_limit` is purely informational for now (used for hints), not enforced.

#### `auth`

List of supported auth providers. See [AUTH.md](./AUTH.md) for the semantics of each type.

```yaml
auth:
  default_provider: oauth
  providers:
    oauth:
      type: oauth2_user
      # ...
    pat:
      type: token_paste
      label: Personal Access Token
      help_url: https://github.com/settings/tokens?type=beta
      validate:
        method: GET
        url: "{api.base_url}/user"
        expect_status: 200
      injection:
        header: Authorization
        format: "Bearer {token}"
```

At login, the user chooses among providers. The `default_provider` is selected if the user does not specify one with `--provider`.

#### `permissions`

The exhaustive list of permissions exposed by the service. **This is the unit of granularity for the scope file.**

```yaml
permissions:
  pages.read:
    kind: query              # query | mutation
    description: Read page properties and content
  pages.write:
    kind: mutation
    description: Create and update pages
    side_effects: write      # write | read (default: inferred from kind)
  pages.archive:
    kind: mutation
    side_effects: destructive
```

- `kind` (`query` or `mutation`) classifies the action. Shown in `one capabilities`. Important for agents that want to filter.
- `side_effects` clarifies for mutations: `write` (create/update), `destructive` (delete, archive). Allows scope files to block destructive operations even when other mutations are allowed.

**Naming convention**:

- Always lowercase.
- Dot-separated path: `resource.verb` (e.g. `pages.read`).
- Standard verbs: `read`, `write`, `delete`, `archive`, `query`, `list`, `search`, `subscribe`.

#### `credentials`

Declares the credentials that actions/handlers can request via `host.creds.get`.

```yaml
credentials:
  access_token:
    type: secret
    source: oauth.access_token
    description: OAuth access token
  refresh_token:
    type: secret
    source: oauth.refresh_token
    optional: true
  region:
    type: string             # non-secret
    source: config.region
    description: AWS region
```

- `type`: `secret` (redacted in logs) or `string` (visible config).
- `source`: where the value comes from. Three sources:
  - `oauth.<field>`: from the OAuth flow (access_token, refresh_token, extras.*)
  - `config.<field>`: additional config entered by the user at login
  - `static.<value>`: fixed value (rare)
- `optional`: if false (default), the action fails if the credential is missing.

#### `required_setup`

Lists the install guides that may be required to use the service.

```yaml
required_setup:
  - id: initial-setup
    description: OAuth connection and integration creation
    blocks: ["*"]
  - id: share-page
    description: Share specific pages with the integration
    blocks: ["pages.*", "blocks.*", "databases.*"]
    optional: false
    detection: |
      If any API call returns 404 with code "object_not_found",
      the integration likely lacks access to that resource.
    auto_detect_on_error: object_not_found
```

- `id`: matches a `guides/<id>.md` file.
- `blocks`: which permissions this setup is required for (globs).
- `optional`: if true, the setup is suggested but not mandatory to use these permissions.
- `detection`: human-readable description of how to detect that this setup is needed.
- `auto_detect_on_error`: error code that triggers the guide suggestion. Deferred to v0.5 (field accepted on read but not wired into the execution pipeline).

## The `actions/<id>.yaml` File

One action = one file. Complete reference.

### Skeleton of a declarative action (simple REST)

```yaml
id: pages.read
permission: pages.read
summary: Retrieve a page object by ID
docs_url: https://developers.notion.com/reference/retrieve-a-page

request:
  method: GET
  path: /pages/{page_id}

inputs:
  page_id:
    type: string
    required: true
    description: UUID of the page (with or without dashes)
    pattern: "^[0-9a-f]{8}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{12}$"
    location: path           # path | query | body | header
  filter_properties:
    type: array
    items: { type: string }
    required: false
    description: Subset of property IDs to return
    location: query

output:
  type: object
  passthrough: true          # returns the JSON as-is

errors:
  404:
    code: not_found
    install_guide: share-page
    hint: "Page does not exist or integration lacks access."
  429:
    code: rate_limited
    retry: backoff
    max_attempts: 3
```

### Detailed fields

#### `id` (required)

Action identifier. Must match the filename (without `.yaml`).

#### `permission` (required)

Reference to a permission declared in `service.yaml > permissions`. One action = one permission. If the action logically requires multiple permissions, it probably needs to be split into two actions.

#### `summary` and `docs_url`

Human-readable description and link to the official endpoint documentation.

#### `request` (for declarative actions)

```yaml
request:
  method: GET                            # GET | POST | PUT | PATCH | DELETE | HEAD
  path: /pages/{page_id}                 # relative to api.base_url
  body: $inputs                          # or template object
  headers:
    Content-Type: application/json       # overrides global headers
  query:
    page_size: "{inputs.limit}"
```

- `path` can contain `{input_name}` placeholders that are substituted.
- `body: $inputs` serializes the inputs as JSON in the body.
- `body` can also be a template object (e.g. `{ data: "{inputs.payload}", meta: { source: "one" } }`).
- `headers` adds to/overrides the API's global headers.

#### `inputs`

Typed schema for the action's arguments. Validated before runtime invocation.

```yaml
inputs:
  page_id:
    type: string                         # string | integer | number | boolean | array | object | file_ref
    required: true                       # default: false
    description: ...
    pattern: "^[0-9a-f-]{36}$"           # regex for string
    enum: [low, medium, high]            # allowed values
    default: "medium"                    # value if not provided
    min: 0                               # for number/integer
    max: 100
    min_length: 1                        # for string
    max_length: 255
    location: path                       # path | query | body | header

  properties:
    type: object
    required: true
    schema:                              # free sub-schema
      type: object
      properties:
        title: { type: array }
        Status: { type: object }

  body:
    type: file_ref                       # reference to a local file
    description: Path to upload
```

**The `file_ref` type** is special: the user passes `--body @path/to/file.pdf`, the binary reads the file and passes it to the runtime as a `Uint8Array`.

#### `output`

Output schema. Validated after invocation. If invalid, it is a handler/runtime bug, not a user error.

```yaml
output:
  type: object
  passthrough: true                      # accepts any JSON
```

Or more strict:

```yaml
output:
  type: object
  schema:
    type: object
    required: [id, properties]
    properties:
      id: { type: string }
      properties: { type: object }
      url: { type: string, format: uri }
```

Or an array (e.g. `search`):

```yaml
output:
  type: array
  items:
    type: object
    schema: ...
```

#### `errors`

Mapping between HTTP codes/native service errors and a stable One CLI code + a hint.

```yaml
errors:
  404:
    code: not_found
    install_guide: share-page            # ref to a guide
    hint: "Page does not exist or integration lacks access."
  401:
    code: not_authenticated
    hint: "Run `one login notion` to authenticate."
  429:
    code: rate_limited
    retry: backoff                       # auto-retry with exponential backoff
    max_attempts: 3
    backoff_initial_ms: 1000
  validation_failed:                     # service error code (e.g. Stripe)
    code: invalid_input
    hint: "Check the inputs match the schema."
```

The `code:` values are the stable codes the agent sees in the JSON output. Must be documented. Stable across versions.

#### `side_effects`, `idempotency`, `dry_run`

For mutations:

```yaml
side_effects: write                      # read | write | destructive
idempotency:
  supported: true                        # true if the API natively supports idempotency
  key: header.Idempotency-Key            # where to pass the key
  required: false                        # if true, the idempotency_key input is required

dry_run:
  supported: true
  behavior: |
    Validates the request without sending. Returns what would have been sent.
```

#### `handler` (for WASM actions)

```yaml
handler: ./handlers/main.wasm
handler_entry: pages_create
host_api_version: 1

calls:                                   # allowlisted URLs
  - method: POST
    url: "https://api.notion.com/v1/pages"
  - method: POST
    url_pattern: "^https://api\\.notion\\.com/v1/pages/[a-f0-9-]+$"
```

See [HANDLERS.md](./HANDLERS.md) for the full WASM contract details.

#### Streaming

```yaml
streaming: true
```

Signals that the action returns a stream of objects (NDJSON) rather than a single object. Useful for aggregated paginated lists. The service skill must document this.

#### Automatic pagination (declarative action)

```yaml
pagination:
  style: cursor                          # cursor | page | offset
  request_param: cursor                  # name of the param passed in the request
  request_location: query
  response_token: next_cursor            # field in the response
  response_has_more: has_more
  max_pages: 50                          # safety cap
```

The declarative runtime handles the loop automatically and aggregates results. The agent calls the action without worrying about pagination.

## The Service `SKILL.md` File

Markdown intended for AI agents. Read via `one info <service>`.

**Required frontmatter**:

```markdown
---
service: notion
version: 1.4.0
summary: Read and write pages, databases, and blocks in a Notion workspace
use_when:
  - "User wants to log, track, or document something in Notion"
  - "User asks to read or update a Notion page/database"
avoid_when:
  - "User wants a quick local note (use filesystem instead)"
---
```

**Recommended structure**:

1. **Mental model** (3-5 sentences): the concepts unique to this service.
2. **Required setup**: what must be done before first use.
3. **Typical workflow**: 2-3 common patterns with command examples.
4. **Gotchas**: 3-5 pitfalls specific to this service.
5. **Common chains**: recipes for composite use cases.
6. **Permissions**: list of typical permissions with a link to `one scope add`.

**Target length**: 100-200 lines. No more — the skill must remain scannable.

Minimal example:

```markdown
---
service: stripe
version: 3.0.2
summary: Read and write customers, subscriptions, charges, refunds via Stripe API
use_when:
  - "User wants to query Stripe data (customers, charges)"
  - "User wants to create or update billing entities"
---

# Stripe

## Mental model

Stripe organizes everything around `Customer`, `PaymentMethod`,
`Subscription`, `Charge`, and `Invoice`. Customer is central: most
mutations attach to a customer.

## Setup

Stripe uses API keys (no OAuth). Get one from
https://dashboard.stripe.com/apikeys. Test mode keys start with
`sk_test_`, live mode with `sk_live_`. Use different accounts via:

  one login stripe          → choose test or live key

## Typical workflows

**Lookup a customer:**
  one stripe customers.search --query "email:'user@example.com'"

**Create a customer:**
  one stripe customers.create --email user@example.com --name "Jane"

**List recent charges:**
  one stripe charges.list --limit 10

## Gotchas

- **Idempotency is critical**. Always pass `--idempotency_key=...`
  when creating customers, charges, or subscriptions, to avoid
  duplicate side effects on retry.
- **Test vs live mode**. Stripe data is fully isolated. A customer
  created in test mode is invisible from a live mode account.
- **Pagination**. Use `--starting_after=<id>` for cursor pagination.
  One CLI auto-paginates `*.list_all` variants.

## Permissions

- `customers.read`, `customers.write`
- `charges.read`, `charges.write`
- `subscriptions.read`, `subscriptions.write`
- `webhooks.read`, `webhooks.write`

Add via `one scope add stripe <permission>`.
```

## Install Guides (`guides/<id>.md`)

Markdown with frontmatter for setups requiring a human.

```markdown
---
id: share-page
title: Share a Notion page with your integration
estimated_time: 30s
requires_human: true
requires:
  - authenticated: true
  - capability: pages.read
verify:
  action: pages.read
  inputs:
    page_id: "${PROMPT:Paste the page ID after sharing}"
  expect_success: true
related_errors:
  - not_found
  - object_not_found
applies_to:
  permissions: ["pages.*", "blocks.*", "databases.*"]
open_url: https://notion.so
---

Notion's permission model is opt-in per page or database. After the
initial OAuth flow, the integration sees nothing until pages are
explicitly shared with it.

## What to do

1. Open the page or database in Notion
2. Click `•••` (top right) → `Connections`
3. Search for your integration name and click it
4. Confirm

## Verify

Run `one install notion share-page --verify` and paste a page ID to
check that the share worked.

## If it still fails

Check that the integration has the required capabilities at
https://www.notion.so/profile/integrations.
```

**Frontmatter fields**:

| Field | Type | Description |
|---|---|---|
| `id` | string | guide identifier, matches the filename |
| `title` | string | human-readable title |
| `estimated_time` | string | "30s", "2m", "10m" |
| `requires_human` | bool | true if a human is required, false if automatable |
| `requires` | object | preconditions (authenticated, capabilities) |
| `verify` | object | how to verify that the setup worked |
| `related_errors` | array | error codes this guide resolves |
| `applies_to` | object | which permissions this guide applies to |
| `open_url` | URL | URL to open when the user presses `[o]` |
| `auto_install` | object | for non-human guides, the action to execute |

**Automatable guides**: `auto_install` deferred to v0.5. In v0.4, `one install <service> <guide>` displays the markdown and prints the `verify` command if defined.

## Catalog Contribution Process

### 1. Fork the repo

```
git clone https://github.com/one-cli/catalog
```

### 2. Scaffold

```bash
one catalog scaffold my-service --lang ts
# creates services/my-service/{service.yaml, handlers/main.ts, package.json, tests/}
```

Or manually, by copying `services/_template/`.

### 3. Implement

- Fill in `service.yaml`
- Write the YAML actions (and WASM handler if needed)
- Write `SKILL.md`
- Write the necessary guides

### 4. Test locally

```bash
# Lint
one catalog lint my-service

# Build WASM if applicable
cd services/my-service && bun run build

# Handler tests
bun test

# Integration test
one catalog test my-service          # invokes the one binary in local mode
```

### 5. Open a PR

Pre-filled PR template. Checklist:

- [ ] `service.yaml` passes lint
- [ ] All actions have a valid inputs schema
- [ ] SKILL.md follows the recommended structure
- [ ] At least one install guide (even just `initial-setup`)
- [ ] Handler tests pass
- [ ] URL allowlist is consistent with what the handler actually does
- [ ] Declared permissions match the credentials used

### 6. Review and merge

Catalog maintainers review. Criteria:

- **Security**: no hardcoded credentials, strict URL allowlist, no destructive writes without warning
- **Skill quality**: understandable by an agent unfamiliar with the service
- **Robustness**: handling of common errors, actionable hints
- **Consistency**: permission naming, input structure

Once merged, CI automatically publishes the new version to the index.

## Service Versioning

Each service has a SemVer `version` in its `service.yaml` (in addition to `version: 1` for the format).

```yaml
version: 1
name: notion
service_version: 1.4.0
```

Versions:

- **Patch** (1.4.0 → 1.4.1): bug fix in a handler, doc clarification, adjusted hint
- **Minor** (1.4.0 → 1.5.0): new action added, new permission, new auth provider
- **Major** (1.4.0 → 2.0.0): breaking change (renamed action, changed input signature, renamed error code)

Users pin the version in `.onerc.lock`. A major bump requires a manual action (`one lock --update notion`).

## Security and Review

### What is forbidden in a service.yaml

- `http://` URLs (except localhost for tests)
- Hardcoded credentials (`access_token: "sk_..."`)
- Headers containing references to `${env.SECRET_*}` (use the credentials mechanism)
- Permissions without a `description`
- Actions without an `errors` mapping (at minimum for 401/403/404/429)

### What is verified automatically

- JSON Schema validation on the YAML
- Each action references a declared permission
- Each guide referenced in `required_setup` exists
- Each error code referenced in guides exists in an action
- For WASM: URLs hit ⊆ declared `calls:`
- For WASM: credentials read ⊆ declared `credentials:`
- For WASM: error codes returned ⊆ declared `errors:`

### Reviewers

For high-impact services (complex auth, sensitive sectors, large catalogs): double review. For simple services: single review is sufficient.

## Complete Examples

See the `one-cli/catalog/examples/` repo for:

- `simple-rest/`: trivial service with API key + GET/POST
- `oauth-paginated/`: OAuth + cursor pagination
- `wasm-graphql/`: GraphQL via WASM
- `wasm-sigv4/`: AWS request signing via WASM
- `multi-account/`: service with complex multi-accounts

Each example is commented to serve as a reference.

---

*To propose an evolution of the `service.yaml` format, open an RFC in `one-cli/rfcs` rather than a direct PR on the catalog.*
