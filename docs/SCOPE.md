# SCOPE.md

> Complete reference for the scope file (`.onerc.yaml`): grammar, layering, lock file, commands. For available permissions per service, see the catalog or `one info <service>`.

## Overview

The scope file is a versioned YAML file at the root of the project that declares **what an agent (or a developer) is allowed to do on this project via One CLI**. It is the central governance unit.

Three non-negotiable properties:

1. **Strict default deny.** Everything not explicitly allowed is denied.
2. **Readable without documentation.** A developer opening the file must understand what is allowed within 30 seconds.
3. **Versioned in the repo.** Reviewable in PRs, shared by the team, reproducible across machines.

## The three files

| File | Role | Git status |
|---|---|---|
| `.onerc.yaml` | Project source of truth | committed |
| `.onerc.local.yaml` | Personal overrides | gitignored |
| `.onerc.lock` | Frozen catalog resolution | committed |

## Full grammar

### Minimal skeleton

```yaml
version: 1

services:
  github:
    allow:
      - issues.*
      - pulls.read
  notion:
    allow:
      - pages.*
```

This is sufficient for the majority of projects.

### Full grammar

```yaml
version: 1

project:
  name: kaampus-backend
  description: Backend API for Kaampus marketplace

defaults:
  github: work
  notion: kaampus
  stripe: test

profile: default                    # active profile (if profiles are defined)

profiles:
  default:
    defaults:
      stripe: test
    services:
      stripe:
        allow: [customers.*, subscriptions.*]

  production:
    extends: default
    defaults:
      stripe: live
    services:
      stripe:
        deny: [customers.delete, subscriptions.cancel]

services:
  github:
    account: work                   # overrides defaults.github
    allow:
      - issues.*
      - pulls.read
      - pulls.review
      - repos.read
    deny:
      - issues.delete
      - repos.delete

  notion:
    account: kaampus
    allow:
      - pages.*
      - blocks.*
      - databases.read

  stripe:
    allow:
      - customers.read
      - subscriptions.read
```

### Detailed fields

#### `version` (required)

Format version. Always `1` currently. Allows the format to evolve in the future without breaking existing projects.

#### `project` (optional)

Human metadata. Displayed in `one info` when inside this project. Useful for distinguishing multiple projects on the same machine.

```yaml
project:
  name: kaampus-backend
  description: Backend API for the Kaampus marketplace
```

#### `defaults` (optional)

Default account per service. If not specified, the alias `default` is used.

```yaml
defaults:
  github: work
  notion: kaampus
  stripe: test
```

Overridable per service (`services.<name>.account`) or ad-hoc (`one --as perso github ...`).

#### `services` (required if you want to allow anything)

Permissions per service.

```yaml
services:
  github:
    account: work              # optional, otherwise inherits from defaults.github
    allow: [...]               # list of allowed patterns
    deny: [...]                # list of denied patterns (overrides allow)
```

#### `profiles` (optional, v1.1+)

Named profiles for managing multiple environments (dev, staging, prod).

```yaml
profiles:
  default:
    defaults:
      stripe: test
    services:
      stripe:
        allow: [customers.*]

  production:
    extends: default
    defaults:
      stripe: live
    services:
      stripe:
        deny: [customers.delete]
```

Selection:

- By default: `default` profile
- Via env var: `ONE_PROFILE=production one ...`
- Via local: `profile: production` in `.onerc.local.yaml`

Inheritance (`extends`) recursively merges parent fields, with the child overriding conflicts.

## Globs

A single, minimalist rule:

| Pattern | Match |
|---|---|
| `pages.read` | exactly this permission |
| `pages.*` | everything starting with `pages.` (single level) |
| `*` | all permissions for the service |

**Not allowed**:

- `**` (rejected at validation, too ambiguous)
- `pages.{read,write}` (no brace expansion)
- `!pages.delete` (no anti-pattern, use `deny`)
- `pages.?` (no wildcard char)

To exclude a specific permission, use `deny`. The grammar remains readable and predictable.

### Evaluation precedence

The order is fixed and documented:

```
1. deny exact         (deny: [pages.delete] vs perm pages.delete)
2. deny glob          (deny: [pages.*] vs perm pages.delete)
3. allow exact        (allow: [pages.read] vs perm pages.read)
4. allow glob         (allow: [pages.*] vs perm pages.read)
5. default deny       (nothing matches)
```

**Examples**:

```yaml
allow: [pages.*]
deny: [pages.delete]
# → all pages.X allowed except pages.delete
```

```yaml
allow: [*]
deny: [databases.write]
# → everything allowed except databases.write
```

```yaml
allow: [pages.delete]
deny: [pages.*]
# → pages.delete allowed (does deny glob beat allow exact? NO, deny glob = 2, allow exact = 3)
# wait, let's redo: deny glob (pages.*) matches pages.delete → DENY (step 2 terminates)
# → pages.delete DENIED
```

The example above shows that **understanding precedence is required**. The `one scope explain` command (see below) traces the evaluation to help debug.

## Layering: `.onerc.local.yaml`

