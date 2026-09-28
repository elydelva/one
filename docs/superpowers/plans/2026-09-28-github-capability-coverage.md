# GitHub capability coverage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expand One's GitHub catalog across its current permission groups, add user-owned Projects v2 lifecycle actions and a full-access example, then verify the flows against disposable GitHub resources.

**Architecture:** Keep GitHub operations as declarative catalog actions. Add a declared response error path so GraphQL `errors` returned with HTTP 200 fail as API errors, while fixed GraphQL operations accept only JSON-escaped typed variables. A checked-in example lists all One permissions; a live smoke procedure uses a dedicated OAuth alias and deletes standalone artifacts and its private repository.

**Tech Stack:** Go, YAML catalog definitions, One CLI, GitHub REST API and GraphQL Projects v2 API.

**Spec:** `docs/superpowers/specs/2026-09-28-github-capability-coverage-design.md`

## Global Constraints

- Keep One permission grants explicit and default-deny; the example is intentionally full access.
- OAuth scopes and One's per-project permissions remain separate controls.
- GraphQL operations are fixed in the catalog; user input is passed only through JSON-escaped variables.
- Use One CLI for all live GitHub API operations; do not use `gh api`, curl, or direct HTTP calls.
- Use a uniquely named private test repository and a dedicated OAuth alias; never persist or print credentials.
- Delete standalone temporary resources, then the test repository, and verify cleanup.
- This work covers current catalog actions plus user-owned Projects v2 lifecycle, not all of GitHub's API.
- Commit each independently discovered and fixed live-test problem before continuing.

## Review Focus

- A GraphQL response can return HTTP 200 with an `errors` array; One must return an error instead of `ok: true` (Task 1).
- Quotes, backslashes, and newlines in GraphQL variable strings must remain JSON data and never alter the fixed operation (Task 3).
- Each GraphQL operation must retain a fixed query string; only declared inputs may vary (Task 3).
- GitHub's upstream 403/404 responses must remain distinct from One's local `not_in_scope`; GitHub can hide inaccessible resources behind 404 (Task 7).
- Cleanup must still attempt gist and project deletion if an intermediate live smoke command fails; repository deletion is last (Task 7).

## Execution Notes

- Task 3's dedicated subprocess E2E scope case was not added because `go test -tags=e2e ./tests/e2e` exercises a legacy harness in which the embedded GitHub service takes precedence over local fixtures; its existing happy-path and 404 cases fail at that boundary. Scope denial is covered by the application test, and live CLI routing was exercised against GitHub.
- Live testing produced focused follow-up commits for response redaction, safe nested file paths, branch ref JSON encoding, and action account flag documentation. The full matrix and cleanup results are in [the live test report](./2026-09-28-github-capability-coverage-live-test.md).

---

### Task 1: Surface declared GraphQL response errors

**Files:**
- Modify: `pkg/catalog/types.go`
- Modify: `internal/core/action.go`
- Modify: `internal/adapters/catalog/fs.go`
- Modify: `internal/adapters/runtime/declarative.go`
- Test: `internal/adapters/catalog/fs_test.go`
- Test: `internal/adapters/runtime/declarative_test.go`

**Interfaces:**
- Produces `RequestDef.ResponseErrorsKey` and `core.RequestSpec.ResponseErrorsKey`, loaded from YAML key `request.response_errors_key`.
- The declarative runtime treats a non-empty value at that top-level response JSON key as an API failure even for a 2xx HTTP status; existing actions without the field retain current behavior.

- [x] **Step 1: Add failing catalog mapping test**

Add `TestCatalogFS_MapsResponseErrorsKey` using a temporary action with `request.response_errors_key: errors`; assert the loaded `core.Action.Request.ResponseErrorsKey` equals `errors`.

- [x] **Step 2: Verify the test fails**

Run: `go test ./internal/adapters/catalog -run TestCatalogFS_MapsResponseErrorsKey -count=1`
Expected: FAIL because the field is not represented in the loaded action.

- [x] **Step 3: Add failing runtime response test**

Add `TestDeclarative_ReturnsErrorForDeclaredResponseErrors` with a fake HTTP 200 GraphQL response containing a non-empty `errors` array and `response_errors_key: errors`. Assert execution returns an error and records the HTTP call with status 200.

