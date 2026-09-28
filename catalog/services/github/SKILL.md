# GitHub

Use GitHub REST actions and Projects v2 GraphQL operations via `one github <action>`. Auth: PAT or OAuth Device Flow (`oauth2_device`).

The [full-access example](../../../examples/github-full-access/.onerc.yaml) grants every GitHub permission currently catalogued by One. Review it before copying: the GitHub OAuth request includes `delete_repo`, which can delete any repository you administer, beyond a project's local One scope. OAuth scopes grant token access at GitHub; `.onerc.yaml` independently limits actions One will execute.

After adding scopes to the GitHub OAuth app request, existing OAuth accounts must authorize again with `one login github --provider oauth2_device --as <alias>`. PATs remain available for users who prefer a token with GitHub-managed repository selection.

Run an action with `one github <action> ... --account <alias>`. Destructive actions require interactive confirmation, or `--confirm` in non-TTY runs.

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
- `repos.read`, `repos.update`, `repos.create`, `repos.delete` (`repo.delete`)

## Projects v2
- `projects.viewer.read`, `projects.list`, `projects.read`
- `projects.create`, `projects.update`, `projects.items.add`, `projects.delete`

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
- `gists.create`, `gists.list`, `gists.update`, `gists.delete`

`gists.delete` and other destructive actions require `--confirm` when run without an interactive terminal.

## Workflows
- `actions.workflow_runs.list`, `actions.workflow.dispatch`
