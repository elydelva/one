# Cloudflare

Use the Cloudflare API v4 via `one cloudflare <action>`. Auth: API Token (Bearer).

Most endpoints are scoped by `{account_id}` (resources) or `{zone_id}` (per-domain). Responses wrap data in `{success, result, result_info, errors}`.

## Accounts + zones
- `accounts.list`, `zones.list`, `zones.read`

## DNS
- `dns_records.list`, `dns_records.read`, `dns_records.create`, `dns_records.update`, `dns_records.delete`

## Cache
- `cache.purge` (by URL, tag, host, prefix, or everything)

## Workers
- `workers.scripts.list`, `workers.scripts.read`, `workers.scripts.upload`, `workers.scripts.delete`
- `workers.routes.list`, `workers.routes.create`, `workers.routes.update`, `workers.routes.delete`
- `workers.subdomain.read`
- `workers_ai.run` (inference against a model slug)

## KV
- `kv.namespaces.list`, `kv.namespaces.create`, `kv.namespaces.delete`
- `kv.values.read`, `kv.values.write`, `kv.values.delete`, `kv.keys.list`

## R2
- `r2.buckets.list`, `r2.buckets.read`, `r2.buckets.create`, `r2.buckets.delete`
- Object I/O uses the separate S3-compatible endpoint, not v4.

## D1
- `d1.databases.list`, `d1.databases.read`, `d1.databases.create`, `d1.databases.delete`, `d1.databases.query`

## Pages
- `pages.projects.list`, `pages.projects.read`

## Page rules + SSL
- `page_rules.list`, `page_rules.create`, `page_rules.update`, `page_rules.delete`
- `ssl_certificates.list`

## Tunnels
- `tunnels.list`, `tunnels.read`, `tunnels.create`, `tunnels.delete`

## Hyperdrive
- `hyperdrive.configs.list`, `hyperdrive.configs.create`, `hyperdrive.configs.update`, `hyperdrive.configs.delete`

## Email Routing
- `email_routing.rules.list`, `email_routing.rules.create`, `email_routing.rules.delete`

## Stream
- `stream.videos.list`, `stream.videos.read`, `stream.videos.copy`, `stream.videos.delete`

## Access + analytics
- `access.apps.list`
- `analytics.dashboard.read`
