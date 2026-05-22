# GitHub

Use the GitHub REST API via `one github <action>`. Auth: PAT or OAuth user-flow.

## Issues
- `issues.list`, `issues.read`, `issues.create`, `issues.update` (close via `state=closed`)
- `issues.comment.create`, `issues.comment.update`, `issues.comment.delete`, `issues.comments.list`
- `issues.labels.add`, `issues.labels.replace`, `issues.assignees.add`

## Pull requests
- `pulls.list`, `pulls.read`, `pulls.create`, `pulls.update`
- `pulls.merge`, `pulls.update_branch`
- `pulls.files`, `pulls.commits`
- `pulls.review.create`, `pulls.review_comment.create`

## Repos
- `repos.read`, `repos.update`, `repos.create`

## Search
- `search.issues`, `search.repos`, `search.code`

## Branches
- `branches.list`, `branches.read`, `branches.create`, `branches.delete`

## Commits + contents
- `commits.list`, `commits.read`
- `contents.read`, `contents.write`

## Labels + milestones
- `labels.list`, `labels.create`, `labels.update`, `labels.delete`
- `milestones.list`, `milestones.create`, `milestones.update`

## Releases + users + gists
- `releases.list`, `releases.create`, `releases.update`
- `users.read`, `users.repos`
- `gists.create`, `gists.list`, `gists.update`

## Workflows
- `actions.workflow_runs.list`, `actions.workflow.dispatch`
