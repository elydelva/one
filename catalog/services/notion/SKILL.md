# Notion

Notion API via `one notion <action>`. Auth: PAT (Internal Integration Token). Share each target page/DB with the integration first.

## Pages
- `pages.create`, `pages.read`, `pages.meta`, `pages.update`, `pages.update_properties`
- `pages.append`, `pages.prepend`, `pages.replace`, `pages.find_replace`
- `pages.move`, `pages.archive`, `pages.restore`

## Blocks
- `blocks.retrieve`, `blocks.update`, `blocks.delete`
- `blocks.children.list`, `blocks.children.append`

## Databases + data sources (API 2025-09-03+)
- `databases.create`, `databases.retrieve`, `databases.update`
- `data_sources.create`, `data_sources.retrieve`, `data_sources.update`, `data_sources.query`, `data_sources.templates`

## Views (API 2026-04-01)
- `views.create`, `views.list`, `views.retrieve`, `views.update`, `views.delete`
- `views.queries.create`, `views.queries.retrieve`, `views.queries.delete`

## Search + users + comments
- `search`
- `users.me`, `users.list`, `users.retrieve`
- `comments.create`, `comments.list`, `comments.retrieve`

## File uploads
- `file_uploads.create`, `file_uploads.send`, `file_uploads.complete`
- `file_uploads.list`, `file_uploads.retrieve`

## Other
- `custom_emojis.list`

## Patterns
- `find_replace` requires exact substring match; use `replace_all=true` when ambiguous.
- Page write actions need integration to have "Update content" capability.
- Honor `Retry-After` on 429 (~3 req/s sustained).
