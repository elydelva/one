---
title: Notion — Initial Setup
---

Three steps to use the Notion service via One CLI, plus a section on the **four traps** that bite every first-time Notion API user.

## 1. Create an Internal Integration

1. Go to https://www.notion.so/profile/integrations
2. Click **New integration**, give it a name (e.g. `one-cli-local`)
3. Pick the workspace. Capabilities to enable (drop those you don't need):
   - **Read content**, **Update content**, **Insert content** — pages/blocks/DBs
   - **Read comments**, **Insert comments** — comments.*
   - **Read user information** (+ **with email** if you need addresses) — users.*
4. Copy the **Internal Integration Token** (starts with `secret_` or `ntn_`)

## 2. Share target pages / databases with the integration

Notion integrations have **zero access by default**. For each page or database you want to reach:

- Open the page in Notion
- Click **⋯** (top right) → **Connections** → pick your integration

A DB you share also gives access to all its child pages. Skipping this step yields `404 not_found` or `403 forbidden` on every call.

## 3. Store the token in the One vault

```bash
one login notion         # --provider pat is the default; paste the token when prompted
one notion users.me      # smoke test — works without any shared page
```

## Notion API versions used in this catalog

`Notion-Version` is pinned **per action** (the runtime supports mixed versions). Current map:

| Action family | Version | Why |
|---|---|---|
| Pages markdown (`pages.read/append/prepend/replace/update/find_replace`) | `2026-03-11` | Enhanced Markdown surface |
| Pages JSON (`pages.create`, `pages.meta`) | `2022-06-28` | Stable baseline, no gain to bump |
| `pages.update_properties`, `pages.archive`, `pages.restore` | `2025-09-03` | `in_trash` field, file_upload property values |
| Databases / data sources | `2025-09-03` | Multi-source databases — schema lives on the data source |
| Comments | `2025-09-03` | `display_name`, `attachments` |
| Blocks (retrieve, update, delete, children.{list,append}) | `2025-09-03` | `in_trash` field |
| Users | `2025-09-03` | Stable everywhere |
| File uploads | `2025-09-03` | API introduced in 2025-09-03 |
| Search | `2022-06-28` | Stable |

## The four traps

### 1. Sharing is not transitive in the way you think

Sharing a **page** with an integration gives access to that page and its descendants. Sharing a **DB** gives access to the DB AND all rows. But pages mentioned via `relation` are NOT auto-shared — querying related pages needs them shared too.

When you see `404`, the integration is almost certainly missing access. Notion deliberately returns 404 (not 403) to avoid leaking page existence.

### 2. In API 2025-09-03, the schema lives on the data source, not the database

A database is a **container** with metadata (title, icon, cover). It wraps one or more **data sources**, and each data source carries the property schema and the rows.

- Legacy DBs (pre-multi-source) have exactly one data source whose id equals the old `database_id`.
- `databases.retrieve` returns `data_sources: [{id, name}]` — use that id for `data_sources.query`, `data_sources.update`, and `data_sources.retrieve`.
- Use `databases.update` for title/icon/cover, **not** for schema edits.

```bash
# Find data source id
one notion databases.retrieve --database_id <db>
# Then query
one notion data_sources.query --data_source_id <ds> --filter '{"property":"Status","status":{"equals":"Done"}}'
```

### 3. Archived is now `in_trash`

In 2025-09-03, the field `archived` was renamed `in_trash` on pages and blocks. Querying a data source does NOT return trashed pages by default. To archive: `pages.archive` (or set `in_trash:true` via `pages.update_properties`). Recoverable via `pages.restore`.

### 4. Property names are case-sensitive and must match the schema exactly

When you write `pages.update_properties`:

```json
{"properties": {"Status": {"status": {"name": "Done"}}}}
```

- `"Status"` must match the column name letter-for-letter.
- The inner key (`"status"`) must match the column **type**. Passing `select` to a `status` column → `400`.
- `"name": "Done"` must be an option already in the column's option list. Add new options via `data_sources.update` first.

Same for `data_sources.query` filters — the `property` field is the column name; the inner key is the column type.

## Worked examples

### Query a DB

```bash
# 1. Resolve the data source
DS=$(one notion databases.retrieve --database_id <db> --json | jq -r '.data_sources[0].id')

# 2. Query
one notion data_sources.query \
  --data_source_id "$DS" \
  --filter '{"and":[
    {"property":"Status","status":{"equals":"In Progress"}},
    {"property":"Date","date":{"on_or_after":"2026-01-01"}}
  ]}' \
  --sorts '[{"property":"Date","direction":"descending"}]'
```

### Create a row

```bash
one notion pages.create \
  --parent '{"database_id":"<db>"}' \
  --properties '{
    "Name":   {"title":     [{"type":"text","text":{"content":"Ship v1"}}]},
    "Status": {"status":    {"name":"In Progress"}},
    "Tags":   {"multi_select":[{"name":"urgent"}]},
    "Date":   {"date":      {"start":"2026-05-21"}}
  }'
```

### Post a comment

```bash
one notion comments.create \
  --parent '{"page_id":"<page>"}' \
  --rich_text '[{"type":"text","text":{"content":"LGTM"}}]'
```

### Append blocks to a page

```bash
one notion blocks.children.append \
  --block_id <page> \
  --children '[
    {"object":"block","type":"heading_2",
     "heading_2":{"rich_text":[{"type":"text","text":{"content":"Notes"}}]}},
    {"object":"block","type":"paragraph",
     "paragraph":{"rich_text":[{"type":"text","text":{"content":"Hello"}}]}}
  ]'
```

## Rate limit

Notion enforces ~3 requests/second average per integration with bursting. On `429`, honor the `Retry-After` header. The runtime applies exponential backoff for actions marked `retry: backoff`.

## Pagination

Cursor-based across the API (`start_cursor` / `next_cursor`), capped at `max_pages: 10` in this fixture — raise per-action if you genuinely need more.

**Output shape:** for any action with a `pagination` block, the One runtime auto-walks pages up to `max_pages` and **returns the flattened items array directly** under `.data`. The raw Notion envelope (`{results, has_more, next_cursor, ...}`) is NOT exposed.

```bash
# Notion raw response:  {"object":"list", "results":[...], "has_more":..., "next_cursor":...}
# One response:         {"data":[ ...items... ], "trace_id":"..."}

one exec notion comments.list --block_id <page> | jq '.data | length'      # ✓
one exec notion comments.list --block_id <page> | jq '.data[0]'            # ✓
one exec notion comments.list --block_id <page> | jq '.data.results[0]'    # ✗ won't work
```

Actions affected (every action with a `pagination:` block in this fixture):

- `search`
- `blocks.children.list`
- `data_sources.query`
- `comments.list`
- `users.list`
- `file_uploads.list`

For non-paginated actions (`*.retrieve`, `*.create`, `*.update`, etc.) the Notion payload is passed through as-is under `.data`.
