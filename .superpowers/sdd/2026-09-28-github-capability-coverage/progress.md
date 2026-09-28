# SDD ledger — plan: docs/superpowers/plans/2026-09-28-github-capability-coverage.md

## Setup
Ruling: use the explicitly requested feature branch in the existing checkout instead of creating a second git worktree — this branch is already isolated from main and is the user's requested delivery surface — cost if wrong: no separate filesystem checkout protects the feature branch from unrelated workspace edits.

Pre-flight: Task 1 produces `RequestErrorsKey` on the catalog/runtime action contract, which Task 3 consumes in each GraphQL action's response metadata; the mapping and runtime behavior must land before Project v2 definitions.
Pre-flight: Task 2 creates the full-access example with future `projects.*` and `repo.delete` permissions; Task 3 and Task 4 add those actions later, after which the completeness test validates them against loaded catalog permissions.
Pre-flight: Tasks 3, 4, and 5 produce Project v2 actions, repository deletion, and gist deletion respectively; Task 7 consumes these for live tests and cleanup.

## Tasks
- [ ] Task 1: Surface declared GraphQL response errors
- [ ] Task 2: Add a full-access example and request required OAuth scopes
- [ ] Task 3: Add safe user-owned Projects v2 actions
- [ ] Task 4: Add repository deletion for complete test cleanup
- [ ] Task 5: Add gist deletion for complete test cleanup
- [ ] Task 6: Document the full-access example and GitHub auth model
- [ ] Task 7: Run the live GitHub smoke matrix and clean up

Task 1: RED observed — catalog test initially failed to compile because `ResponseErrorsKey` was missing; runtime test independently failed to compile for the same missing contract field. GREEN observed — both focused tests passed after mapping the key and surfacing non-empty response error envelopes as `ErrAPIError`; `make test` and `make lint` passed (0 lint issues).
Task 1: complete (commits 2a4e552..719d8f1, tests: make test → ok  	elydelva/one/tests/security	(cached))

Task 2: RED observed — the OAuth scope test reported missing `project`, `gist`, `workflow`, and `delete_repo`; the example-completeness test reported the absent example. GREEN observed — both focused tests and `go test ./internal/adapters/catalog` passed after the five scopes and explicit permission example were added; `make test` passed and `make lint` reported 0 issues.
Task 2: complete (commits 719d8f1..d219e69, tests: go test ./internal/adapters/catalog → ok  	elydelva/one/internal/adapters/catalog	(cached))

Task 3 Ruling: the planned CLI subprocess E2E check uses `-tags=e2e`, but the existing harness cannot route its GitHub fixture because the embedded GitHub service shadows fixture URLs; tagged legacy happy-path and 404 tests also fail for that reason. Keep executable scope evidence at the app layer for this task and report the E2E limitation instead of widening catalog precedence semantics beyond the approved feature. Cost if wrong: there is no CLI-process proof for the new Projects action in this task.
Task 3: RED observed — catalog tests failed until the seven Projects v2 action definitions were added; runtime test failed on unknown operation. GREEN observed — project action contract tests, JSON-safe variable encoding, scope denial, full catalog/runtime/app suite, make test, and make lint passed.
Task 3: complete (commits d219e69..88bd77c, tests: go test ./internal/adapters/catalog ./internal/adapters/runtime ./internal/app → ok  	elydelva/one/internal/app	(cached))
Task 4: RED observed — repos.delete catalog test failed because the action was unknown. GREEN observed — DELETE route, repo.delete permission, destructive classification, required owner/repo inputs, and 401/403/404 mappings pass; catalog tests, make test, and make lint passed.
Task 4: complete (commits 88bd77c..bf9ac67, tests: go test ./internal/adapters/catalog → ok  	elydelva/one/internal/adapters/catalog	(cached))
Task 5: RED observed — gists.delete catalog test failed because the action was unknown. GREEN observed — destructive DELETE action, gist.write permission, required gist_id, error mappings, skill listing and non-TTY confirmation note pass; catalog tests, make test, and make lint passed.
Task 5: complete (commits bf9ac67..ecbc7f1, tests: go test ./internal/adapters/catalog → ok  	elydelva/one/internal/adapters/catalog	(cached))
Task 6: Documented the full-access scope file, GitHub REST/Projects v2 actions, alias execution, destructive confirmation, OAuth scope breadth, reauthorization, and PAT alternatives in the GitHub skill, setup guide, AUTH.md, and README. Validation: catalog tests and git diff --check passed.
Task 6: complete (commits ecbc7f1..8d50ab2, tests: go test ./internal/adapters/catalog → ok  	elydelva/one/internal/adapters/catalog	(cached))
Task 7 live finding: GitHub repos.read returned `temp_clone_token` in a successful repository response. The disposable repository was deleted immediately through `repos.delete --confirm`, and `repos.read` confirmed not found. Added opt-in recursive response-field redaction, configured it for repos.read, and verified nested/top-level redaction with tests; make test, make lint, and git diff --check passed. No token value is retained here.
Task 7 live finding: contents.write rejected `.github/workflows/one-smoke.yml` as path traversal because all slash-containing path values were blocked. Added explicit `{path|path}` interpolation that preserves safe nested directories and URL-escapes segments while rejecting absolute, empty, dot, dot-dot, encoded traversal, and backslash segments. Focused path tests, make test, make lint, and diff check passed.
