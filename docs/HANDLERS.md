# HANDLERS.md

> Complete reference for writing WASM handlers used by One CLI catalog services. For the `service.yaml` format, see [CATALOG.md](./CATALOG.md). For the threat model, see [SECURITY.md](./SECURITY.md).

## Why WASM handlers

Declarative YAML covers 95% of cases (simple REST, Bearer auth, body passthrough). WASM handlers exist for the remaining 5%: request signing, multiple chained calls, GraphQL with typed interpolation, non-trivial response transformation, complex automatic pagination.

**Choosing between YAML and WASM**:

| Case | Pure YAML | WASM |
|---|:---:|:---:|
| One endpoint, body passthrough, fixed headers | ✓ | |
| Bearer auth, API key in a header | ✓ | |
| Simple cursor pagination (param + token) | ✓ | |
| Computed headers (hash, signature, JWT) | | ✓ |
| ≥ 2 HTTP requests per action | | ✓ |
| Response transformation beyond passthrough | | ✓ |
| GraphQL with input interpolation in the query | | ✓ |
| Conditional logic (if X then call Y else Z) | | ✓ |
| Rollback on partial failure | | ✓ |

When in doubt, **start with YAML**. Migrate to WASM only if the declarative option is not achievable.

## Isolation model

A handler runs in a WASM module via [wazero](https://wazero.io/). Minimal WASI, **nothing is exposed by default**:

- No filesystem
- No env vars
- No direct network
- No direct clock
- No direct random
- No process exec
- No uncontrolled stdin/stdout

The handler can only interact with the outside world via **host functions** explicitly exposed by the One CLI binary.

This isolation is what makes the catalog *community-safe*: a `stripe.wasm` handler cannot read `~/.ssh/`, exfiltrate the vault, or make an arbitrary HTTP call to an external server.

## The invocation flow

```
┌──────────────┐   1. invoke handler          ┌─────────────────┐
│   One CLI    │ ──────────────────────────►  │  WASM handler   │
│   (host)     │   inputs (JSON)              │  (sandboxed)    │
│              │                              │                 │
│              │  ◄── 2. host functions ──── │                 │
│              │      creds.get               │                 │
│              │      http.request            │                 │
│              │      crypto.*                │                 │
│              │      time.*                  │                 │
│              │      log.*                   │                 │
│              │      fail                    │                 │
│              │                              │                 │
│              │  ──── 3. responses ───────► │                 │
│              │                              │                 │
│              │  ◄── 4. return ────────────  │                 │
│              │      output (JSON)           │                 │
│              │      or structured error     │                 │
└──────────────┘                              └─────────────────┘
```

## The I/O contract

### What the handler receives (on invocation)

```json
{
  "action": "pages.create",
  "inputs": {
    "parent": { "page_id": "abc-123" },
    "properties": { "title": [...] }
  },
  "config": {
    "api_version": "2025-09-03"
  },
  "context": {
    "account": "kaampus",
    "dry_run": false,
    "trace_id": "01HXYZ..."
  }
}
```

**Credentials are not in this payload.** The handler requests them via `host.creds.get`.

### What the handler returns

Either an arbitrary JSON object (validated against `output:` in the YAML if defined):

```json
{
  "id": "page_abc",
  "url": "https://notion.so/...",
  "created_at": "2026-05-20T..."
}
```

Or terminates via `host.fail.withCode(...)` (equivalent to a structured throw).

## Host functions

### `host.creds`

```ts
namespace host.creds {
  /**
   * Retrieves a credential by its logical name (declared in service.yaml > credentials).
   * Fails if the key is not declared.
   * The returned value is marked sensitive and redacted in logs.
   */
  function get(key: string): string;
}
```

**Example**:

```ts
const token = host.creds.get('access_token');
// throws if not declared in service.yaml > credentials
```

### `host.http`

```ts
namespace host.http {
  function request(req: HttpRequest): HttpResponse;
}

interface HttpRequest {
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE' | 'HEAD';
  url: string;
  headers?: Record<string, string>;
  body?: string | Uint8Array;
  timeout_ms?: number;        // default 30000, max 120000
  follow_redirects?: boolean; // default true
  max_redirects?: number;     // default 5, max 10
}

interface HttpResponse {
  status: number;
  headers: Record<string, string>;
  body: Uint8Array;
}
```

**Mandatory allowlist**. Before invocation, the host parses `service.yaml > actions > calls:` and builds a list of allowed URL patterns. If the handler calls `host.http.request` with a URL that does not match any pattern, the host fails with `url_not_allowed`.

**Auto-injection of standard credentials**. For common auth schemes (Bearer, Basic), if `service.yaml > auth.providers.X.injection` declares `inject: auto`, the host adds the header automatically. The handler does not need to touch the token. Increased security.

For custom schemes (SigV4, JWT, signing), `inject: manual` and the handler forges the header itself via `host.creds.get` + `host.crypto.*`.

**Audit log**. Each request is logged (method, host, path, status, duration). Not the body. Visible via `one trace`.

### `host.crypto`

**Primitives only**, no high-level "encrypt(data, password)".

```ts
namespace host.crypto {
  function sha256(data: Uint8Array): Uint8Array;
  function sha512(data: Uint8Array): Uint8Array;
  function hmacSha256(key: Uint8Array, data: Uint8Array): Uint8Array;
  function hmacSha512(key: Uint8Array, data: Uint8Array): Uint8Array;
  function randomBytes(n: number): Uint8Array;
  function uuidV4(): string;
  function base64Encode(data: Uint8Array, urlSafe?: boolean): string;
  function base64Decode(s: string, urlSafe?: boolean): Uint8Array;
  function hexEncode(data: Uint8Array): string;
  function hexDecode(s: string): Uint8Array;
}
```

Sufficient for all modern auth schemes (SigV4, JWT signing, webhook verification). The rest of the logic is handled in pure code inside the handler.

### `host.time`

```ts
namespace host.time {
  function now(): number;          // unix ms
  function sleep(ms: number): void; // blocking, capped at 30s per call, 60s cumulative
}
```

No monotonic clock access. No timezone. If the handler needs an ISO 8601 string, it builds it itself from the timestamp.

### `host.log`

```ts
namespace host.log {
  function debug(msg: string, attrs?: Record<string, unknown>): void;
  function info(msg: string, attrs?: Record<string, unknown>): void;
  function warn(msg: string, attrs?: Record<string, unknown>): void;
}
```

No `error`: use `host.fail`.

Handler-side logs appear in One CLI's structured output when `--debug` or `--trace` is active. Otherwise silently ignored.

**Do not spam.** Cap of 1000 log lines per invocation.

### `host.fail`

```ts
namespace host.fail {
  function withCode(code: string, message: string, hint?: string): never;
}
```

Terminates execution with a structured error. The `code` **must** match an entry in the action's YAML `errors:` block. Otherwise, the host maps it to `unknown_error` and emits a warning to the reviewer.

```ts
const res = host.http.request({ method: 'GET', url: '...' });
if (res.status === 404) {
  host.fail.withCode(
    'not_found',
    'Page does not exist or integration lacks access',
    'one install notion share-page'
  );
}
```

The agent receives on the CLI side:

```json
{
  "error": {
    "code": "not_found",
    "message": "Page does not exist or integration lacks access",
    "hint": "one install notion share-page",
    "install": {
      "service": "notion",
      "guide": "share-page",
      "requires_human": true
    }
  }
}
```

## Lifecycle

**One invocation = one instantiated module.** No reuse between calls. Slower (~10ms overhead for wazero AOT-compiled), but isolates state bugs.

**No `main()` function.** The handler exports one or more functions by name, dispatched via `handler_entry` in the YAML:

```ts
// handler.ts compiled to handler.wasm
export function upload(inputs: UploadInputs): UploadOutput {
  // ...
}

export function bucket_create(inputs: BucketCreateInputs): BucketCreateOutput {
  // ...
}
```

The host invokes `handler.upload(deserialize(stdin_json).inputs)`. Signatures are typed via YAML schemas: the SDK generates types from `service.yaml`.

### Resource limits

```
memory:     64 MB by default, max 256 MB (configurable via service.yaml)
cpu time:   30s wall-clock per invocation (max 120s)
stack:      1 MB
http calls: 50 max per invocation
log lines:  1000 max per invocation
sleep:      30s per call, 60s cumulative
```

Beyond these limits, hard kill and `resource_exhausted` error on the agent side.

## SDKs by language

Three officially supported toolchains at launch.

### TypeScript (recommended for getting started)

Compiled via [Javy](https://github.com/bytecodealliance/javy) (QuickJS embedded in WASM).

```bash
bun add -d @one-cli/handler-sdk-ts
bun add -d @bytecodealliance/javy
```

**`handlers/main.ts`**:

```ts
import { host } from '@one-cli/handler-sdk-ts';
import type { Inputs, Output } from './generated/pages-create.types';

export function pages_create(inputs: Inputs): Output {
  host.log.info('Creating Notion page', { parent: inputs.parent });

  const res = host.http.request({
    method: 'POST',
    url: 'https://api.notion.com/v1/pages',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      parent: inputs.parent,
      properties: inputs.properties,
      children: inputs.children,
    }),
  });

  if (res.status === 404) {
    host.fail.withCode(
      'not_found',
      'Parent page not accessible',
      'one install notion share-page'
    );
  }

  if (res.status >= 400) {
    const body = JSON.parse(new TextDecoder().decode(res.body));
    host.fail.withCode('api_error', body.message || 'Unknown error');
  }

  return JSON.parse(new TextDecoder().decode(res.body));
}
```

**Build**:

```bash
bun run build
# generates handlers/main.wasm
```

**Tests** (with fake host):

```ts
import { test, fakeHost } from '@one-cli/handler-test';
import { pages_create } from './main';

test('pages_create POSTs to /v1/pages', async () => {
  const host = fakeHost();
  host.creds.set('access_token', 'secret_token');
  host.http.expect({
    method: 'POST',
    url: 'https://api.notion.com/v1/pages',
    headers: { 'Content-Type': 'application/json' },
  }).respond({
    status: 200,
    body: { id: 'page_123', url: '...' },
  });

  const result = pages_create({
    parent: { page_id: 'parent_abc' },
    properties: { title: [{ text: { content: 'Test' } }] },
  });

  expect(result.id).toBe('page_123');
});
```

### Go (via tinygo)

```bash
tinygo build -o handlers/main.wasm -target=wasi handlers/main.go
```

**`handlers/main.go`**:

```go
package main

import (
    "encoding/json"
    "one.dev/handler"
)

//export pages_create
func pages_create() {
    var inputs struct {
        Parent     map[string]string      `json:"parent"`
        Properties map[string]interface{} `json:"properties"`
    }
    handler.ReadInputs(&inputs)

    body, _ := json.Marshal(inputs)
    res, err := handler.Host.HTTP.Request(handler.HTTPRequest{
        Method: "POST",
        URL:    "https://api.notion.com/v1/pages",
        Headers: map[string]string{
            "Content-Type": "application/json",
        },
        Body: body,
    })
    if err != nil {
        handler.Host.Fail("api_error", err.Error(), "")
    }

    if res.Status == 404 {
        handler.Host.Fail("not_found", "Parent not accessible",
            "one install notion share-page")
    }

    var out map[string]interface{}
    json.Unmarshal(res.Body, &out)
    handler.WriteOutput(out)
}

func main() {} // required by tinygo
```

### Rust (for performance-critical needs)

```bash
cargo build --target wasm32-wasi --release
```

**`src/lib.rs`**:

```rust
use one_handler::{host, Inputs, Output};

#[no_mangle]
pub extern "C" fn pages_create() {
    let inputs: Inputs = host::read_inputs();
    let token = host::creds::get("access_token");

    let body = serde_json::to_vec(&inputs).unwrap();
    let res = host::http::request(host::HttpRequest {
        method: "POST".into(),
        url: "https://api.notion.com/v1/pages".into(),
        headers: [("Content-Type", "application/json")].into(),
        body: Some(body),
        ..Default::default()
    });

    if res.status == 404 {
        host::fail::with_code("not_found", "Parent not accessible",
            Some("one install notion share-page"));
    }

    let out: serde_json::Value = serde_json::from_slice(&res.body).unwrap();
    host::write_output(&out);
}
```

## Common patterns

### Bearer auth with validation

```ts
const token = host.creds.get('access_token');
const res = host.http.request({
  method: 'GET',
  url: 'https://api.service.com/v1/me',
  headers: { Authorization: `Bearer ${token}` },
});
```

Often unnecessary if `injection: auto` is configured.

### SigV4 signing for AWS

```ts
import { signSigV4 } from '@one-cli/handler-sdk-ts/aws';

const ak = host.creds.get('access_key_id');
const sk = host.creds.get('secret_access_key');
const region = host.creds.get('region');

const signed = signSigV4({
  method: 'PUT',
  url: `https://${inputs.bucket}.s3.${region}.amazonaws.com/${inputs.key}`,
  headers: { 'x-amz-content-sha256': '...' },
  body: inputs.body,
  service: 's3',
  region,
  credentials: { accessKeyId: ak, secretAccessKey: sk },
});

