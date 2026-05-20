# One CLI Handler SDKs

Two official SDKs ship with the binary:

- `ts/` — `@one-cli/handler-sdk-ts`, compiled to wasm via [Javy](https://github.com/bytecodealliance/javy).
- `go/handler` — tinygo (`-target=wasi`) SDK.

Both expose the same host surface: `creds`, `http`, `log`, `time`, `fail`, plus
crypto primitives. Every host capability is bounded by the per-action allowlists
declared in `service.yaml > actions.<id>.handler`:

```yaml
handler:
  file: handler.wasm
  sha256: <hex>
  host_api_version: 1
  calls:        ["^https://api\\.notion\\.com/v1/pages$"]
  credentials:  ["access_token"]
  fail_codes:   ["not_found", "api_error"]
```

## Testing without WASM

- **Go**: `handler/handlertest.FakeHost` installs in-process bridges so handler
  logic runs as a plain Go test (no tinygo required). See `handler_test.go`.
- **TS**: `test/fake-host.ts` outlines the shim recipe; vitest's
  `vi.stubGlobal` mocks the host imports for unit tests.

## Building

```bash
# Go (tinygo)
tinygo build -o handler.wasm -target=wasi -no-debug ./main.go

# TS (Javy)
npm run build   # bundles src then runs scripts/build-wasm.sh
```

## Host API version

Pinned in `internal/adapters/runtime/wazero.go > HostAPIVersion`. Bump only on
breaking ABI changes; handlers are refused at load time when they request a
different major. See `docs/HANDLERS.md` for the full spec.
