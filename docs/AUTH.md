# AUTH.md

> Référence complète de l'authentification et de la gestion des credentials dans One CLI. Pour le format de déclaration auth dans un service, voir [CATALOG.md](./CATALOG.md). Pour la sécurité globale, voir [SECURITY.md](./SECURITY.md).

## Vue d'ensemble

L'auth dans One CLI ne signifie pas "une seule chose". Six modèles d'authentification sont supportés, parce qu'ils existent tous dans la nature et qu'aucun ne peut être substitué proprement par un autre.

| Schéma | Exemples | Renouvellement | Setup utilisateur |
|---|---|---|---|
| OAuth 2.0 user-flow | Notion, Linear, Google, Slack | refresh token | callback browser |
| OAuth 2.0 device flow | GitHub, Microsoft, Google (CLI) | refresh token | code + URL manuel |
| OAuth 2.0 client credentials | Twitch app, Reddit script | re-fetch | client_id + secret |
| API key statique | Stripe, OpenAI, Anthropic, Resend | jamais | copier-coller |
| Personal Access Token | GitHub, GitLab, Notion (legacy) | jamais ou manuel | génération web + copier |
| AWS-style signature | AWS, Cloudflare R2, MinIO | jamais | access + secret + region |
| Mutual TLS / certificat | services privés, registries | jamais | fichier de clé |

## Le piège classique : tout OAuth

Beaucoup de systèmes tentent de tout faire passer par OAuth. Mauvaise idée. Stripe **ne veut pas** d'OAuth pour les outils internes, leur DX c'est "copie ta clé". GitHub a un OAuth mais les devs préfèrent souvent un PAT scopé.

**Le système doit suivre la nature de chaque service**, pas la forcer dans un modèle unique.

## Le modèle générique : `auth.providers`

Un service déclare **un ou plusieurs providers d'auth** dans son `service.yaml`. L'utilisateur choisit au login.

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

Au login :

```
$ one login github
GitHub supports two authentication methods:

  1. OAuth (recommended) - opens browser, no copying needed
  2. Personal Access Token - paste an existing token

Choose [1]: _
```

Avec `--provider` pour scripter : `one login github --provider pat`.

## Les types de providers

### `oauth2_user`

Flow OAuth 2.0 standard avec PKCE + local server pour le callback.

```yaml
oauth:
  type: oauth2_user
  authorize_url: https://api.notion.com/v1/oauth/authorize
  token_url: https://api.notion.com/v1/oauth/token
  client_id: "{env.ONE_NOTION_CLIENT_ID}"
  scopes: [read, write]              # optionnel, dépend du service
  pkce: true                          # recommandé
  callback:
    mode: local_server
    path: /callback
    port: ephemeral                  # ou explicit: 54287
  refresh:
    supported: true
    rotation: true                   # si le service rotate le refresh token
  injection:
    header: Authorization
    format: "Bearer {access_token}"
    inject: auto                     # le binaire injecte, le handler n'y touche pas
  validate:
    method: GET
    url: "{api.base_url}/me"
    expect_status: 200
```

#### Flow en détail

1. **Resolve config.** Charge `service.yaml`, sélectionne le provider, lit `client_id`. Si manquant : erreur claire.

2. **Generate PKCE.** `code_verifier` (43-128 chars random) et `code_challenge` (SHA256 du verifier, base64url).

3. **Bind local server.** `net.Listen("tcp", "127.0.0.1:0")` → port éphémère. Sert un handler unique sur `/callback`.

4. **Build authorize URL.** Avec `state` (CSRF protection), `code_challenge`, `redirect_uri`, scopes.

5. **Open browser.** Via `open` (macOS), `xdg-open` (Linux), `start` (Windows). Fallback : afficher l'URL.

6. **Wait callback.** Timeout 5 minutes.

7. **Handle callback.** Vérifie `state`, extrait `code`. POST sur `token_url` avec `code` + `code_verifier`. Reçoit `access_token`, `refresh_token`, `expires_in`, `scope`.

8. **Render success page.** HTML minimaliste "Login successful, you can close this tab".

9. **Ask alias.** Terminal prompt "Save this account as [default]:".

10. **Validate.** Avant de stocker, appel `validate` (genre `GET /me`). Si 401, erreur claire.

11. **Store.** `vault.Store({service, alias}, credential)`.

```
$ one login notion
Opening browser for Notion authentication...
Waiting for callback on http://127.0.0.1:54287/callback

✓ Authenticated as ely@kaampus.fr
Save this account as [default]: kaampus
✓ Stored notion:kaampus in keychain
```

### `oauth2_device`

Pour les contextes sans browser (SSH headless, terminaux distants). RFC 8628.