- [x] **Step 4: Verify the runtime test fails**

Run: `go test ./internal/adapters/runtime -run TestDeclarative_ReturnsErrorForDeclaredResponseErrors -count=1`
Expected: FAIL because the response is currently returned as successful output.

- [x] **Step 5: Implement declared response-error handling**

Map the new field through the catalog loader. In `DeclarativeRuntime.Execute`, after reading the response body, decode the top-level JSON object and return `core.ErrAPIError` with the service, HTTP status, and response error details when the declared key is non-empty. Leave ordinary 2xx responses unchanged.

- [x] **Step 6: Run focused tests**

Run: `go test ./internal/adapters/catalog ./internal/adapters/runtime`
Expected: PASS, including existing REST response tests.

- [x] **Step 7: Commit the runtime issue fix**

Commit as `fix(runtime): surface declared API response errors`.

### Task 2: Add a full-access example and request required OAuth scopes

**Files:**
- Create: `examples/github-full-access/.onerc.yaml`
- Modify: `catalog/services/github/service.yaml`
- Modify: `internal/adapters/catalog/embed_test.go`
- Create: `internal/adapters/catalog/github_example_test.go`

**Interfaces:**
- The example grants all current GitHub permission paths plus `repo.delete` from Task 4 and `projects.read` / `projects.write` from Task 3.
- Device Flow requests `repo`, `project`, `gist`, `workflow`, and `delete_repo` scopes. `delete_repo` permits deletion of any repository the account can administer, independently of the current project's One allowlist.

- [x] **Step 1: Add failing OAuth scope assertion**

Update `TestCatalogEmbed_GitHubUsesDeviceOAuthForPrivateRepositories` to assert the exact scope set `repo`, `project`, `gist`, `workflow`, and `delete_repo` independent of ordering.

- [x] **Step 2: Verify the scope test fails**

Run: `go test ./internal/adapters/catalog -run TestCatalogEmbed_GitHubUsesDeviceOAuthForPrivateRepositories -count=1`
Expected: FAIL because the catalog currently requests only `repo`.

- [x] **Step 3: Add failing example completeness test**

Add `TestGitHubFullAccessExampleAllowsEveryCatalogPermission`. Load the YAML example and embedded GitHub service, collect each action's permission, and assert each permission is explicitly present in `services.github.allow`.

- [x] **Step 4: Verify the example test fails**

Run: `go test ./internal/adapters/catalog -run TestGitHubFullAccessExampleAllowsEveryCatalogPermission -count=1`
Expected: FAIL because the example does not exist.

- [x] **Step 5: Add OAuth scopes and the explicit permission example**

Set the five scopes in `catalog/services/github/service.yaml`. Add one `allow` entry for every permission used by the GitHub catalog, including `repo.delete`, `projects.read`, and `projects.write`; keep the example intentionally broad and self-describing. Label the account-wide deletion effect in its comment.

- [x] **Step 6: Run focused catalog tests**

Run: `go test ./internal/adapters/catalog`
Expected: PASS and the example includes every permission used by the loaded catalog, with entries for the `repo.delete` and Projects permissions added in Tasks 3 and 4.

- [x] **Step 7: Commit the scope and example change**

Commit as `feat(github): configure full-access scopes and example`.

### Task 3: Add safe user-owned Projects v2 actions

**Files:**
- Create: `catalog/services/github/actions/projects.viewer.read.yaml`
- Create: `catalog/services/github/actions/projects.list.yaml`
- Create: `catalog/services/github/actions/projects.read.yaml`
- Create: `catalog/services/github/actions/projects.create.yaml`
- Create: `catalog/services/github/actions/projects.update.yaml`
- Create: `catalog/services/github/actions/projects.items.add.yaml`
- Create: `catalog/services/github/actions/projects.delete.yaml`
- Modify: `internal/adapters/catalog/embed_test.go`
- Modify: `internal/app/execute_test.go`
- Modify: `tests/e2e/execute_test.go`
- Modify: `internal/testing/fixture/catalog/v1-minimal-e2e/github/actions/projects.read.yaml`
- Test: `internal/adapters/runtime/declarative_test.go`

