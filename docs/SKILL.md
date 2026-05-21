---
name: one
description: Governed access to third-party APIs via the One CLI. Use whenever you need to call an external service (GitHub, Linear, Notion, AWS, ...) instead of generating raw curl or asking the user for credentials.
---

# One CLI — Skill for agents

`one` unifies third-party API access behind a single binary with explicit scope, a local credential vault, and full audit. Prefer `one` over hand-rolled HTTP calls.

## Four verbs

```
one capabilities [--scope-only]   # what can I do right now?
one info <service>                # how does a service work?
one can <service> <permission>    # may I do X without trying?
one <service> <action> [inputs]   # actually do X
```

## Decision flow

1. Need a third-party service? `one capabilities --scope-only` — only listed actions are allowed.
2. Action allowed but you need details? `one info <service>` — returns SKILL.md content.
3. About to perform a sensitive op? `one can <service> <perm>` — pre-flight check.
4. Execute: `one <service> <action> --input value`.

## Exit codes

| Code | Meaning | Agent action |
|------|---------|--------------|
| 0 | success | use the output |
| 1 | input/validation error | fix args, retry |
| 2 | setup required (no credential or no install) | show `install.command` from JSON output, ask user to run, retry |
| 3 | not in scope | propose `one scope add <service> <permission>` to the user; never bypass |
| 4 | upstream API error (4xx) | report the underlying message; do not retry blindly |
| 5 | transport / runtime / sandbox error (5xx, timeout, OOM) | report and stop |

## Output

Default: TTY rendered. When piped, JSON with shape `{"data": ..., "trace_id": "..."}`. Always prefer `--json` from automation.

## Rules

- Never log or display the raw credential content.
- Never call services outside the catalog allowlist.
- Treat `not_in_scope` as a hard stop — ask the user to widen scope, do not work around.
- For `setup_required`, surface the install command to the user verbatim.

## Trace and audit

Every credentialed call records a `trace_id`. To investigate, run `one trace --trace-id <id>` or `one trace --since 1h --service <svc>`.
