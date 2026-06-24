---
title: Linear — Initial Setup
---

One CLI talks to the Linear API at `https://api.linear.app` using a personal API
key (`pat` provider).

## 1. Create a personal API key

1. Go to https://linear.app/settings/api
2. Under **Personal API keys**, create a key, label it, and copy the value.

## 2. Store the key

```bash
one login linear            # --provider pat is the default
one capabilities linear     # confirm actions are visible
```

## Notes

- Linear's primary API is GraphQL at `https://api.linear.app/graphql`; the
  actions in this catalog wrap the operations they need. TODO: verify — confirm
  the exact endpoint paths used by this catalog's Linear actions.
- A personal API key carries your own permissions; you only see teams and issues
  your Linear user can access.