```yaml
oauth:
  type: oauth2_device
  device_authorization_url: https://github.com/login/device/code
  token_url: https://github.com/login/oauth/access_token
  client_id: "{env.ONE_GITHUB_CLIENT_ID}"
  scopes: [repo, read:org]
```

Flow :

```
$ one login github --device
To authenticate, visit:
  https://github.com/login/device

And enter the code:
  ABCD-1234

Waiting for authorization... (5 minutes)
```

Le binaire poll le `token_url` toutes les 5s jusqu'à approbation ou timeout.

### `oauth2_client`

Client credentials, machine-to-machine. Pas d'utilisateur, juste une app.

```yaml
m2m:
  type: oauth2_client
  token_url: https://api.example.com/oauth/token
  scopes: [api.read, api.write]
  injection:
    header: Authorization
    format: "Bearer {access_token}"
```

L'utilisateur fournit `client_id` et `client_secret` une fois, le binaire les utilise pour obtenir un access token. Renouvellement automatique.

### `token_paste`

L'utilisateur copie-colle un token statique.

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

Flow :

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

L'input est masqué (no-echo, comme `sudo` password).

### `api_key`

Variante de `token_paste` pour les API keys statiques (Stripe, OpenAI, etc.).

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

Identique à `token_paste` côté UX, mais sémantiquement différent (pas un PAT user-scoped, c'est une clé d'app).

### `aws_keys`

Access key ID + secret + session token optionnel + region.

```yaml
default:
  type: aws_keys
  prompts:
    - { name: access_key_id, label: "AWS Access Key ID", secret: false }
    - { name: secret_access_key, label: "AWS Secret Access Key", secret: true }
    - { name: session_token, label: "Session Token (optional)", secret: true, optional: true }
    - { name: region, label: "AWS Region", default: us-east-1 }
```

Flow :

```
$ one login aws
AWS Access Key ID: AKIA...
AWS Secret Access Key: ●●●●●●●●●●
Session Token (optional, press Enter to skip):
AWS Region [us-east-1]: eu-west-1

Validating...
✓ Authenticated as arn:aws:iam::123456789:user/ely

Save this account as [default]:
✓ Stored aws:default in keychain
```

Validation via STS `GetCallerIdentity`. Region stockée dans `Credential.Extras.region`.

### `certificate`

Path vers une clé et un cert (mutual TLS).

```yaml
mtls:
  type: certificate
  prompts:
    - { name: cert_path, label: "Path to client certificate (PEM)" }
    - { name: key_path, label: "Path to client key (PEM)" }
```

Le binaire lit le contenu des fichiers et le stocke dans le vault (pas le path, pour éviter le déplacement). Utilisé via host TLS au moment de l'appel HTTP.

## Le type `Credential`

```go
// core/credential.go
type Credential struct {
    Service      ServiceID
    Account      AccountAlias              // "work", "perso", "default"
    Provider     string                    // "oauth", "pat", "api_key", etc.
    AccessToken  Secret                    // toujours présent (sauf certificate)
    RefreshToken Secret                    // optionnel
    ExpiresAt    *time.Time                // nil = pas d'expiration
    Scopes       []string                  // ce qui a été accordé (peut différer du demandé)
    Extras       map[string]Secret         // region AWS, session_token, etc.
    CreatedAt    time.Time
    LastUsedAt   time.Time
}

func (c Credential) NeedsRefresh(now time.Time) bool {
    if c.ExpiresAt == nil { return false }
    return c.ExpiresAt.Sub(now) < 60*time.Second
}
```

Le type `Secret` masque la valeur dans tous les logs/erreurs. Pour révéler : `secret.Reveal()`. À n'appeler qu'au moment d'injecter dans un header HTTP.

## Multi-comptes

Un service peut avoir N accounts. Le vault est indexé par `(service, alias)`.

### Créer un nouveau compte

```bash
$ one login github
Save as [default]: work
✓ Stored github:work

$ one login github
Save as [default]: perso
✓ Stored github:perso
```

### Lister les comptes d'un service

```bash
$ one accounts github
work    elydelva@protonmail.com     authenticated   refresh in 1h2m
perso   ely.delvallee@gmail.com     authenticated   refresh in 23m
```

### Sélectionner un compte

Trois niveaux de précédence (du moins prioritaire au plus) :

1. **`defaults.<service>` dans `.onerc.yaml`** : default du projet
2. **`defaults.<service>` dans `.onerc.local.yaml`** : override perso
3. **`--as <alias>` ad-hoc** : pour cette commande

```bash
one --as perso github issues.list           # override one-shot
```

### Supprimer un compte

```bash
$ one logout github --account perso
✓ Removed github:perso from keychain
```

## Le vault

Stockage local sécurisé des credentials. Trois sources possibles, chaînées par priorité.

### Source 1 : variables d'environnement