The `.onerc.local.yaml` is gitignored and allows each developer to adjust settings for their machine. **Strict rule**: it can only **remove or change accounts**, never add a permission that is not in the base.

### What is allowed in `.local`

```yaml
version: 1

# Change profile
profile: development

# Change default account
defaults:
  github: perso

# Restrict further
services:
  github:
    deny:
      - issues.delete
      - pulls.merge
```

### What is forbidden in `.local`

```yaml
# FORBIDDEN: adding a permission absent from the base
services:
  github:
    allow:
      - repos.delete           # not in .onerc.yaml → ignored + warning
  newservice:                  # service absent from base → ignored + warning
    allow: [*]
```

### Merge algorithm

```
result.profile = local.profile ?? base.profile

result.defaults = merge(base.defaults, local.defaults)
  # local overrides base per service

For each service present in base.services:
  result.allow = base.allow                              # local CANNOT extend
  result.deny = base.deny ∪ local.deny                   # local CAN restrict
  result.account = local.account ?? base.account         # local CAN change

For each service present only in local.services:
  → warning "service X in .onerc.local.yaml not in .onerc.yaml, ignored"
```

**Rationale for the no-extension rule**: if local could extend, the committed scope file would no longer be the source of truth. Someone opening the repo would not see what is *actually* allowed on the developer's machine. Bad for audit and reproducibility.

## The lock file `.onerc.lock`

Freezes the resolved catalog versions. Like a `package-lock.json`.

### Format

```yaml
version: 1
generated_at: 2026-05-20T14:32:11Z

catalog:
  source: https://catalog.one-cli.dev
  index_sha256: 7a8b9c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b

services:
  github:
    version: 2.1.4
    tarball_sha256: 3f2a1b...
  notion:
    version: 1.4.0
    tarball_sha256: e9d8c7...
  stripe:
    version: 3.0.2
    tarball_sha256: 5c4b3a...
```

### Strict behavior

If a locally installed version does not match the lock, the binary **refuses** to execute the action:

```
$ one notion pages.read --page_id ...
Error: notion version mismatch
  Lock:       1.4.0 (sha256:e9d8c7...)
  Installed:  1.5.2 (sha256:abc123...)

Run `one catalog update` to fetch the locked version, or
`one lock --update notion` to bump the lock.

Exit 1
```

Guarantees reproducibility across machines: all developers and CI use the same catalog versions.

### Commands

```bash
one lock                              # generates the lock from current versions
one lock --update notion              # updates a specific service
one lock --update-all                 # updates all services
one lock --check                      # verifies that local matches (exit 0/1)
```

## Commands to modify the scope

Agents and humans should **never** edit the file directly, except during a PR review. The CLI provides these commands:

```bash
one scope show                        # displays the effective scope (including merge) as JSON
one scope add <service> <perm>        # adds a permission to allow
one scope add <service> <perm> --deny # adds to deny
one scope remove <service> <perm>     # removes a permission
one scope remove <service> <perm> --deny  # removes from deny
one scope use <service> --as <alias>  # changes the default account
one scope check                       # validates consistency (schema + catalog)
one scope explain <service> <perm>    # traces evaluation, shows why a permission is allowed/denied
```

### `one scope show`

Displays the effective scope after merging base + local + profile:

```json
{
  "version": 1,
  "project": "kaampus-backend",
  "profile": "default",
  "services": {
    "github": {
      "account": "perso",
      "allow": ["issues.*", "pulls.read"],
      "deny": ["issues.delete"]
    },
    "notion": {
      "account": "kaampus",
      "allow": ["pages.*", "blocks.*"]
    }
  }
}
```

Useful for agents that want to introspect before acting.

### `one scope check`

Exhaustive validation of the file:

```
$ one scope check
✓ Schema valid (.onerc.yaml v1)
✓ Schema valid (.onerc.local.yaml v1)
✓ Merge has no conflicts
✓ All services exist in catalog
✓ All permissions exist in their services
✓ Lock file matches installed catalog
⚠ Service 'discord' is in scope but not authenticated
    → Run `one login discord` to authenticate
✗ Permission 'github.actions.read' does not exist
    → Did you mean: 'actions.list'?

Errors: 1   Warnings: 1
```

Exit codes:

- 0: no errors or warnings
- 1: warnings only
- 2: errors present

Ideal as a pre-commit hook or in CI.

### `one scope explain`

Traces the evaluation, line by line:

```
$ one scope explain github issues.delete
Permission: github.issues.delete
Result: DENIED

Evaluated in order:
  1. .onerc.local.yaml:deny - matched 'issues.delete' (exact) → DENY (final)

Rules not evaluated (would have applied):
  - .onerc.yaml:allow - 'issues.*' would have matched
  - .onerc.yaml:deny  - 'issues.delete' would have matched (redundant)

To allow:
  Remove the deny rule from .onerc.local.yaml:
    one scope remove github issues.delete --deny --local
```

This is what makes the grammar usable beyond trivial cases. When an agent says "I don't have permission", the developer runs `one scope explain` and sees exactly why in 5 seconds.

