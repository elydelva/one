# Jira

Atlassian Jira Cloud via `one jira <action>`. Covers Platform REST v3 + Agile v1.0 (Boards/Sprints).

## Critical setup (read before use)

- **Tenant URL**: the `base_url` ships as the placeholder `https://your-site.atlassian.net`. Override it for your instance via `one scope set jira.base_url https://acme.atlassian.net` (or edit the service config) before any call. There is no global Jira host.
- **Auth**: provider is `pat`, but Jira wants Basic auth with `email:api_token`. Pre-encode locally: `printf '%s' "you@x.com:ATATT3xFfGF0..." | base64`. Store that single base64 string as the PAT. The header becomes `Authorization: Basic <base64>`.
- **JQL**: search uses POST `/rest/api/3/search/jql` with `{"jql": "project = ENG AND status = 'In Progress'", "fields": ["summary","status"]}`.
- **ADF**: every rich-text field (`description`, comment `body`, worklog `comment`) is Atlassian Document Format, a JSON tree. Pass it as a raw object, e.g. `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"hi"}]}]}`.

## Issues
- `issues.search`, `issues.read`, `issues.create`, `issues.update`, `issues.delete`
- `issues.assign`, `issues.transitions.list`, `issues.transition`
- `issues.watchers.list`, `issues.watchers.add`, `issues.watchers.remove`

## Comments
- `comments.list`, `comments.create`, `comments.update`, `comments.delete`

## Worklogs
- `worklogs.list`, `worklogs.create`, `worklogs.update`, `worklogs.delete`

## Issue links + types + fields
- `issue_links.create`, `issue_link_types.list`
- `issue_types.list`, `fields.list`, `statuses.list`, `priorities.list`, `resolutions.list`

## Projects
- `projects.list`, `projects.read`, `projects.create`
- `project_versions.list`, `project_versions.create`, `project_versions.update`, `project_versions.delete`
- `project_components.list`, `project_components.create`

## Users
- `users.search`, `users.read`, `users.assignable.search`

## Agile (Boards + Sprints)
- `boards.list`, `boards.read`, `boards.backlog`, `boards.epics.list`, `boards.issues.list`
- `sprints.list`, `sprints.read`, `sprints.create`, `sprints.update`, `sprints.issues.list`

## Dashboards + filters
- `dashboards.list`, `filters.list`, `filters.read`

## Pagination

Most endpoints use `?startAt=N&maxResults=M`. New-style responses look like `{startAt,maxResults,total,isLast,values:[...]}`. The JQL search endpoint returns `{issues:[...], nextPageToken}` and is token-paginated: pass `nextPageToken` back in the body.