```bash
ONE_CREDS_GITHUB_DEFAULT='{"access_token":"...","provider":"pat"}'
```

Override total du vault, utile en CI. Format JSON sérialisé de `Credential`.

### Source 2 : keychain natif de l'OS

Implémenté via [`zalando/go-keyring`](https://github.com/zalando/go-keyring) qui abstrait :

- **macOS** : Keychain via Security framework
- **Linux** : Secret Service via libsecret (GNOME Keyring, KWallet)
- **Windows** : Credential Manager via wincred

Structure dans le keychain :

- **Service name** (keychain field) : `one`
- **Account name** (keychain field) : `<service>:<account_alias>` (ex: `github:work`)
- **Password** : JSON sérialisé de `Credential`

### Source 3 : fichier chiffré age

Pour les contextes headless sans keychain (CI runners, conteneurs Docker, SSH headless).

```bash
ONE_VAULT_FILE=/path/to/vault.age
ONE_VAULT_PASSPHRASE=...               # ou prompt interactif
```

Le fichier est chiffré avec [age](https://age-encryption.org/), un format moderne et auditable. La passphrase est demandée au prompt ou via env var.

### Chaînage

Implémenté en `adapters/vault/chain.go` :

```go
vlt := vault.NewChain(
    vault.NewEnvVar(),            // 1. env vars d'abord
    vault.NewKeyring(clock),      // 2. keychain ensuite
    vault.NewAge(path, passphrase), // 3. fallback age
)
```

Le premier qui répond gagne. Un `Fetch` qui retourne `ErrNotAuthenticated` fait passer à la source suivante. Toute autre erreur propage.

## Refresh des tokens

Refresh **lazy**, déclenché au moment de l'utilisation, pas en background.

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

### Race sur le refresh

Deux invocations concurrentes peuvent tenter de refresh en même temps. Certains services rotent le refresh token, donc le premier qui réussit révoque l'autre.

Solution : **file lock** dans `~/.one/locks/<service>:<account>.lock`. Première instance acquiert le lock et refresh. Les autres attendent, puis relisent le vault et utilisent le nouveau token.

Timeout d'acquisition : 10s. Au-delà, erreur "concurrent refresh timeout".

### Refresh avec rotation

Pour les services qui rotent le refresh token (GitHub, Google) : le nouveau refresh remplace l'ancien. Le vault est écrit **avant** que la requête API n'utilise le nouvel access token.

Si l'écriture vault fail (genre keychain unreachable), on rollback (re-store l'ancien). Évite "j'ai un access valide mais j'ai perdu le refresh".

### Refresh échoue

Si le refresh fail (refresh token révoqué, expirée définitivement) : retour de l'erreur `ErrReAuthRequired`. La couche CLI mappe sur exit code 2 avec hint :

```
Error: re-authentication required for service 'github'
Hint: run `one login github --account work` to re-authenticate.
```

## Auth dans les contextes headless

CI, conteneurs Docker, sessions SSH : pas de browser, pas toujours de keychain.

### Mécanisme 1 : pre-populated vault

Le développeur copie le fichier `vault.age` chiffré depuis sa machine vers le runner CI. Fournit la passphrase via secret env var.

```yaml
# .github/workflows/agent.yml
env:
  ONE_VAULT_FILE: ${{ runner.temp }}/vault.age
  ONE_VAULT_PASSPHRASE: ${{ secrets.ONE_VAULT_PASSPHRASE }}

steps:
  - name: Download vault
    run: aws s3 cp s3://secrets/vault.age $ONE_VAULT_FILE
  - name: Run agent
    run: ./run-agent.sh
```

Bien pour les déploiements contrôlés, moins bien pour le grand nombre de devs (lourdeur du process).

### Mécanisme 2 : service account files (pour AWS, Google)

```bash
one login google --service-account /path/to/sa.json
```

Pour les services qui supportent (Google, AWS via IAM roles), un fichier de credentials machine est importé directement.

### Mécanisme 3 : env var injection

```bash
ONE_CREDS_GITHUB_DEFAULT='{"access_token":"...","provider":"pat","service":"github","account":"default"}'
```

Override total du vault. À utiliser uniquement en CI.

### Mécanisme 4 : device flow

Pour les humains sur des terminaux headless mais qui ont accès à un browser ailleurs (téléphone).

```bash
one login github --device
```

Affiche un code et une URL, l'utilisateur valide sur son téléphone.

## Le `client_id`, question politique

OAuth nécessite un `client_id` enregistré chez chaque service. Deux options possibles :

### Option A : client_id officiels One CLI

Le binaire ships avec un `client_id` hardcodé par service. L'app OAuth s'appelle "One CLI". Simple pour l'utilisateur, mais :

- Tu deviens responsable des limites de rate
- Tu deviens responsable des conditions d'usage chez chaque service
- Tu dois maintenir l'app enregistrée chez chaque service

### Option B : BYOC (Bring Your Own client_id)

L'utilisateur enregistre sa propre app. Le `service.yaml` documente comment. L'utilisateur set un env var (`ONE_GITHUB_CLIENT_ID`) ou passe `--client-id`.

Plus de friction, mais aucune dépendance à toi.

### Hybride recommandé

Pour la v0 :

- **Apps officielles** pour : GitHub, Notion, Linear, Slack (services majeurs, peu de risque)
- **BYOC obligatoire** pour : Google, Microsoft (vérifications longues, friction à publier une app)
- **Pas concerné** pour : Stripe, OpenAI, AWS (pas d'OAuth)

Documenter sur le site :

- Liste des apps officielles
- Disclaimer "vos données ne transitent jamais par les serveurs One CLI"
- Comment révoquer en cas de besoin
- Comment migrer vers BYOC

## Sécurité

### Le type `Secret`

Tout token, refresh token, secret, password est typé `core.Secret`. Ce type implémente `String()`, `GoString()`, `MarshalJSON()` pour retourner `[REDACTED]`. Ne révèle la valeur que via `Reveal()`, appelable uniquement au point d'injection.

Test continu : un test de sécurité qui injecte un token de valeur reconnaissable, capture toutes les sorties (logs, stderr, audit), grep pour la valeur. Si trouvée → CI fail.

### Audit log d'auth

Chaque login, refresh, logout est loggé dans `~/.one/audit.log` :

```
2026-05-20T14:32:11Z LOGIN github:work via=oauth2_user expires=2026-05-20T15:32:11Z
2026-05-20T15:31:45Z REFRESH github:work via=oauth2_user rotated=true
2026-05-20T16:00:00Z LOGOUT github:work
```

Visible via `one trace --auth`. Sans les tokens, juste les métadonnées.

### Détection d'anomalies (v1+)

Optionnel et opt-in : alertes locales si :

- Login d'un service depuis un device différent de l'historique
- Refresh dans un intervalle de temps anormal
- Échec de refresh répété

Pas envoyé externe, juste un warning dans le terminal.

### Disclosure de credentials

En cas de fuite suspectée (token loggé par erreur, vault.age partagé) : `one rotate <service> <account>` qui :

1. Force un nouveau login pour ce compte
2. Révoque l'ancien token via l'endpoint OAuth (si supporté par le service)
3. Met à jour le vault

## Commandes récapitulatives

```bash
one login <service>                       # login avec default_provider
one login <service> --provider pat        # login avec un provider spécifique
one login <service> --as <alias>          # créer un nouvel alias
one login <service> --device              # device flow (headless)
one login <service> --client-id <id>      # BYOC

one logout <service> [--account <alias>]  # déconnecter, supprime du vault
one logout --all                          # déconnecte tout (debug, attention)

one accounts                              # liste tous les comptes
one accounts <service>                    # liste les comptes d'un service

one rotate <service> <account>            # force re-login (cas de fuite)
one refresh <service> <account>           # force refresh manuellement

one vault export --to vault.age           # export le vault (pour CI / backup)
one vault import vault.age                # import (sur une nouvelle machine)
one vault status                          # affiche quelle source du vault est active
```

## Anti-patterns

### Stocker des credentials dans `.onerc.yaml`

```yaml
# JAMAIS
services:
  github:
    token: "ghp_xxxxx"           # le scope file est public/committable
```

Le scope file ne contient **jamais** de credentials. Refusé par le validateur si détecté.

### Hardcoder client_id dans service.yaml

```yaml
# JAMAIS
client_id: "ABCDEF1234567890"
```

Toujours `{env.ONE_<SERVICE>_CLIENT_ID}`. Permet aux utilisateurs d'utiliser leur propre app et au projet de changer le client_id officiel sans repush du catalogue.

### Logger les tokens

```go
// JAMAIS
log.Info("token", token.Reveal())
```

Si tu as besoin du token pour debug, utilise `token.String()` qui retourne `[REDACTED]`.

### Réutiliser un access token expiré

Le runtime check `NeedsRefresh` avant chaque action. Si tu construis du code custom qui utilise une `Credential` directement, fais le check toi-même.

### Sauvegarder le refresh token dans des logs custom

```ts
// MAUVAIS dans un handler
host.log.info('Got refresh', { token: refresh_token });
```

Les host.log côté handler ne sont pas redactés automatiquement. Ne log jamais ce qui vient de `host.creds.get`.

---

*Pour proposer un nouveau type de provider d'auth, ouvrir un RFC dans `one-cli/rfcs`. Un nouveau type implique un nouvel adapter, donc validation soigneuse de la sécurité et de la compatibilité cross-platform.*