## Example: typical workflow on a new project

### First installation

```bash
cd my-project
one init                              # creates an empty .onerc.yaml
```

`.onerc.yaml` is created minimal:

```yaml
version: 1
services: {}
```

### Login to required services

```bash
one login github                      # OAuth, alias "default"
one login notion                      # OAuth, alias prompted
```

### Define the scope

```bash
one scope add github issues.*
one scope add github pulls.read
one scope add notion pages.*
one scope add notion blocks.*
```

`.onerc.yaml` becomes:

```yaml
version: 1
services:
  github:
    allow:
      - issues.*
      - pulls.read
  notion:
    allow:
      - pages.*
      - blocks.*
```

### Lock

```bash
one lock
git add .onerc.yaml .onerc.lock
git commit -m "init: add One CLI scope"
```

### A colleague clones

```bash
git clone ...
cd my-project
one catalog update                    # fetch the locked versions
one login github                      # creates their own account
one login notion
# The scope is already defined, no need to redefine it
```

## JSON Schema (formal excerpt)

Reference for validation. The full schema is in `pkg/catalog/schema/onerc-v1.json`.

```yaml
$schema: https://json-schema.org/draft/2020-12/schema
type: object
required: [version]
additionalProperties: false
properties:
  version:
    const: 1
  project:
    type: object
    additionalProperties: false
    properties:
      name: { type: string }
      description: { type: string }
  profile:
    type: string
  profiles:
    type: object
    additionalProperties:
      $ref: "#/$defs/profile"
  defaults:
    type: object
    additionalProperties:
      type: string
  services:
    type: object
    additionalProperties:
      $ref: "#/$defs/serviceScope"

$defs:
  profile:
    type: object
    properties:
      extends:
        type: string
      defaults: { $ref: "#/properties/defaults" }
      services: { $ref: "#/properties/services" }

  serviceScope:
    type: object
    additionalProperties: false
    properties:
      account:
        type: string
        pattern: "^[a-z][a-z0-9_-]*$"
      allow:
        type: array
        items:
          $ref: "#/$defs/permPattern"
      deny:
        type: array
        items:
          $ref: "#/$defs/permPattern"

  permPattern:
    type: string
    pattern: "^[a-z][a-z0-9_]*(\\.[a-z0-9_*]+)*$"   # disallows `**` and `?`
```

## Common recipes

### Read-only agent

```yaml
version: 1
services:
  github:
    allow:
      - "*.read"
      - "*.list"
      - "*.search"
      - search
```

### Restrict destructive operations on all services

```yaml
version: 1
services:
  github:
    allow: ["*"]
    deny:
      - repos.delete
      - issues.delete
  notion:
    allow: ["*"]
    deny:
      - pages.archive
      - databases.delete
```

### Test account in dev, prod in CI

```yaml
# .onerc.yaml
version: 1
profile: default
profiles:
  default:
    defaults:
      stripe: test
  production:
    extends: default
    defaults:
      stripe: live
services:
  stripe:
    allow: [customers.*, charges.*]
```

```yaml
# .onerc.local.yaml (for each developer)
# Nothing, the "test" default is fine
```

```yaml
# On prod CI: env var ONE_PROFILE=production
```

### Strictly limiting an AI agent

Use case: a project where an AI agent runs autonomously on production code. We want to be very strict.

```yaml
version: 1
services:
  github:
    allow:
      - issues.read
      - issues.create
      - issues.comment
      - pulls.read
      - pulls.review            # allowed to review but not merge
    deny:
      - "*"                     # explicit, at the bottom
  # no notion, no stripe, no other service
```

The `deny: ["*"]` at the bottom is redundant due to default deny, but makes the intent explicit.

## Anti-patterns to reject in code review

### Allow `["*"]` without deny

```yaml
# DANGEROUS
services:
  github:
    allow: ["*"]
```

Everything is allowed. This is rarely what you want. Either list permissions explicitly, or use `allow: ["*"]` + `deny:` for destructive actions.

### Permissions without namespace

```yaml
# BAD
services:
  github:
    allow:
      - "read"                  # read what?
      - "*"                     # too broad
      - "issue.delete"          # singular instead of plural
```

Respect the `<resource>.<verb>` convention exactly as declared by the service.

### Unresolved conflicts

```yaml
# CONFUSING
services:
  github:
    allow:
      - issues.*
      - issues.delete
    deny:
      - issues.delete
```

`issues.*` already covers `issues.delete`, adding `issues.delete` to allow is redundant. With the deny that follows, it is ambiguous for the reader. `one scope check` warns in this case.

### Mixing profiles and top-level

```yaml
# CONFUSING
services:                       # defined at top-level
  github:
    allow: [issues.read]
profiles:
  default:
    services:                   # AND in the default profile
      github:
        allow: [issues.write]
```

Choose one or the other. Either no profiles, or everything inside profiles.

---

*To propose a grammar change for the scope file, open an RFC in `one-cli/rfcs`. Major change = `version` bump. The binary must support previous versions for at least 12 months.*
