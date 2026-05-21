# AUTH.md

> Complete reference for authentication and credential management in One CLI. For the auth declaration format in a service, see [CATALOG.md](./CATALOG.md). For global security, see [SECURITY.md](./SECURITY.md).

## Overview

Auth in One CLI does not mean "one single thing". Six authentication models are supported, because they all exist in the wild and none can be cleanly substituted by another.

| Scheme | Examples | Renewal | User setup |
|---|---|---|---|
| OAuth 2.0 user-flow | Notion, Linear, Google, Slack | refresh token | browser callback |
| OAuth 2.0 device flow | GitHub, Microsoft, Google (CLI) | refresh token | code + manual URL |
| OAuth 2.0 client credentials | Twitch app, Reddit script | re-fetch | client_id + secret |
| Static API key | Stripe, OpenAI, Anthropic, Resend | never | copy-paste |
| Personal Access Token | GitHub, GitLab, Notion (legacy) | never or manual | web generation + copy |
| AWS-style signature | AWS, Cloudflare R2, MinIO | never | access + secret + region |
| Mutual TLS / certificate | private services, registries | never | key file |

## The classic trap: OAuth for everything

Many systems try to route everything through OAuth. Bad idea. Stripe **does not want** OAuth for internal tools, their DX is "copy your key". GitHub has an OAuth but developers often prefer a scoped PAT.

**The system must follow the nature of each service**, not force it into a single model.

## The generic model: `auth.providers`

A service declares **one or more auth providers** in its `service.yaml`. The user chooses at login.

```yaml
auth:
  default_provider: oauth
  providers:
    oauth:
      type: oauth2_user
      # ...
    pat:
      type: token_paste
      # ...
```

At login:

```
$ one login github
GitHub supports two authentication methods:

  1. OAuth (recommended) - opens browser, no copying needed
  2. Personal Access Token - paste an existing token

Choose [1]: _
```

With `--provider` for scripting: `one login github --provider pat`.

## Provider types

### `oauth2_user`

Standard OAuth 2.0 flow with PKCE + local server for the callback.

```yaml
oauth:
  type: oauth2_user
  authorize_url: https://api.notion.com/v1/oauth/authorize
  token_url: https://api.notion.com/v1/oauth/token
  client_id: "{env.ONE_NOTION_CLIENT_ID}"
  scopes: [read, write]              # optional, depends on the service
  pkce: true                          # recommended
  callback:
    mode: local_server
    path: /callback
    port: ephemeral                  # or explicit: 54287
  refresh:
    supported: true
    rotation: true                   # if the service rotates the refresh token
  injection:
    header: Authorization
    format: "Bearer {access_token}"
    inject: auto                     # the binary injects, the handler does not touch it
  validate:
    method: GET
    url: "{api.base_url}/me"
    expect_status: 200
```

#### Flow in detail

1. **Resolve config.** Load `service.yaml`, select the provider, read `client_id`. If missing: clear error.

2. **Generate PKCE.** `code_verifier` (43-128 chars random) and `code_challenge` (SHA256 of verifier, base64url).

3. **Bind local server.** `net.Listen("tcp", "127.0.0.1:0")` → ephemeral port. Serves a unique handler on `/callback`.

4. **Build authorize URL.** With `state` (CSRF protection), `code_challenge`, `redirect_uri`, scopes.

5. **Open browser.** Via `open` (macOS), `xdg-open` (Linux), `start` (Windows). Fallback: display the URL.

6. **Wait callback.** Timeout 5 minutes.

7. **Handle callback.** Verify `state`, extract `code`. POST to `token_url` with `code` + `code_verifier`. Receives `access_token`, `refresh_token`, `expires_in`, `scope`.

8. **Render success page.** Plain text "Login complete. You can close this window."

9. **Store.** `vault.Store({service, alias}, credential)`. The alias comes from the `--as` flag (default `default`). No interactive prompt for the alias.

```
$ one login notion --as kaampus
Open this URL to continue:
  https://api.notion.com/v1/oauth/authorize?...
```

(Post-login validation via `validate_url`: reserved for `token_paste` / `api_key` / `aws_keys` providers; not applied to `oauth2_user` currently.)

### `oauth2_device`

For contexts without a browser (headless SSH, remote terminals). RFC 8628.

```yaml
oauth:
  type: oauth2_device
  device_authorization_url: https://github.com/login/device/code
  token_url: https://github.com/login/oauth/access_token
  client_id: "{env.ONE_GITHUB_CLIENT_ID}"
  scopes: [repo, read:org]
```

