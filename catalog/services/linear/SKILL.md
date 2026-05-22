# Linear

Linear API access via `one linear <action>`. Auth: PAT (Linear Personal API Key).

All actions wrap GraphQL operations as single `POST /graphql` calls. Object inputs (like `input`, `filter`) take the raw GraphQL input shape.

## Lookups (resolve names → IDs first)
- `users.me`, `users.list`, `teams.list`
- `workflow_states.list` (filter by team), `labels.list`

## Issues
- `issues.read` (by UUID or human key like `ENG-123`)
- `issues.list` (with IssueFilter), `issues.search` (full-text)
- `issues.create`, `issues.update`, `issues.archive`
- `issues.comment.create`, `comments.list`

## Projects + cycles
- `projects.create`, `projects.update`, `projects.read`, `projects.list`
- `cycles.list`, `cycles.create`, `cycles.update`
- `project_milestones.list`

## Labels + attachments
- `labels.create`, `labels.update`
- `attachments.create` (link GitHub PR / URL to issue), `attachments.list`

## Docs + initiatives + relations + webhooks
- `documents.create`, `documents.update`, `documents.list`
- `initiatives.create`, `initiatives.list`
- `issue_relations.create`, `issue_relations.remove`
- `webhooks.create`, `webhooks.list`

## Patterns
- Always resolve `teamId`, `assigneeId`, `stateId`, `labelIds[]` via lookup actions before mutations.
- GraphQL errors arrive as HTTP 200 with `errors[]` in body — inspect output JSON, not just status.
