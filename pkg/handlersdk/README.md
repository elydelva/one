# One Handler SDK

> SDKs for writing One CLI WASM handlers in Go, TypeScript, and Rust.

## Status

Planned for Phase 3. Not yet implemented.

## Overview

Handlers are WASM modules that implement complex API interactions that cannot be expressed in the declarative YAML format. They run inside a sandboxed wazero environment with access to a set of host functions.

## Host Functions (planned)

| Function | Description |
|---|---|
| `host.creds.get(service, account)` | Get credentials for a service account |
| `host.http.fetch(method, url, headers, body)` | Make an allowlisted HTTP request |
| `host.crypto.random_bytes(n)` | Get n random bytes |
| `host.time.now()` | Get current Unix timestamp |
| `host.log(level, message)` | Emit a structured log line |
| `host.fail(code, message)` | Fail with a typed error code |

## Language Support (planned)

- **Go**: via TinyGo
- **TypeScript**: via Extism JS PDK
- **Rust**: via wasm-bindgen + wasi

See [HANDLERS.md](../../docs/HANDLERS.md) for the full specification.
