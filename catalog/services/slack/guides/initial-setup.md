---
title: Slack — Initial Setup
---

One CLI talks to the Slack Web API at `https://slack.com/api` using a token
(`pat` provider).

## 1. Create a Slack app and get a token

1. Go to https://api.slack.com/apps → **Create New App** (From scratch), pick a
   workspace.
2. Under **OAuth & Permissions**, add the **Bot Token Scopes** the actions you
   use need, for example:
   - `channels:read`, `groups:read` — `conversations.list`
   - `chat:write` — `chat.postMessage`
   - `files:read` — `files.list`
3. Click **Install to Workspace** and authorize.
4. Copy the **Bot User OAuth Token** (`xoxb-…`). For user-level actions you may
   instead need a User OAuth Token (`xoxp-…`).

## 2. Store the token

```bash
one login slack             # --provider pat is the default
one capabilities slack      # confirm actions are visible
```

## Notes

- A method failing with `not_in_channel` means the bot must be invited to the
  channel (`/invite @your-app`).
- `missing_scope` errors name the exact scope to add under OAuth & Permissions,
  after which you must reinstall the app.
- TODO: verify — choose bot vs user token based on whether the actions you call
  act as the app or as a user.
