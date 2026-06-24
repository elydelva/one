---
title: Sentry — Initial Setup
---

One CLI talks to the Sentry API at `https://sentry.io/api/0` using an auth token
(`pat` provider).

## 1. Create an auth token

1. Go to https://sentry.io/settings/account/api/auth-tokens/ (user token) or
   create an **Internal Integration** token under your organization settings for
   a scoped, org-owned token.
2. Grant the scopes the actions you use need, for example:
   - `project:read`, `event:read` — reading issues/events
   - `project:write` — mutating project resources
3. Copy the token.

## 2. Store the token

```bash
one login sentry            # --provider pat is the default
one capabilities sentry     # confirm actions are visible
```

## Notes

- Most endpoints are scoped to an organization (and often a project) — you will
  typically pass an `organization_slug` / `project_slug` as action inputs.
- TODO: verify — if you self-host Sentry, the base URL differs from
  `sentry.io/api/0`; confirm how your project overrides it.