Flow:

```
$ one login github --device
To authenticate, visit:
  https://github.com/login/device

And enter the code:
  ABCD-1234

Waiting for authorization... (5 minutes)
```

The binary polls the `token_url` at the interval announced by the server (`interval`, default 5s, increased by 5s on `slow_down`) until approval or expiration of the `device_code`.

### `oauth2_client`

Client credentials, machine-to-machine. No user, just an app.

```yaml
m2m:
  type: oauth2_client
  token_url: https://api.example.com/oauth/token
  scopes: [api.read, api.write]
  injection:
    header: Authorization
    format: "Bearer {access_token}"
```

The user provides `client_id` and `client_secret` once, the binary uses them to obtain an access token. Automatic renewal.

### `token_paste`

The user copies and pastes a static token.

```yaml
pat:
  type: token_paste
  label: Personal Access Token
  help_url: https://github.com/settings/tokens?type=beta
  prompt: "Paste your token (input hidden):"
  validate:
    method: GET
    url: "{api.base_url}/user"
    expect_status: 200
  injection:
    header: Authorization
    format: "Bearer {token}"
```

Flow:

```
$ one login github --provider pat
Opening browser to generate a token...
URL: https://github.com/settings/tokens?type=beta

Paste your token (input hidden): ●●●●●●●●●●●●●●●●

Validating...
✓ Authenticated as elydelva

Save this account as [default]: work
✓ Stored github:work in keychain
```

Input is masked (no-echo, like `sudo` password).

### `api_key`

Variant of `token_paste` for static API keys (Stripe, OpenAI, etc.).

```yaml
api_key:
  type: api_key
  label: API key
  help_url: https://dashboard.stripe.com/apikeys
  prompt: "Paste your API key:"
  validate:
    method: GET
    url: "{api.base_url}/charges?limit=1"
    expect_status: 200
  injection:
    header: Authorization
    format: "Bearer {key}"
```

Identical to `token_paste` from a UX standpoint, but semantically different (not a user-scoped PAT, it is an app key).

### `aws_keys`

Access key ID + secret + optional session token. Three no-echo prompts.

Flow:

```
$ one login aws --as default
AWS access key ID (input hidden): ●●●●●●●●●●
AWS secret access key (input hidden): ●●●●●●●●●●
AWS session token (optional; press enter to skip): _
```

Validation: custom SigV4 signature (without `aws-sdk-go`) on `sts.amazonaws.com` (region `us-east-1`) → `GetCallerIdentity`. HTTP failure ≥ 300 → error.

Storage: `AccessToken = access_key_id`, `RefreshToken = JSON{secret, session_token}`. No region field: if a region is required by the service, it comes from the `service.yaml`.

### `certificate`

mTLS. No prompt (PEMs are too large to paste). The binary reads two files via environment variables:

```
ONE_CERT_<SERVICE>_<ACCOUNT>_CERT=/path/to/cert.pem
ONE_CERT_<SERVICE>_<ACCOUNT>_KEY=/path/to/key.pem
```

(`<SERVICE>` and `<ACCOUNT>` upper-cased, characters not matching `[A-Z0-9]` replaced by `_`.)

Validation: `tls.X509KeyPair` at login. Storage: `AccessToken = cert PEM`, `RefreshToken = key PEM`. `Refresh` is a no-op (re-login to rotate).

## The `Credential` type

```go
// core/credential.go
type Credential struct {
    Service      ServiceID
    Account      AccountAlias    // "work", "perso", "default"
    Provider     ProviderKind    // typed: ProviderOAuthUser, ProviderPAT, ...
    AccessToken  Secret
    RefreshToken Secret          // optional (or secondary field depending on provider)
    ExpiresAt    *time.Time      // nil = no expiration
    Scopes       []string
}
```

No `Extras`, `CreatedAt`, `LastUsedAt` in v0.4: providers that need a second secret (AWS session token, mTLS key, OAuth client secret) store it in `RefreshToken` in the appropriate format.

The `Secret` type masks the value in all logs/errors. To reveal it: `secret.Reveal()`. Only call this at the point of injection into an HTTP header.

## Multi-account

A service can have N accounts. The vault is indexed by `(service, alias)`.

### Creating a new account

The alias is passed via the `--as` flag (default `default`). No prompt.

```bash
$ one login github --as work
$ one login github --as perso
```

