# DESIGN.md

> The **north star** of the project. This document describes *why* One CLI exists, *what* it does, and *which principles* guide trade-offs when a choice arises. If a future contributor reads only one document, this is it.

## Pitch in 30 seconds

**One CLI is a unified abstraction layer between AI agents (or humans) and third-party services.** Instead of learning N SDKs and managing N authentication models, the agent calls `one <service> <action>`. The binary handles auth, enforces project-defined permissions, executes the action, and returns clean JSON. If the action falls outside its scope, it returns an actionable guide.

Three durable differentiators:

1. **A local multi-account vault** encrypted in the OS keychain, never in SaaS.
2. **A versioned scope file** in the repo, making explicit and reviewable what an agent is allowed to do on that specific project.
3. **First-class install guides** that honestly materialize the boundaries between what is automatable and what requires a human.

## The problem

AI agents need to act in the real world. Today they do so in three ways, all imperfect:

**1. Via specific SDKs (OpenAI tools, Anthropic tools).** Each agent re-implements calls to GitHub, Stripe, Notion. No governance, no reuse, credentials managed ad hoc by each dev (often scattered in env vars).

**2. Via MCP (Model Context Protocol).** Promising standardization, but server-oriented, without a per-project scope model, without a unified multi-account vault, without handling human-required operations. And MCP tends to push toward "everything-in-a-container" while many devs prefer a local binary under their direct control.

**3. By writing code every time.** The agent generates `curl` or fetch, the human hands over their tokens. Fragile, dangerous, non-auditable.

**What is missing, and nobody does well today**:

- A single local binary that speaks to N services with a unified model.
- A file versioned in the repo that explicitly states "on this project, the agent can do X but not Y".
- A clean mechanism for setups that cannot be automated (e.g. "share this Notion page with the integration"), instead of incomprehensible error loops.
- An open source catalog of services, in the style of Homebrew formulae, where the community adds its integrations.

One CLI is the intersection of these four needs.

## Positioning vs MCP

MCP is the closest conceptual alternative. The question "why not just use MCP?" must have a clear answer.

| Dimension | MCP | One CLI |
|---|---|---|
| Form | HTTP/stdio server per service | A single local binary |
| Auth | Managed by each server, ad hoc | Unified vault, multi-account, multi-provider |
| Per-project scope | No equivalent | Versioned `.onerc.yaml` file |
| Human setup | Not natively covered | First-class `install` verb |
| Distribution | Each server is a separate package | Reviewable centralized catalog |
| Audit | Per-server logs | Unified audit log `one trace` |
| Mental model | "The agent talks to servers" | "The agent uses a local tool" |

**One CLI is not anti-MCP.** Both can coexist. Long-term, One CLI could expose its catalog in "MCP server" mode for those who want to integrate it into that protocol. The differentiator remains **per-project governance via scope file** and **local vault**, two things MCP does not cover and is not intended to cover.

## Core concepts

Five concepts to master in order to understand everything else. They connect like this:

```
     ┌─────────────────────────────────────────┐
     │              Agent or human             │
     └─────────────────┬───────────────────────┘
                       │  one <service> <action>
                       ▼
     ┌─────────────────────────────────────────┐
     │              Binary `one`              │
     │                                         │
     │   ┌────────┐  ┌────────┐  ┌─────────┐   │
     │   │ Scope  │  │ Vault  │  │ Catalog │   │
     │   │ file   │  │        │  │         │   │
     │   └────────┘  └────────┘  └─────────┘   │
     │                                         │
     │     skill                install        │
     └─────────────────────────────────────────┘
```

### Catalog

A structured directory (distributed via a public Git repo + JSON index on CDN) that contains the definition of each supported service. For each service: its actions, its permissions, its auth providers, its install guides, its markdown skill.

The catalog is **purely declarative** (YAML + Markdown). When a service requires more than declarative (request signing, complex GraphQL, call chains), a **sandboxed WASM handler** is attached.

The catalog is **reviewable by PR**: adding a service means opening a pull request. See [CATALOG.md](./CATALOG.md).

### Vault

Secure local storage for credentials. Three implementations chained by priority:

1. **Environment variables** (`ONE_CREDS_*`) for CI and overrides
2. **Native OS keychain** (macOS Keychain, libsecret Linux, Credential Manager Windows) for desktop machines
3. **age-encrypted file** (`~/.one/vault.age`) for headless contexts without a keychain

The vault stores typed `Credential` objects (access token, refresh token, expiration, scopes, metadata). Never in plaintext in config files, never transmitted anywhere other than in the HTTP requests of handlers.