const res = host.http.request(signed);
```

The `signSigV4` helper is provided by the SDK and uses `host.crypto.*` internally.

### Call chains with rollback

```ts
// stripe customer.full-create : 3 calls with best-effort rollback
export function customer_full_create(inputs: CustomerFullCreateInputs) {
  const apiKey = host.creds.get('api_key');
  const headers = { Authorization: `Bearer ${apiKey}` };

  // 1. Create customer
  const cust = postJSON('https://api.stripe.com/v1/customers', {
    email: inputs.email,
  }, { 'Idempotency-Key': inputs.idempotency_key });

  try {
    // 2. Attach payment method
    postJSON(`https://api.stripe.com/v1/payment_methods/${inputs.payment_method_id}/attach`, {
      customer: cust.id,
    });

    // 3. Set default
    postJSON(`https://api.stripe.com/v1/customers/${cust.id}`, {
      invoice_settings: { default_payment_method: inputs.payment_method_id },
    });

    return { customer_id: cust.id };
  } catch (e) {
    // Best-effort rollback
    try {
      host.http.request({
        method: 'DELETE',
        url: `https://api.stripe.com/v1/customers/${cust.id}`,
        headers,
      });
    } catch {}

    host.fail.withCode('partial_failure',
      'Customer was created but setup failed; rolled back',
      `Customer ID: ${cust.id}`);
  }
}
```

### Auto pagination

```ts
export function* messages_list_all(inputs: ListAllInputs) {
  let cursor: string | undefined;
  let count = 0;
  while (count < inputs.max_total) {
    const url = new URL('https://gmail.googleapis.com/gmail/v1/messages');
    url.searchParams.set('q', inputs.query);
    if (cursor) url.searchParams.set('pageToken', cursor);

    const res = host.http.request({ method: 'GET', url: url.toString() });
    const page = JSON.parse(new TextDecoder().decode(res.body));

    for (const msg of page.messages || []) {
      yield msg;
      count++;
      if (count >= inputs.max_total) return;
    }

    if (!page.nextPageToken) return;
    cursor = page.nextPageToken;
  }
}
```

The SDK handles serializing the generator to NDJSON for the host.

## The `service.yaml > calls:` (URL allowlist)

This is the mechanism that makes the sandbox real. Without an allowlist, a malicious handler could exfiltrate credentials to a third-party server.

```yaml
calls:
  - method: POST
    url: "https://api.notion.com/v1/pages"
  - method: GET
    url_pattern: "^https://api\\.notion\\.com/v1/pages/[a-f0-9-]+$"
  - method: PATCH
    url_pattern: "^https://api\\.notion\\.com/v1/pages/[a-f0-9-]+$"
