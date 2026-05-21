# One CLI

> A local binary that gives your AI agents unified, governed, and auditable access to third-party services.

```bash
one notion pages.create --parent_page_id abc-123 --properties '{"title":[{"text":{"content":"Hello"}}]}'
```

No need to learn ten SDKs. No need to manage ten vaults. No need to code ten auth workflows. A single binary, a community catalog, and a versioned `.onerc.yaml` file that says exactly what your agents are allowed to do on your project.

## In 30 seconds

- **One binary**: cross-platform Go, single-line install, cold start <30ms.
- **One catalog**: open source services, contributed via PR (~40 services at launch).
- **One vault**: local multi-account: native OS keychain, never SaaS, multi-provider (OAuth, PAT, API key, AWS).
- **One scope file**: versioned `.onerc.yaml` that makes what an agent can do explicit, reviewable, and default-deny.
- **First-class install guides**: for setups that can't be automated (e.g. "share this Notion page with the integration").
- **A local audit log**: `one trace` to see what was done.

## Quick start

### Install

```bash
curl -fsSL https://one-cli.dev/install.sh | sh
one --version
```

Or via Homebrew:

```bash
brew install one-cli/tap/one
```

### Project setup

```bash
cd my-project
one init                              # creates .onerc.yaml

one login github                      # OAuth flow, browser
one login notion --as kaampus         # OAuth, alias for multi-account

one scope add github "issues.*"       # globs supported
one scope add github pulls.read
one scope add notion "pages.*"

one lock                              # pins catalog versions
git add .onerc.yaml .onerc.lock
git commit -m "init: setup One CLI"
```

### Run an action

```bash
one github issues.create \
  --repo me/myrepo \
  --title "Bug: foo" \
  --body "Detailed description"
```

JSON output:

```json
{
  "ok": true,
  "data": { "id": 42, "url": "https://github.com/me/myrepo/issues/42" },
  "trace_id": "01HXYZ..."
}
```

### Workflow for an AI agent

```bash
one capabilities --scope-only         # what can I do?
one info github                       # how does it work?
one can github issues.create          # is this in scope?
one github issues.create ...          # execute
```

See [`one skill`](./CLI.md#one-skill) to install the skill in Claude Code, Cursor, etc.

## Vs MCP

| Dimension | MCP | One CLI |
|---|---|---|
| Form | HTTP/stdio server per service | One local binary |
| Auth | Managed by each server | Unified multi-account vault |
| Per-project scope | Not covered | Versioned `.onerc.yaml` |
| Human setup | Not covered | First-class `one install` |
| Distribution | One package per server | Centralized reviewable catalog |
| Audit | Per-server logs | Unified audit log `one trace` |

One CLI is not anti-MCP. Both coexist. The core difference is **per-project governance** via scope file and the **unified local vault**.

## Documentation

| Document | For |
|---|---|
| [DESIGN.md](./DESIGN.md) | Understanding the **why**: pitch, positioning, concepts, philosophy |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Contributing to the Go binary: layout, ports & adapters, patterns |
| [CATALOG.md](./CATALOG.md) | Adding a service to the catalog: service.yaml format, PR process |
| [HANDLERS.md](./HANDLERS.md) | Writing a WASM handler: contract, host functions, TS/Go/Rust SDKs |
| [SCOPE.md](./SCOPE.md) | Mastering the scope file: grammar, layering, lock file |
| [AUTH.md](./AUTH.md) | Understanding auth: providers, OAuth flows, vault, multi-account |
| [SECURITY.md](./SECURITY.md) | Threat model, WASM sandbox, disclosure policy |
| [TESTING.md](./TESTING.md) | Test strategy: pyramid, contract tests, fakes vs mocks |
| [CLI.md](./CLI.md) | Command reference, exit codes, env vars |
| [CONTRIBUTING.md](./CONTRIBUTING.md) | First setup, workflow, code style |

## Supported services

GitHub, Notion, Linear, Stripe, Resend, Slack, Google Drive, Google Calendar, Gmail, AWS S3, Cloudflare, OpenAI, Anthropic, Vercel, Supabase, PostgreSQL, Discord, Twilio, SendGrid, HubSpot, Airtable, Asana, Trello, Figma, Bitbucket, GitLab, Sentry, PagerDuty, Datadog, MongoDB Atlas, Mailchimp, Calendly, Zoom, Microsoft Graph, Reddit, Twitter/X, Webflow, Shopify.

Full catalog: `one catalog search` or https://one-cli.dev/catalog.

## Status

**v1.0** — stable release. Open an issue or contribute if something frustrates you.

## License

Apache 2.0. See [LICENSE](./LICENSE).

## Credits

Designed and developed by [@elydelva](https://github.com/elydelva). Inspired by: Homebrew formulae (catalog model), [Inngest](https://inngest.com) (agent-native philosophy), MCP (positioning), git (local versioning philosophy).

Contributions welcome: see [CONTRIBUTING.md](./CONTRIBUTING.md).

---

*Questions? Ideas? [GitHub Discussions](https://github.com/one-cli/one/discussions).*
