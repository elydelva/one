---
title: Stripe — Initial Setup
---

One CLI talks to the Stripe API at `https://api.stripe.com` using a secret API
key (`pat` provider).

## 1. Get a secret API key

1. Go to https://dashboard.stripe.com/apikeys
2. Use a **test mode** key (`sk_test_…`) while developing; switch to a live key
   (`sk_live_…`) only when you intend to move real money.
3. Prefer a **restricted key** (Create restricted key) scoped to just the
   resources the actions you call need, instead of the full secret key.

## 2. Store the key

```bash
one login stripe            # --provider pat is the default
one capabilities stripe     # confirm actions are visible
```

## Notes

- Test and live keys hit the same base URL; the key itself selects the mode.
- Treat a live secret key like a password — it can charge cards and move funds.
- TODO: verify — if you need webhook events (e.g. for `events.*`), create the
  endpoint at https://dashboard.stripe.com/webhooks and confirm the signing
  secret handling for your use case.
