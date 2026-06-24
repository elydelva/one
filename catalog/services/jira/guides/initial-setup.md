---
title: Jira — Initial Setup
---

One CLI talks to the Jira Cloud REST API. The base URL is **site-specific** —
the catalog default is `https://your-site.atlassian.net` and must point at your
own Atlassian site.

## 1. Set your site base URL

TODO: verify — confirm how your project overrides `base_url` for jira (per-site
configuration). The default `your-site.atlassian.net` is a placeholder and will
not work until it points at your real site (e.g. `https://acme.atlassian.net`).

## 2. Create an API token

1. Go to https://id.atlassian.com/manage-profile/security/api-tokens
2. **Create API token**, give it a label, and copy the value.
3. Jira Cloud authenticates with your **account email + API token** (HTTP Basic).
   TODO: verify — confirm how this catalog's `pat` injection expects the
   email:token pair to be supplied for your setup.

## 3. Store the credential

```bash
one login jira              # --provider pat is the default
one capabilities jira       # confirm actions are visible
```

## Notes

- API tokens inherit your user's permissions; a `404`/`403` usually means the
  account lacks access to that project or issue.
