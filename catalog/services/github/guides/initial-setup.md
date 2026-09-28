---
title: GitHub — Initial Setup
---

One CLI talks to the GitHub REST API at `https://api.github.com`. You can sign
in with a Personal Access Token (`pat`) or the GitHub OAuth Device Flow
(`oauth2_device`). The Device Flow needs a client ID from a GitHub OAuth App
with Device Flow enabled. One's public client ID is included in the catalog.

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

## Option B — OAuth Device Flow

Run `one login github --provider oauth2_device`. One opens the GitHub device
authorization page, displays a short code, and waits while you approve it.
The `repo` OAuth scope supports private repositories. One stores the resulting
credential in the local OS keychain.

## Notes

- A fine-grained token only reaches repositories you explicitly select — if a
  call 404s, check the token's repository access first.
- GitHub returns `404` (not `403`) for resources a token cannot see, to avoid
  leaking their existence.
