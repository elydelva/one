---
title: GitHub — Initial Setup
---

One CLI talks to the GitHub REST API at `https://api.github.com`. Two auth
options are supported: a Personal Access Token (`pat`, simplest) or an OAuth
user flow (`oauth2_user`).

## Option A — Personal Access Token (recommended for local use)

1. Go to https://github.com/settings/personal-access-tokens (fine-grained) or
   https://github.com/settings/tokens (classic).
2. Create a token. Grant only the scopes the actions you use need:
   - Repository contents / issues / pull requests — read or write as required.
   - `workflow` — only if you dispatch or read Actions workflows.
3. Copy the token (fine-grained tokens start with `github_pat_`, classic with `ghp_`).
4. Store it in the vault:

   ```bash
   one login github            # --provider pat is the default
   one capabilities github     # confirm actions are visible
   ```

## Option B — OAuth user flow

Use `one login github --provider oauth2_user` if your project is configured for
an OAuth app. This opens a browser to authorize and stores the resulting token
in the vault.

## Notes

- A fine-grained token only reaches repositories you explicitly select — if a
  call 404s, check the token's repository access first.
- GitHub returns `404` (not `403`) for resources a token cannot see, to avoid
  leaking their existence.
