# GitHub capability coverage in One

## Goal

Make One's GitHub integration usable across every permission group represented
by its catalog, provide a clearly labeled full-access example, and prove the
main read/write flows against a disposable GitHub repository and Projects v2
project.

## Current state

The embedded GitHub catalog has 52 actions under 11 One permissions:
`actions.read`, `actions.write`, `gist.read`, `gist.write`, `issues.read`,
`issues.write`, `pulls.read`, `pulls.write`, `repo.read`, `repo.write`, and
`user.read`. There is no full-access example and no GitHub Projects action.

One's permission file is an execution boundary for the current project. It does
not narrow the GitHub token itself. OAuth `repo` is broad; GitHub OAuth Apps do
not provide the per-resource permission model of GitHub Apps.

## Design

Add `examples/github-full-access/.onerc.yaml` as an intentionally broad,
explicit example listing every permission used by the GitHub catalog, including
the Projects permissions added here. Keep `allow` entries explicit so readers
can remove capabilities they do not want. Document that OAuth scopes and One
permissions are separate controls.

Add a destructive `repos.delete` action under a separate `repo.delete`
permission so One can clean up the disposable test repository without treating
repository deletion as an ordinary write. Deletion of repositories requires
the explicit GitHub OAuth `delete_repo` scope; make this elevated account-wide
capability clear in the setup guide and full-access example.

Add focused Projects v2 actions over GitHub's GraphQL endpoint: list and read
the authenticated user's projects, create a user-owned project, add an
existing issue or pull request, and delete a project for lifecycle cleanup.
Also support updating a project's title.
Use a dedicated One permission pair `projects.read` / `projects.write` so a
project can grant read access without granting mutations. Keep GraphQL
operation text fixed in catalog actions and expose only typed variables as
inputs; do not expose an arbitrary query input that bypasses One's action
allowlist.

Expand GitHub Device Flow scopes to `repo`, `project`, `gist`, `workflow`, and
`delete_repo` to cover private repository access, Projects v2, gists, workflow
dispatch, and repository deletion. `delete_repo` allows the OAuth token to
delete any repository the user can administer, even when a particular One
project does not allow `repo.delete`. Existing OAuth accounts do not gain
scopes automatically; users must authorize the updated request again. PAT
login remains supported, and GitHub's token permissions still depend on the
token type and its grants.

## Verification

Add catalog tests that ensure the example covers every GitHub permission and
that each Projects action has a fixed GraphQL operation, the expected
permission, and declared inputs. Add One CLI integration checks for permission
denials and successful action routing.

Run a live, opt-in manual matrix through One using a dedicated OAuth alias and
a randomly named private repository. Exercise representative reads and writes
for all 11 existing permission groups, plus `repo.delete` and Projects v2
create/read/update/add/delete.
Create only disposable issues, branches, pull requests, a gist, and a minimal
workflow as needed to exercise operations. Delete standalone resources
explicitly and delete the test repository last; verify cleanup. Never use `gh`
or direct HTTP calls to execute the service operations. Do not print, commit,
or otherwise persist credentials or the temporary repository's sensitive
contents.

## Boundaries

This delivers the current catalog's capabilities plus basic Projects v2
lifecycle, not the entire GitHub REST or GraphQL API. Organization-owned
Projects v2, organization administration, webhooks, security APIs, and other
uncatalogued GitHub surfaces remain out of scope. A live test failure should
be fixed and committed separately before continuing, as requested.

## References

- [GitHub OAuth scopes](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/scopes-for-oauth-apps)
- [GitHub Projects GraphQL reference](https://docs.github.com/en/graphql/reference/projects)
- [Managing Projects with the API](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/using-the-api-to-manage-projects)
