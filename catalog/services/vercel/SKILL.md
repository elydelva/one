# Vercel

Use the Vercel REST API via `one vercel <action>`. Auth: PAT (Bearer token). Most actions accept an optional `teamId` query input for team-scoped operations.

## Deployments
- `deployments.list`, `deployments.read`, `deployments.create`, `deployments.cancel`, `deployments.delete`
- `deployments.events`, `deployments.files.list`, `deployments.file.read`

## Projects
- `projects.list`, `projects.read`, `projects.create`, `projects.update`, `projects.delete`

## Environment variables (project-scoped)
- `env.list`, `env.create`, `env.update`, `env.delete`

## Shared environment variables (team-scoped)
- `shared_env.list`, `shared_env.create`, `shared_env.update`, `shared_env.delete`

## Domains
- `domains.list`, `domains.read`, `domains.add`, `domains.remove`, `domains.config`

## DNS
- `dns.list`, `dns.create`, `dns.update`, `dns.delete`

## Aliases
- `aliases.list`, `aliases.read`, `aliases.assign`, `aliases.delete`

## Logs + checks
- `logs.runtime`
- `checks.list`, `checks.create`, `checks.read`, `checks.update`, `checks.rerequest`

## Edge config
- `edge_config.list`, `edge_config.create`, `edge_config.read`, `edge_config.delete`
- `edge_config.items.list`, `edge_config.items.update`

## Teams
- `teams.list`, `teams.read`, `teams.members.list`

## Webhooks
- `webhooks.list`, `webhooks.create`, `webhooks.read`, `webhooks.delete`

## Certs
- `certs.read`, `certs.issue`, `certs.delete`

## User
- `user.read`