### Listing accounts for a service

```bash
$ one accounts github
work    elydelva@protonmail.com     authenticated   refresh in 1h2m
perso   ely.delvallee@gmail.com     authenticated   refresh in 23m
```

### Selecting an account

`--as <alias>` (or `--account <alias>`) on the execution command. If absent, the `default` alias is used.

```bash
one github issues.list --as perso
```

### Deleting an account

```bash
$ one logout github --as perso
```

## The vault

Local secure storage for credentials. Three possible sources, chained by priority.

### Source 1: environment variables

```bash
ONE_CREDS_GITHUB_DEFAULT='{"access_token":"...","provider":"pat"}'
```

Total vault override, useful in CI. Serialized JSON format of `Credential`.

### Source 2: OS native keychain

Implemented via [`zalando/go-keyring`](https://github.com/zalando/go-keyring) which abstracts:

- **macOS**: Keychain via Security framework
- **Linux**: Secret Service via libsecret (GNOME Keyring, KWallet)
- **Windows**: Credential Manager via wincred

Structure in the keychain:

- **Service name** (keychain field): `one`
- **Account name** (keychain field): `<service>:<account_alias>` (e.g. `github:work`)
- **Password**: serialized JSON of `Credential`

### Source 3: age-encrypted file

For headless contexts without a keychain (CI runners, Docker containers, headless SSH).

```bash
ONE_AGE_VAULT_PATH=/path/to/vault.age    # default: $HOME/.one/vault.age
ONE_AGE_PASSPHRASE=...                   # required (no prompt in v0.4)
```

Encrypted with [age](https://age-encryption.org/) in scrypt mode (passphrase). The age layer is wired only if `ONE_AGE_PASSPHRASE` or `ONE_AGE_VAULT_PATH` is defined (otherwise vault = env + keyring only).

### Chaining

Implemented in `adapters/vault/chain.go`:

```go
vlt := vault.NewChain(
    vault.NewEnvVar(),            // 1. env vars first
    vault.NewKeyring(clock),      // 2. keychain next
    vault.NewAge(path, passphrase), // 3. age fallback
)
```

The first one to respond wins. A `Fetch` that returns `ErrNotAuthenticated` passes to the next source. Any other error propagates.

## Token refresh

**Lazy** refresh, triggered at the moment of use, not in the background.

```go
func (uc *ExecuteAction) resolveCredentials(...) (Credential, error) {
    cred, err := uc.vault.Fetch(ctx, ref)
    if err != nil { return cred, err }

    if cred.NeedsRefresh(uc.clock.Now()) {
        provider := uc.authProviders[cred.Provider]
        refreshed, err := provider.Refresh(ctx, cred)
        if err != nil {
            return cred, core.ErrReAuthRequired{Service: cred.Service}
        }
        uc.vault.Store(ctx, ref, refreshed)
        return refreshed, nil
    }
    return cred, nil
}
```

### Refresh race condition

Two concurrent invocations may attempt to refresh at the same time. Some services rotate the refresh token, so the first to succeed revokes the other.

Solution: **file lock** in `~/.one/locks/<service>:<account>.lock`. The first instance acquires the lock and refreshes. Others wait, then re-read the vault and use the new token.

Acquisition timeout: 10s. Beyond that, error "concurrent refresh timeout".

### Refresh with rotation

For services that rotate the refresh token (GitHub, Google): the new refresh replaces the old one. The vault is written **before** the API request uses the new access token.

If the vault write fails (keychain unreachable, etc.): the refreshed credential is not returned; `ErrReAuthRequired` bubbles up. The old credential remains intact in the vault.

### Refresh fails

If the refresh fails (refresh token revoked, permanently expired): returns `ErrReAuthRequired`. The CLI layer maps to exit code 2 with a hint:

```
Error: re-authentication required for service 'github'
Hint: run `one login github --account work` to re-authenticate.
```

## Auth in headless contexts

CI, Docker containers, SSH sessions: no browser, no keychain always available.

### Mechanism 1: pre-populated vault

The developer copies the encrypted `vault.age` file from their machine to the CI runner. Provides the passphrase via a secret env var.

```yaml
# .github/workflows/agent.yml
env:
  ONE_AGE_VAULT_PATH: ${{ runner.temp }}/vault.age
  ONE_AGE_PASSPHRASE: ${{ secrets.ONE_AGE_PASSPHRASE }}

steps:
  - name: Download vault
    run: aws s3 cp s3://secrets/vault.age $ONE_AGE_VAULT_PATH
  - name: Run agent
    run: ./run-agent.sh
```

Good for controlled deployments, less so for large numbers of developers (process overhead).

### Mechanism 2: service account files

Deferred to v0.5.

### Mechanism 3: env var injection

```bash
ONE_CREDS_GITHUB_DEFAULT='{"access_token":"...","provider":"pat","service":"github","account":"default"}'
```

Total vault override. Use only in CI.

### Mechanism 4: device flow

For humans on headless terminals who have access to a browser elsewhere (phone). Selected via the provider, not a dedicated flag:

```bash
one login github --provider oauth2_device
```

Displays a code and a URL, the user validates on their phone.

## The `client_id`, a policy question

OAuth requires a `client_id` registered with each service. Two possible options:

### Option A: official One CLI client_ids

The binary ships with a hardcoded `client_id` per service. The OAuth app is called "One CLI". Simple for the user, but:

- You become responsible for rate limits
- You become responsible for the terms of use at each service
- You must maintain the registered app at each service

### Option B: BYOC (Bring Your Own client_id)

The user registers their own app. The `service.yaml` documents how. The user sets an env var (`ONE_GITHUB_CLIENT_ID`) or passes `--client-id`.

More friction, but no dependency on you.

### Recommended hybrid

For v0:

- **Official apps** for: GitHub, Notion, Linear, Slack (major services, low risk)
- **BYOC required** for: Google, Microsoft (long verification processes, friction to publish an app)
- **Not applicable** for: Stripe, OpenAI, AWS (no OAuth)

Document on the site:

- List of official apps
- Disclaimer "your data never transits through One CLI servers"
- How to revoke if needed
- How to migrate to BYOC

## Security

### The `Secret` type

Every token, refresh token, secret, password is typed `core.Secret`. This type implements `String()`, `GoString()`, `MarshalJSON()` to return `[REDACTED]`. The value is only revealed via `Reveal()`, callable only at the injection point.

Continuous test: a security test that injects a token with a recognizable value, captures all outputs (logs, stderr, audit), greps for the value. If found → CI fails.

### Auth audit log

Deferred to v0.5. `one trace` is wired on the CLI side but returns "not implemented".

### Anomaly detection

Deferred to v0.5.

### Credential disclosure

`one rotate <service> <account>` re-runs the login flow and overwrites the credential in the vault. Revocation of the old token via the OAuth endpoint is deferred to v0.5.

## Command summary

```bash
one login <service>                       # default provider: pat
one login <service> --provider <kind>     # pat, api_key, oauth2_user, oauth2_device, oauth2_client, aws_keys, certificate
one login <service> --as <alias>          # alias (default: default)

one logout <service> [--as <alias>]       # removes an account from the vault

one accounts <service>                    # lists accounts for a service

one rotate <service> <account>            # re-run login flow
one refresh <service> <account>           # force refresh (ignores margin)

one vault status                          # JSON: accounts by service in scope
one vault export                          # JSON plaintext on stdout (pipe to `age -p`)
one vault import                          # restore from JSON on stdin
```

No `--device` / `--client-id` / `--all` / global `one accounts` / `one vault export --to` in v0.4.

## Anti-patterns

### Storing credentials in `.onerc.yaml`

```yaml
# NEVER
services:
  github:
    token: "ghp_xxxxx"           # the scope file is public/committable
```

The scope file **never** contains credentials. Rejected by the validator if detected.

### Hardcoding client_id in service.yaml

```yaml
# NEVER
client_id: "ABCDEF1234567890"
```

Always `{env.ONE_<SERVICE>_CLIENT_ID}`. Allows users to use their own app and lets the project change the official client_id without repushing the catalog.

### Logging tokens

```go
// NEVER
log.Info("token", token.Reveal())
```

If you need the token for debugging, use `token.String()` which returns `[REDACTED]`.

### Reusing an expired access token

The runtime checks `NeedsRefresh` before each action. If you build custom code that uses a `Credential` directly, perform the check yourself.

### Saving the refresh token in custom logs

```ts
// BAD in a handler
host.log.info('Got refresh', { token: refresh_token });
```

`host.log` on the handler side is not automatically redacted. Never log anything that comes from `host.creds.get`.

---

*To propose a new auth provider type, open an RFC in `one-cli/rfcs`. A new type implies a new adapter, so careful validation of security and cross-platform compatibility is required.*