```

**Supported patterns**:

- `url`: exact match
- `url_pattern`: regex (validated at registration)

**Best practices**:

- Use patterns as specific as possible. Avoid `^https://api\\.notion\\.com/.*` which allows everything.
- Listed at fine granularity (one endpoint = one entry), to ease review.
- HTTP method always explicit.

**Verified in CI**: a linter statically analyzes the handler and verifies that all URLs hit by the code match at least one pattern.

## Host contract versioning

The `host_api_version` in service.yaml declares the expected host API version:

```yaml
host_api_version: 1
```

The binary checks at load time. If the handler expects v2 but the binary only exposes v1, it refuses to load with a clear message: "upgrade `one` to use this service".

Allows the host contract to evolve without breaking old handlers. Compatible changes (adding functions) do not bump the major version; incompatible changes (renaming, changed signature) are rare but do bump it.

## Distribution

Compiled by the catalog repo's CI, bundled with the service tarball. SHA-256 hash in the index, verified at download by `one`. The handler source is in the repo, so it is auditable, and the binary is reproducible.

**Contributors never commit the `.wasm`.** CI builds it. This prevents drift between source and binary.

## Tests

### Recommended pattern

For each handler:

1. One "happy path" test: valid input, mock the HTTP, verify the result.
2. One test per error code declared in `errors:`: assert that `host.fail.withCode` is called with the correct code.
3. One test for each non-trivial conditional branch in the handler.