**Native multi-account.** A service can have multiple accounts (`github:work`, `github:personal`). The user chooses which one to use per project via the scope file or via `--as <alias>` at runtime.

See [AUTH.md](./AUTH.md) for details.

### Scope file

A `.onerc.yaml` file versioned at the root of the project that declares *what an agent can do on that specific project*. Minimal format:

```yaml
version: 1
services:
  github:
    allow: [issues.*, pulls.read]
    deny: [issues.delete]
  notion:
    allow: [pages.*, blocks.*]
```

**Strict default deny**: anything not explicitly allowed is denied.

**Layering**: a `.onerc.local.yaml` file (gitignored) can further restrict or change accounts, but never extend. This rule is non-negotiable: the committed scope file is the source of truth.

**Lock file** `.onerc.lock` pins the resolved catalog versions, like a `package-lock.json`.

See [SCOPE.md](./SCOPE.md) for the full grammar.

### Skill

A markdown document embedded in the binary (`one skill`) that tells an AI agent how to use One CLI. It describes the discovery flow, the four verbs, exit codes, idiomatic patterns, and anti-patterns.

The skill is **installable in the agent's IDE** (`one skill --install`) which detects Claude Code, Cursor, Aider, etc., and writes the skill to the correct location in the project.

Each service also has its own skill (`one info <service>`), focused on the mental model and gotchas of that service.

### Install guides

Markdown recipes that describe the human steps required to make a service work. Examples:

- Share a Notion page with the integration
- Create a Stripe webhook
- Configure an AWS IAM role

Each guide has an interactive mode (TTY) and a JSON mode (agent). It can declare a `verify` action that proves the install worked.

The central mechanism: when an action fails with a code mapped to a guide, the outgoing error directly includes `install.command: "one install <service> <guide>"`. The agent recognizes it needs to ask the human, rather than looping.

## The four agent verbs

The API surface offered to agents is intentionally minimal:

```
one <service> <action> [args]    # execute an action
one capabilities [<service>]     # JSON introspection (what exists)
one info [<service> [<action>]]  # markdown documentation (how to use it)
one can <service> <action>       # permission precheck (exit 0/3)
```

Plus a utility hint command:

```
one install <service> <guide>    # display a required human guide
```

Everything else (login, scope, accounts, lock, doctor, trace) is for the human who *configures* the binary, not for the agent who *uses* it.

## Philosophy

Six non-negotiable rules that guide trade-offs:

### 1. Declarative first, code as last resort

YAML describes 95% of services (standard REST with auth). WASM only arrives when declarative is insufficient (SigV4 signing, GraphQL with typed interpolation, call chains with rollback).

**Benefit**: a reviewer sees what a service will do by reading the YAML. No "read the code to understand".

### 2. Default deny everywhere

The scope file is deny by default. The vault refuses undeclared credentials. WASM handlers can only hit allowlisted URLs. No permission is implicit.

**Benefit**: a dev opening `.onerc.yaml` sees *exactly* what is allowed, without needing to know the system's defaults.

### 3. Audit is first-class, not an add-on

Every execution is traceable via `one trace`. Every HTTP request emitted by a handler is logged (method, URL, status, duration, without body). Credentials are redacted via the `Secret` type.

**Benefit**: when an agent does something unexpected, the trace can be recovered in minutes.

### 4. No network magic

No background daemon, no auto-discovery, no mDNS, no hidden polling. The binary does what the user or agent explicitly asks of it, nothing more.

**Benefit**: latency, battery consumption, and network security are predictable.

### 5. Documentation is part of the deliverable

A service without an up-to-date `SKILL.md` is not merged into the catalog. A command not documented in the `onecli` skill is not released. A feature without an example in the docs does not exist.

**Benefit**: agents that don't have the docs in context don't use the binary correctly. The docs are the interface, not a bonus.

### 6. The code is stubbornly simple

No DI framework. No custom code generation. No DSL. If choosing between an abstraction and 50 duplicated lines, take the 50 lines.

**Benefit**: a contributor can understand the code in a day. No wizard needed to evolve the project.

## Structural decisions and their rationale

Documented here so that future contributors (and Ely) know why these choices were made, and can revisit them with context.

### Go for the binary

**Why**: cold start <30ms, native cross-compilation single-binary, mature keychain ecosystem, trivial distribution (no runtime to install).

**Not Rust**: unjustified complexity overhead, compile times that kill velocity, no critical invariants where the borrow checker would save us.

**Not TypeScript/Bun**: too slow cold start for a binary called 50 times per session, runtime to install or bundle (50+ MB), less clean keychain bindings.

