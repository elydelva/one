---
title: Set up a GitHub personal access token
verify:
  action: issues.read
  hint: "Try `one github issues.read --owner X --repo Y --issue_number 1` to confirm."
---

1. Go to https://github.com/settings/tokens
2. Generate a fine-grained token with `issues:read` scope.
3. Run `one login github` and paste the token when prompted.
