# GitHub capability smoke test

Date: 2026-09-28  
Branch: `feat/github-capability-coverage`  
Authentication: GitHub OAuth Device Flow succeeded with the dedicated One alias `full-test`. No credential or token value is recorded here.

The run used a temporary local project with `examples/github-full-access/.onerc.yaml` and a private, disposable GitHub repository. The tested actions were invoked through One with `--account full-test`.

| Permission group | Live actions | Result |
|---|---|---|
| `actions.read` | `actions.workflow_runs.list` | Passed; listed the dispatched run. |
| `actions.write` | `actions.workflow.dispatch` | Passed; dispatched the temporary `workflow_dispatch` workflow. |
| `gist.read` | `gists.list` | Passed; listed the authenticated user's gists. |
| `gist.write` | `gists.create`, `gists.update`, `gists.delete` | Passed; the disposable gist was absent from a subsequent list. |
| `issues.read` | `issues.read` | Passed; read the issue created during the run. |
| `issues.write` | `issues.create` | Passed. |
| `pulls.read` | `pulls.list`, `pulls.read` | Passed; read the disposable pull request. |
| `pulls.write` | `pulls.create` | Passed; created a pull request from a temporary branch. |
| `projects.read` | `projects.viewer.read`, `projects.list`, `projects.read` | Passed; the temporary Project v2 was present before cleanup. |
| `projects.write` | `projects.create`, `projects.update`, `projects.items.add`, `projects.delete` | Passed; deletion succeeded and a subsequent list no longer contained the project. |
| `repo.read` | `repos.read`, `commits.list` | Passed; repository and commit data were readable. `repos.read` now redacts GitHub's `temp_clone_token` field. |
| `repo.write` | `contents.write`, `branches.create` | Passed; created a workflow file, a branch, and a file on that branch. |
| `repo.delete` | `repos.delete --confirm` | Passed; deleted the private repository, and a subsequent `repos.read` returned not found. |
| `user.read` | `users.read` | Passed. |

The temporary gist and Project v2 were deleted before the private repository. Repository deletion removed the associated issue, pull request, branch, file, and workflow. The local test directory and temporary runner files were removed after this report was written.

## Live findings and fixes

- `repos.read` returned GitHub's `temp_clone_token` unfiltered. The disposable repository was deleted immediately. Commit `c63156c` added recursive JSON-field redaction and configured it for this action; a runtime regression test covers top-level and nested fields. Retest: passed, with sensitive values excluded from CLI output and this report.
- `contents.write` rejected `.github/workflows/one-smoke.yml` as path traversal. Commit `f9c9057` added a `{path|path}` interpolation filter that retains safe directory separators while rejecting unsafe path segments. Retest: workflow file creation passed.
- `branches.create` produced malformed JSON because its template wrapped an already JSON-encoded branch value in quotes. Commit `fb6f11b` corrected the body template and added a dry-run JSON decoding test. Retest: branch creation passed.
- The documented position of the action-level `--account` option selected `default` instead of `full-test`. Commit `5c07b80` corrected the GitHub skill and README examples. The failed invocation created no resource; subsequent actions used the working syntax `one github <action> ... --account full-test`.

After the fixes, `make test` and `make lint` passed. The tagged subprocess suite `go test -tags=e2e ./tests/e2e` still cannot reliably exercise GitHub fixture overrides because the embedded GitHub service wins over local fixtures; its existing happy-path and 404 cases fail for that pre-existing catalog-precedence mismatch. Project scope enforcement is covered by the passing application test instead.
