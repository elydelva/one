# Sentry

Use the Sentry API via `one sentry <action>`. Auth: User or Organization Auth Token (PAT). Default base is `https://sentry.io/api/0`; for self-hosted, override `base_url`. Most paths use `{organization_slug}` and `{project_slug}`. Pagination is cursor-based via the `Link` header; pass `cursor=<value>`.

## Organizations + teams
- `organizations.list`, `organizations.read`
- `teams.list`, `teams.create`, `teams.read`
- `members.list`

## Projects
- `projects.list`, `projects.read`, `projects.create`, `projects.update`, `projects.delete`
- `projects.keys.list`

## Issues + events
- `issues.list` (supports `query=` like `is:unresolved`), `issues.read`
- `issues.update` (resolve, assign, ignore, bookmark)
- `issues.delete`
- `issues.events.list`, `events.read`
- `events.list` (per project)

## Releases + deploys
- `releases.list`, `releases.read`, `releases.create`, `releases.update`, `releases.delete`
- `releases.deploys.list`, `releases.deploys.create`

## Alerts + monitors
- `alerts.rules.list`, `alerts.rules.create`, `alerts.rules.update`, `alerts.rules.delete`
- `monitors.list`, `monitors.create`, `monitors.update`, `monitors.delete`

## Discover + sessions + stats
- `discover.query`
- `sessions.read`
- `stats.read`

## Dashboards + integrations
- `dashboards.list`, `dashboards.create`, `dashboards.update`, `dashboards.delete`
- `integrations.list`