**Interfaces:**
- Read actions use `projects.read`; mutations use `projects.write`.
- Operations POST to `/graphql`, declare `request.response_errors_key: errors`, and embed fixed query/mutation text in the catalog body template.
- Variable values use the existing `{name|json}` interpolation filter.
- `projects.viewer.read` returns the authenticated viewer's GraphQL node ID; `projects.list` lists the viewer's first 100 projects; `projects.read` loads a project by `project_id`; `projects.create` accepts `title`; `projects.update` accepts `project_id` and `title`; `projects.items.add` accepts `project_id` and `content_id`; `projects.delete` accepts `project_id` and is marked destructive.

- [x] **Step 1: Add failing catalog action tests**

Add `TestCatalogEmbed_GitHubProjectV2Actions` assertions for all seven action IDs, permission paths, POST `/graphql`, and `errors` response key.

- [x] **Step 2: Verify the catalog tests fail**

Run: `go test ./internal/adapters/catalog -run 'TestCatalogEmbed_GitHubProjectV2Actions' -count=1`
Expected: FAIL because the Projects v2 actions are absent.

- [x] **Step 3: Add a failing JSON-escaping runtime test**

Add `TestDeclarative_GraphQLVariablesAreJSONEscaped` with quote, backslash, and newline characters in a title input. Assert the captured GraphQL body decodes as JSON and the original string appears only in `variables`, while the fixed query remains unchanged.

- [x] **Step 4: Verify the runtime test fails**

Run: `go test ./internal/adapters/runtime -run TestDeclarative_GraphQLVariablesAreJSONEscaped -count=1`
Expected: FAIL if the catalog operation body is absent or interpolation is not JSON-safe.

- [x] **Step 5: Add the seven Projects v2 action definitions**

Use fixed GraphQL operations and typed action inputs. Mark viewer/list/read as reads, create/update/add as writes, and delete as destructive. Keep mutation inputs required where a fixed operation needs them.

- [x] **Step 6: Add One scope and CLI routing checks**

Add `TestExecute_ProjectsActionRequiresProjectsRead` in `internal/app/execute_test.go`, then add an E2E case that loads a Projects read fixture action, returns a fixture GraphQL response over the local fake API, and proves `projects.read` can execute when explicitly allowed and is denied when absent.

- [x] **Step 7: Run focused catalog, app, E2E, and runtime tests**

Run: `go test ./internal/adapters/catalog ./internal/adapters/runtime ./internal/app ./tests/e2e`
Expected: PASS, including GraphQL error-envelope handling from Task 1.

- [x] **Step 8: Commit the missing Projects v2 capability**

Commit as `feat(github): add user-owned Projects v2 actions`.

### Task 4: Add repository deletion for complete test cleanup

**Files:**
- Create: `catalog/services/github/actions/repos.delete.yaml`
- Modify: `internal/adapters/catalog/embed_test.go`

- [x] **Step 1: Add failing repository delete catalog test**

Add `TestCatalogEmbed_GitHubRepoDelete` asserting `repos.delete` uses a distinct `repo.delete` permission, is marked `destructive`, sends HTTP DELETE to `/repos/{owner}/{repo}`, and requires both path inputs.

- [x] **Step 2: Verify the test fails**

Run: `go test ./internal/adapters/catalog -run TestCatalogEmbed_GitHubRepoDelete -count=1`
Expected: FAIL because repository deletion is not catalogued.

- [x] **Step 3: Add the destructive repository delete action**

Define the declarative action with `repo.delete` permission and map 401, 403, and 404. Keep deletion classified as destructive.

- [x] **Step 4: Run focused catalog tests**

Run: `go test ./internal/adapters/catalog`
Expected: PASS.

- [x] **Step 5: Commit the cleanup capability**

Commit as `feat(github): add repository deletion action`.

### Task 5: Add gist deletion for complete test cleanup

**Files:**
- Create: `catalog/services/github/actions/gists.delete.yaml`
- Modify: `catalog/services/github/SKILL.md`
- Modify: `internal/adapters/catalog/embed_test.go`

- [x] **Step 1: Add failing gist delete catalog test**

Add `TestCatalogEmbed_GitHubGistDelete` asserting `gists.delete` uses `gist.write`, is marked `destructive`, sends HTTP DELETE to `/gists/{gist_id}`, and requires `gist_id`.

- [x] **Step 2: Verify the test fails**

