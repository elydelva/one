---
title: Cloudflare — Initial Setup
---

One CLI talks to the Cloudflare API at `https://api.cloudflare.com/client/v4`
using an API token (`pat` provider).

## 1. Create an API token

1. Go to https://dash.cloudflare.com/profile/api-tokens
2. **Create Token**. Start from a template or build a custom token, granting
   only the permissions the actions you use need, for example:
   - **Account → D1 → Edit** for `d1.*`
   - **Zone → DNS → Edit** for `dns_records.*`
   - **Account → Workers KV Storage → Edit** for `kv.*`
3. Scope the token to the specific account/zone resources rather than "all".
4. Copy the token (shown once).

## 2. Store the token

```bash
one login cloudflare        # --provider pat is the default
one capabilities cloudflare # confirm actions are visible
```

## Notes

- Use scoped **API Tokens**, not the legacy Global API Key.
- Most account-level actions need an `account_id`; zone-level actions need a
  `zone_id`. Find them in the dashboard URL or via `accounts.list`.
- A `403` / `Authentication error (10000)` usually means the token is missing a
  required permission — edit the token and add it.
