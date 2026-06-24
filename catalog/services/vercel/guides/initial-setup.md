---
title: Vercel — Initial Setup
---

One CLI talks to the Vercel API at `https://api.vercel.com` using an access
token (`pat` provider).

## 1. Create an access token

1. Go to https://vercel.com/account/tokens
2. **Create Token**, set a name, scope (personal account or a specific team),
   and expiration.
3. Copy the token (shown once).

## 2. Store the token

```bash
one login vercel            # --provider pat is the default
one capabilities vercel     # confirm actions are visible
```

## Notes

- For resources owned by a **team**, most actions require a `teamId` (or `slug`)
  query parameter — without it the API resolves against your personal scope and
  may 404.
- A token's scope is fixed at creation; create a separate token per team if you
  work across several.