Run: `go test ./internal/adapters/catalog -run TestCatalogEmbed_GitHubGistDelete -count=1`
Expected: FAIL because gist deletion is not catalogued.

- [x] **Step 3: Add and document the delete action**

Define the declarative delete action as `destructive`, map 401, 403, and 404, then list it in the GitHub skill's Gists section. Document the CLI `--confirm` requirement for non-TTY runs.

- [x] **Step 4: Run focused catalog tests**

Run: `go test ./internal/adapters/catalog`
Expected: PASS.

- [x] **Step 5: Commit the cleanup capability**

Commit as `feat(github): add gist deletion action`.

### Task 6: Document the full-access example and GitHub auth model

**Files:**
- Modify: `catalog/services/github/SKILL.md`
- Modify: `catalog/services/github/guides/initial-setup.md`
- Modify: `docs/AUTH.md`
- Modify: `README.md`

- [x] **Step 1: Document the example and permission groups**

Add a short link to `examples/github-full-access/.onerc.yaml`, enumerate Projects v2 actions, explain `one --account <alias> github <action>`, note that destructive actions require interactive confirmation or `--confirm` in non-TTY runs, and state clearly that the example enables all catalogued GitHub permissions.

- [x] **Step 2: Document OAuth scopes and reauthorization**

Explain that existing OAuth accounts must authorize the expanded request again, that PATs remain an option, and that OAuth scopes are broader token grants while `.onerc.yaml` remains One's local action boundary.

- [x] **Step 3: Validate catalog and documentation**

Run: `go test ./internal/adapters/catalog && git diff --check`
Expected: PASS with no whitespace errors.

- [x] **Step 4: Commit the user-facing guidance**

Commit as `docs(github): explain full access and Projects v2`.

### Task 7: Run the live GitHub smoke matrix and clean up

**Files:**
- Create: `docs/superpowers/plans/2026-09-28-github-capability-coverage-live-test.md`
- Use: `examples/github-full-access/.onerc.yaml`
- Use: built local One binary from the current branch

- [x] **Step 1: Build and verify the branch binary**

Run: `go build -o /tmp/one-github-capability-test ./cmd/one` and `/tmp/one-github-capability-test --version`.
Expected: the build passes and the binary reports the current commit's version.

- [x] **Step 2: Reauthorize a dedicated One account alias**

Run `/tmp/one-github-capability-test login github --provider oauth2_device --as full-test` and have the user authorize the displayed code. Verify only that the alias exists using `/tmp/one-github-capability-test accounts github`; never display the token.

- [x] **Step 3: Create an isolated local test project and private repository**

Use a fresh temporary directory with the checked-in full-access `.onerc.yaml`, then pass `--account full-test` to every action invocation. Create a uniquely named private repository through One's `github repos.create` action with `--private true`. Record only the repository name and returned ID needed for cleanup.

- [x] **Step 4: Exercise each permission group through One**

Run one representative action for each of the 11 existing permission groups, plus the new `repo.delete` permission: `actions.read`, `actions.write`, `gist.read`, `gist.write`, `issues.read`, `issues.write`, `pulls.read`, `pulls.write`, `repo.read`, `repo.write`, `user.read`, and `repo.delete`; then create/read/update/delete a user-owned Project v2 and add the temporary repository's issue to it.

- [x] **Step 5: Commit each newly found live defect before resuming**

For each defect: reproduce through One, add a failing test, fix it, run the narrow tests and `make test && make lint`, commit the focused fix, rebuild the branch binary, and resume the matrix. Record exact action, safe error summary, fix commit, and retest result in the live-test report; omit tokens and private contents.

- [x] **Step 6: Clean up and verify cleanup**

Run cleanup even when an earlier matrix action fails: attempt `gists.delete --confirm` and `projects.delete --confirm` first, then delete the private repository with One's `github repos.delete --confirm` action. Verify each resource is gone through One read actions, and remove the local temporary directory after preserving the sanitized report. If a cleanup call fails, keep its resource ID only in the local test session and retry until cleanup succeeds.

- [x] **Step 7: Run repository quality gates**

Run: `make test && make lint`
Expected: both pass after the live smoke run and any focused fixes.

- [x] **Step 8: Commit the sanitized live-test report**

Commit as `test(github): record disposable capability smoke run`.