### Catalog in a separate Git repo + JSON index on CDN

**Why**: reviewable by PR like Homebrew, free to host (Pages or R2), no server infrastructure to maintain.

**Not a custom HTTP registry**: single point of failure, infrastructure cost, maintenance complexity.

**Not decentralized (one repo per service)**: broken discovery, variable quality, no centralized listing.

### WASM (wazero) for complex handlers

**Why**: sandbox by default (WASI = nothing exposed), polyglot (TS via Javy, Go via tinygo, Rust directly), distributable in the service tarball.

**Not native Go plugins**: no sandbox, catastrophic security, non-reproducible binary.

**Not arbitrary script execution (Lua, JS V8)**: same sandbox problem, or adds native dependencies that break cross-compilation.

### Scope file versioned in the repo

**Why**: code review of permissions, reproducibility across devs, auditable source of truth.

**Not in a cloud config**: would create a dependency, break offline-first, create a single point of failure.

**Not in the vault**: the scope is public (committable), not secret. Mixing the two is a design error.

### Install guides in markdown with frontmatter

**Why**: human-readable, machine-parsable, versionable in the catalog, translatable.

**Not pure YAML**: unreadable for a human following the steps.

**Not an executable script**: brings the sandbox back into guides, attack vector.

### Typed exit codes (0/1/2/3/4/5)

**Why**: the agent can write clean conditional logic (`if exit==3 then ask scope`) without parsing natural language error messages.

**Not just 0/1**: too limited, forces stderr parsing.

**Not full 7-bit ASCII** (e.g. 0-127): Unix convention says >128 are signals, we stay within the standard.

## Non-goals

What One CLI **does not do** and is not intended to do. Important to clarify to avoid scope creep.

**One CLI is not a workflow orchestrator.** No DAGs, no automatic retries between actions, no scheduling. Want that? Use Temporal, Trigger.dev, or a script.

**One CLI is not a public API proxy.** It is not a service to deploy, it is a local binary. If you want a proxy with rate limiting, route to `kong` or `nginx`.

**One CLI is not an LLM gateway.** Anthropic, OpenAI, etc. can be *called* via the catalog, but it is not a unified entry point for completions (like portkey, OpenRouter). If you want that, use those dedicated tools.

**One CLI does not manage code generated by agents.** No code execution in a sandbox. It is a tool for *calling services*, not for *running code*.

**One CLI does not do usage-based pricing.** Pure open source. Credentials are yours, API calls are billed by third-party services as usual.

**One CLI is not an agent.** It is a *tool* usable by agents. It does not decide, it executes.

## Target users

Three audiences, in this order of priority:

**1. The dev building AI agents in 2026.** They use Claude Code, Cursor, or a custom agent. They want to ship integrations quickly without managing 10 SDKs. They prefer a local binary under their control to a third-party service.

**2. The AI agent itself.** It consumes the `onecli` skill, does `capabilities`/`info`/`exec`, handles exit codes. It is the most *frequent* user of the binary but not the one who configures it.

**3. The team lead who wants governance.** They define the scope file for their team, review PRs on `.onerc.yaml`, and ensure no agent accidentally `delete repos` in prod.

**Not targeted initially**: non-technical general public users, enterprises with strict SOC2 compliance (coming in v1.0+).

## Design success metrics

How to know if the design delivers on its promises? Four signals to monitor:

1. **Time to add a simple service to the catalog** (e.g. Resend): <2h for a dev discovering it for the first time. If it takes 8h, the service.yaml format is too complex.

2. **Time for an agent to understand One CLI**: <30s of reading the skill before being able to use it. If the agent consistently makes mistakes after reading the skill, the docs are poorly written.

3. **Number of "how do I do X" questions on GitHub Discussions**: should decrease over time as the docs grow. If it stays stable, the project is less well documented than it should be.

4. **Surface of breaking changes per version**: 0 between minor versions, explicitly listed between major versions. If there are 3 per minor release, the API was not mature.

## Planned evolution

This design is frozen for v0.1 to v1.0. After v1.0, possible evolutions:

- **MCP server mode**: expose the One CLI catalog as an MCP server for interop.
- **Remote audit log**: opt-in sending of traces to an endpoint for teams wanting centralized monitoring.
- **Scope templates**: `.onerc.yaml.template` shared by the community (e.g. "scope for a customer support agent", "scope for a DevOps agent").
- **Enterprise tier**: paid support for organizations, without changing the open source core.

None of these evolutions breaks the concepts described in this document.

---

*Document maintained in parallel with the code. If a decision described here becomes obsolete, either update this document with a clear changelog, or revert the code.*