### Example

```ts
import { test, fakeHost } from '@one-cli/handler-test';
import { pages_create } from './main';

test('happy path', async () => {
  const host = fakeHost();
  host.creds.set('access_token', 'tok');
  host.http.expect({
    method: 'POST',
    url: 'https://api.notion.com/v1/pages',
  }).respond({ status: 200, body: { id: 'page_123' } });

  const result = pages_create({ parent: { page_id: 'p' }, properties: {} });
  expect(result.id).toBe('page_123');
});

test('404 maps to not_found with hint', async () => {
  const host = fakeHost();
  host.creds.set('access_token', 'tok');
  host.http.expect({}).respond({ status: 404, body: { message: 'Not found' } });

  expect(() => pages_create({ parent: { page_id: 'p' }, properties: {} }))
    .toThrow(/not_found/);
  expect(host.fail.lastCall).toMatchObject({
    code: 'not_found',
    hint: 'one install notion share-page',
  });
});
```

### Lint in CI

The catalog repo runs static lint on every PR:

- All URLs hit by the handler match an entry in `calls:`
- All `host.creds.get(key)` calls match an entry in `credentials:`
- All `host.fail.withCode(code, ...)` calls match an entry in `errors:`
- No forbidden `import` (no `fs`, `child_process`, etc.)

This is what allows the community to contribute without breaking things. The linter catches 80% of errors before code review.

## Anti-patterns

### Storing a token in a global variable

```ts
// BAD
let cachedToken: string | null = null;
export function action(inputs) {
  if (!cachedToken) cachedToken = host.creds.get('access_token');
  // ...
}
```

One invocation = one fresh module. The cache does not survive and is a source of confusion. Always call `host.creds.get` on each invocation.

### Making undeclared HTTP calls

```ts
// BAD: crashes at runtime with url_not_allowed
host.http.request({ url: 'https://my-personal-server.com/log' });
```

All URLs must be in `service.yaml > calls:`.

### Logging secrets

```ts
// BAD
host.log.info('Auth', { token: host.creds.get('access_token') });
```

`Secret` values on the host side are redacted, but values returned by `host.creds.get` on the handler side are plain strings. The host has no reliable way to detect the leak. **Never log what you receive from `creds.get`.**

### Bypassing auto injection

```yaml
# service.yaml
auth:
  providers:
    oauth:
      injection:
        header: Authorization
        format: "Bearer {access_token}"
        inject: auto
```

```ts
// BAD: auto injection is already active, you are duplicating it
const token = host.creds.get('access_token');
host.http.request({
  url: '...',
  headers: { Authorization: `Bearer ${token}` },
});
```

With `inject: auto`, the handler does not need to touch the token. Simpler and safer.

### Not handling errors

```ts
// BAD: the handler returns a raw crash instead of a typed error
const res = host.http.request({ ... });
return JSON.parse(new TextDecoder().decode(res.body));
// If res.status != 200, the JSON may be invalid
```

Always branch on the status code and call `host.fail.withCode` with a mapped code.

## Performance

### Handler cold start

A WASM module is compiled on first use and then cached. The cache is shared between invocations within the same process, but not across processes (each `one ...` is a fresh process).

Typical handler cold start: 10-30ms (compilation) + 5ms (instantiation). Acceptable for most cases. If critical, the binary can pre-compile frequent handlers to AOT (`wazero compile`) during `one catalog update`.

### Memory footprint

A simple handler uses ~5-10 MB. A complex handler with heavy string manipulation can reach 30-50 MB. The default 64 MB cap is generous.

If a handler approaches 64 MB, it is likely a bug (leak, loop, accumulation). Raising the cap is rarely the right fix.

### Throughput

Not the bottleneck for a CLI. If a handler does 10ms of logic + 200ms of HTTP latency, the total is dominated by HTTP. WASM adds <5ms of overhead.

## Security (summary)

Details are in [SECURITY.md](./SECURITY.md). Key points for a handler author:

1. **Strict URL allowlist.** No escape possible.
2. **Credentials read only via `host.creds.get`.** Declared in `service.yaml > credentials`.
3. **No I/O outside host functions.** No filesystem, no env vars, no exec.
4. **Typed outgoing error codes.** No string-matching on the user side.
5. **No logging of secrets.** Ever.

If you follow these rules, your handler can be merged without concern.

---

*To propose an evolution of the host contract (new function, new type), open an RFC in `one-cli/rfcs`. Any evolution must be backward-compatible with existing handlers.*
