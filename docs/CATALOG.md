# CATALOG.md

> Référence complète pour ajouter ou modifier un service dans le catalogue One CLI. Pour contributeurs externes au repo `one-cli/catalog`. Pour le contrat WASM, voir [HANDLERS.md](./HANDLERS.md).

## Vue d'ensemble

Le catalogue est un repo Git public (`one-cli/catalog`) qui contient la définition de chaque service supporté. Sa CI publie un index JSON statique sur CDN, qu'un binaire `one` fetch pour résoudre les services.

**Ajouter un service = ouvrir une PR.** Le repo a une CI stricte qui valide le format, lance les tests, vérifie les contraintes de sécurité. Une fois mergée, la PR déclenche la publication automatique sur l'index.

## Structure d'un service

```
services/
└── notion/
    ├── service.yaml             # métadonnées + déclaration auth + permissions
    ├── SKILL.md                 # markdown pour `one info notion`
    ├── actions/
    │   ├── pages.read.yaml
    │   ├── pages.read.md        # optionnel, sinon généré depuis YAML
    │   ├── pages.create.yaml
    │   ├── pages.update.yaml
    │   ├── databases.query.yaml
    │   ├── blocks.append.yaml
    │   └── search.yaml
    ├── guides/
    │   ├── initial-setup.md
    │   └── share-page.md
    └── handlers/                # uniquement si WASM requis
        ├── main.ts              # source
        ├── main.wasm            # compilé par CI, pas commité
        ├── package.json
        └── tests/
            └── main.test.ts
```

**Convention** :

- Le **nom du dossier** est l'identifiant du service utilisé dans les commandes (`one notion ...`).
- Les **noms d'actions** suivent `<resource>.<verb>` (ex: `pages.create`, pas `createPage` ou `create_page`).
- Les **noms de guides** sont des slugs kebab-case (`share-page`, `iam-setup`).

## Le fichier `service.yaml`

C'est le manifeste du service. Référence complète des champs.

### Squelette minimal

```yaml
version: 1
name: notion
display_name: Notion
description: Workspace pages, databases, blocks
homepage: https://notion.so
docs_url: https://developers.notion.com
license: MIT
maintainers:
  - github: elydelva

api:
  base_url: https://api.notion.com/v1
  version: "2025-09-03"            # version d'API à passer
  headers:
    Notion-Version: "{api.version}"

auth:
  default_provider: oauth
  providers:
    oauth:
      type: oauth2_user
      authorize_url: https://api.notion.com/v1/oauth/authorize
      token_url: https://api.notion.com/v1/oauth/token
      client_id: "{env.ONE_NOTION_CLIENT_ID}"
      pkce: true
      callback:
        mode: local_server
        path: /callback
      injection:
        header: Authorization
        format: "Bearer {access_token}"
      validate:
        method: GET
        url: "{api.base_url}/users/me"
        expect_status: 200

permissions:
  pages.read:
    kind: query
    description: Read page properties and content
  pages.write:
    kind: mutation
    description: Create and update pages
  blocks.read:
    kind: query
  blocks.write:
    kind: mutation
  databases.read:
    kind: query
  search:
    kind: query
```

### Sections en détail

#### `version` (obligatoire)

Version du format du service.yaml. Toujours `1` actuellement. Permet l'évolution du format dans le futur sans casser les services existants.

#### Identité

| Champ | Type | Obligatoire | Description |
|---|---|:---:|---|
| `name` | string | oui | identifiant du service, doit matcher le nom du dossier |
| `display_name` | string | oui | nom humain pour l'affichage |
| `description` | string | oui | 1-2 phrases, affiché dans `one info` |
| `homepage` | URL | oui | site officiel du service |
| `docs_url` | URL | oui | doc API officielle (utile pour les agents) |
| `license` | string | non | license du service.yaml et handlers, défaut MIT |
| `maintainers` | array | non | liste des contributeurs principaux |
| `tags` | array | non | catégories (productivity, payment, dev, etc.) |

#### `api`

Configuration HTTP commune à toutes les actions du service.

```yaml
api:
  base_url: https://api.notion.com/v1
  version: "2025-09-03"
  headers:
    Notion-Version: "{api.version}"
    User-Agent: "one-cli/{version}"
  timeout_ms: 30000
  rate_limit:
    requests_per_second: 3
    burst: 10
```

- `base_url` est préfixé devant chaque `request.url` relative des actions.
- `version` est référencée via `{api.version}` dans les headers ou les paths.
- `headers` sont injectés sur chaque requête.
- `timeout_ms` est le timeout par défaut, surchargeable par action.
- `rate_limit` est purement informatif au début (utilisé pour les hints), pas appliqué.

#### `auth`

Liste des providers d'auth supportés. Voir [AUTH.md](./AUTH.md) pour la sémantique de chaque type.

```yaml
auth:
  default_provider: oauth
  providers:
    oauth:
      type: oauth2_user
      # ...
    pat:
      type: token_paste
      label: Personal Access Token
      help_url: https://github.com/settings/tokens?type=beta
      validate:
        method: GET
        url: "{api.base_url}/user"
        expect_status: 200
      injection:
        header: Authorization
        format: "Bearer {token}"
```

Au login, l'utilisateur choisit parmi les providers. Le `default_provider` est sélectionné si l'utilisateur ne précise pas avec `--provider`.

#### `permissions`

La liste exhaustive des permissions exposées par le service. **C'est l'unité de granularité du scope file.**

```yaml
permissions:
  pages.read:
    kind: query              # query | mutation
    description: Read page properties and content
  pages.write:
    kind: mutation
    description: Create and update pages
    side_effects: write      # write | read (défaut: déduit de kind)
  pages.archive:
    kind: mutation
    side_effects: destructive
```

- `kind` (`query` ou `mutation`) classe l'action. Affichée dans `one capabilities`. Important pour les agents qui veulent filtrer.
- `side_effects` précise pour les mutations : `write` (création/update), `destructive` (delete, archive). Permet aux scope files de bloquer les opérations destructives même si les autres mutations sont autorisées.

**Convention de nommage** :

- Toujours en minuscules.
- Path dot-separated : `resource.verb` (ex: `pages.read`).
- Verbes standards : `read`, `write`, `delete`, `archive`, `query`, `list`, `search`, `subscribe`.

#### `credentials`

Déclare les credentials que les actions/handlers peuvent demander via `host.creds.get`.

```yaml
credentials:
  access_token:
    type: secret
    source: oauth.access_token
    description: OAuth access token
  refresh_token:
    type: secret
    source: oauth.refresh_token
    optional: true
  region:
    type: string             # non-secret
    source: config.region
    description: AWS region
```

- `type` : `secret` (redacté dans les logs) ou `string` (config visible).
- `source` : d'où vient la valeur. Trois sources :
  - `oauth.<field>` : du flow OAuth (access_token, refresh_token, extras.*)
  - `config.<field>` : config additionnelle saisie par l'utilisateur au login
  - `static.<value>` : valeur fixe (rare)
- `optional` : si false (défaut), l'action échoue si la cred est manquante.

#### `required_setup`

Liste les guides d'install qui peuvent être requis pour utiliser le service.

```yaml
required_setup:
  - id: initial-setup
    description: OAuth connection and integration creation
    blocks: ["*"]
  - id: share-page
    description: Share specific pages with the integration
    blocks: ["pages.*", "blocks.*", "databases.*"]
    optional: false
    detection: |
      If any API call returns 404 with code "object_not_found",
      the integration likely lacks access to that resource.
    auto_detect_on_error: object_not_found
```

- `id` : matche un fichier `guides/<id>.md`.
- `blocks` : sur quelles permissions ce setup est requis (globs).
- `optional` : si true, le setup est suggéré mais pas obligatoire pour utiliser ces permissions.
- `detection` : description humaine de comment détecter qu'il faut faire ce setup.
- `auto_detect_on_error` : code d'erreur (matché contre `errors.<code>`) qui déclenche automatiquement la suggestion du guide.

## Le fichier `actions/<id>.yaml`

Une action = un fichier. Référence complète.

### Squelette d'une action déclarative (REST simple)

```yaml
id: pages.read
permission: pages.read
summary: Retrieve a page object by ID
docs_url: https://developers.notion.com/reference/retrieve-a-page

request:
  method: GET
  path: /pages/{page_id}

inputs:
  page_id:
    type: string
    required: true
    description: UUID of the page (with or without dashes)
    pattern: "^[0-9a-f]{8}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{12}$"
    location: path           # path | query | body | header
  filter_properties:
    type: array
    items: { type: string }
    required: false
    description: Subset of property IDs to return
    location: query

output:
  type: object
  passthrough: true          # renvoie le JSON tel quel

errors:
  404:
    code: not_found
    install_guide: share-page
    hint: "Page does not exist or integration lacks access."
  429:
    code: rate_limited
    retry: backoff
    max_attempts: 3
```

### Champs détaillés

#### `id` (obligatoire)

Identifiant de l'action. Doit matcher le nom du fichier (sans `.yaml`).

#### `permission` (obligatoire)

Référence à une permission déclarée dans `service.yaml > permissions`. Une action = une permission. Si l'action requiert plusieurs permissions logiques, c'est probablement qu'il faut deux actions.

#### `summary` et `docs_url`

Description humaine et lien vers la doc officielle de l'endpoint.

#### `request` (pour les actions déclaratives)

```yaml
request:
  method: GET                            # GET | POST | PUT | PATCH | DELETE | HEAD
  path: /pages/{page_id}                 # relatif à api.base_url
  body: $inputs                          # ou objet template
  headers:
    Content-Type: application/json       # surcharge les headers globaux
  query:
    page_size: "{inputs.limit}"
```

- `path` peut contenir des placeholders `{input_name}` qui sont substitués.
- `body: $inputs` sérialise les inputs en JSON dans le body.
- `body` peut aussi être un objet template (ex: `{ data: "{inputs.payload}", meta: { source: "one" } }`).
- `headers` complète/surcharge les headers globaux de l'API.

#### `inputs`

Schéma typé des arguments de l'action. Validé avant l'invocation du runtime.

```yaml
inputs:
  page_id:
    type: string                         # string | integer | number | boolean | array | object | file_ref
    required: true                       # défaut: false
    description: ...
    pattern: "^[0-9a-f-]{36}$"           # regex pour string
    enum: [low, medium, high]            # valeurs possibles
    default: "medium"                    # valeur si non fournie
    min: 0                               # pour number/integer
    max: 100
    min_length: 1                        # pour string
    max_length: 255
    location: path                       # path | query | body | header

  properties:
    type: object
    required: true
    schema:                              # sous-schéma libre
      type: object
      properties:
        title: { type: array }
        Status: { type: object }

  body:
    type: file_ref                       # référence à un fichier local
    description: Path to upload
```

**Le type `file_ref`** est spécial : le user passe `--body @path/to/file.pdf`, le binaire lit le fichier et le passe au runtime comme `Uint8Array`.

#### `output`

Schéma de la sortie. Validé après l'invocation. Si non valide, c'est un bug du handler/runtime, pas une erreur user.

```yaml
output:
  type: object
  passthrough: true                      # accepte n'importe quel JSON
```

Ou plus strict :

```yaml
output:
  type: object
  schema:
    type: object
    required: [id, properties]
    properties:
      id: { type: string }
      properties: { type: object }
      url: { type: string, format: uri }
```

Ou un array (genre `search`) :

```yaml
output:
  type: array
  items:
    type: object
    schema: ...
```

#### `errors`

Mapping entre codes HTTP/erreurs natives du service et un code stable côté One CLI + un hint.

```yaml
errors:
  404:
    code: not_found
    install_guide: share-page            # ref à un guide
    hint: "Page does not exist or integration lacks access."
  401:
    code: not_authenticated
    hint: "Run `one login notion` to authenticate."
  429:
    code: rate_limited
    retry: backoff                       # auto-retry avec backoff exponentiel
    max_attempts: 3
    backoff_initial_ms: 1000
  validation_failed:                     # code d'erreur du service (genre Stripe)
    code: invalid_input
    hint: "Check the inputs match the schema."
```

Les `code:` sont les codes stables que l'agent voit dans l'output JSON. Doivent être documentés. Stables entre versions.

#### `side_effects`, `idempotency`, `dry_run`

Pour les mutations :

```yaml
side_effects: write                      # read | write | destructive
idempotency:
  supported: true                        # true si l'API supporte l'idempotence native
  key: header.Idempotency-Key            # où passer la clé
  required: false                        # si true, l'input idempotency_key est obligatoire

dry_run:
  supported: true
  behavior: |
    Validates the request without sending. Returns what would have been sent.
```

#### `handler` (pour les actions WASM)

```yaml
handler: ./handlers/main.wasm
handler_entry: pages_create
host_api_version: 1

calls:                                   # URLs allowlistées
  - method: POST
    url: "https://api.notion.com/v1/pages"
  - method: POST
    url_pattern: "^https://api\\.notion\\.com/v1/pages/[a-f0-9-]+$"
```

Voir [HANDLERS.md](./HANDLERS.md) pour le détail du contrat WASM.

#### Streaming

```yaml
streaming: true
```

Signale que l'action retourne un flux d'objets (NDJSON) plutôt qu'un seul objet. Utile pour les listes paginées agrégées. Le skill du service doit le documenter.

#### Pagination automatique (action déclarative)

```yaml
pagination:
  style: cursor                          # cursor | page | offset
  request_param: cursor                  # nom du param qu'on passe à la requête
  request_location: query
  response_token: next_cursor            # field dans la réponse
  response_has_more: has_more
  max_pages: 50                          # safety cap
```

Le runtime déclaratif gère la boucle automatiquement, agrège les résultats. L'agent appelle l'action sans se soucier de la pagination.

## Le fichier `SKILL.md` du service

Markdown destiné aux agents IA. Lu via `one info <service>`.

**Frontmatter obligatoire** :

```markdown
---
service: notion
version: 1.4.0
summary: Read and write pages, databases, and blocks in a Notion workspace
use_when:
  - "User wants to log, track, or document something in Notion"
  - "User asks to read or update a Notion page/database"
avoid_when:
  - "User wants a quick local note (use filesystem instead)"
---
```

**Structure recommandée** :

1. **Mental model** (3-5 phrases) : les concepts uniques à ce service.
2. **Required setup** : ce qui doit être fait avant la première utilisation.
3. **Typical workflow** : 2-3 patterns courants avec exemples de commandes.
4. **Gotchas** : 3-5 pièges spécifiques à ce service.
5. **Common chains** : recettes pour les usages composites.
6. **Permissions** : liste des perms typiques avec lien vers `one scope add`.

**Longueur cible** : 100-200 lignes. Pas plus, le skill doit rester scannable.

Exemple minimal :

```markdown
---
service: stripe
version: 3.0.2
summary: Read and write customers, subscriptions, charges, refunds via Stripe API
use_when:
  - "User wants to query Stripe data (customers, charges)"
  - "User wants to create or update billing entities"
---

# Stripe

## Mental model

Stripe organizes everything around `Customer`, `PaymentMethod`,
`Subscription`, `Charge`, and `Invoice`. Customer is central: most
mutations attach to a customer.

## Setup

Stripe uses API keys (no OAuth). Get one from
https://dashboard.stripe.com/apikeys. Test mode keys start with
`sk_test_`, live mode with `sk_live_`. Use different accounts via:

  one login stripe          → choose test or live key

## Typical workflows

**Lookup a customer:**
  one stripe customers.search --query "email:'user@example.com'"

**Create a customer:**
  one stripe customers.create --email user@example.com --name "Jane"

**List recent charges:**
  one stripe charges.list --limit 10

## Gotchas

- **Idempotency is critical**. Always pass `--idempotency_key=...`
  when creating customers, charges, or subscriptions, to avoid
  duplicate side effects on retry.
- **Test vs live mode**. Stripe data is fully isolated. A customer
  created in test mode is invisible from a live mode account.
- **Pagination**. Use `--starting_after=<id>` for cursor pagination.
  One CLI auto-paginates `*.list_all` variants.

## Permissions

- `customers.read`, `customers.write`
- `charges.read`, `charges.write`
- `subscriptions.read`, `subscriptions.write`
- `webhooks.read`, `webhooks.write`

Add via `one scope add stripe <permission>`.
```

## Les guides d'install (`guides/<id>.md`)

Markdown avec frontmatter pour les setups requérant un humain.

```markdown
---
id: share-page
title: Share a Notion page with your integration
estimated_time: 30s
requires_human: true
requires:
  - authenticated: true
  - capability: pages.read
verify:
  action: pages.read
  inputs:
    page_id: "${PROMPT:Paste the page ID after sharing}"
  expect_success: true
related_errors:
  - not_found
  - object_not_found
applies_to:
  permissions: ["pages.*", "blocks.*", "databases.*"]
open_url: https://notion.so
---

Notion's permission model is opt-in per page or database. After the
initial OAuth flow, the integration sees nothing until pages are
explicitly shared with it.

## What to do

1. Open the page or database in Notion
2. Click `•••` (top right) → `Connections`
3. Search for your integration name and click it
4. Confirm

## Verify

Run `one install notion share-page --verify` and paste a page ID to
check that the share worked.

## If it still fails

Check that the integration has the required capabilities at
https://www.notion.so/profile/integrations.
```

**Champs du frontmatter** :

| Champ | Type | Description |
|---|---|---|
| `id` | string | identifiant du guide, matche le nom de fichier |
| `title` | string | titre humain |
| `estimated_time` | string | "30s", "2m", "10m" |
| `requires_human` | bool | true si humain requis, false si automatisable |
| `requires` | object | pré-conditions (authenticated, capabilities) |
| `verify` | object | comment vérifier que le setup a fonctionné |
| `related_errors` | array | codes d'erreur que ce guide résout |
| `applies_to` | object | sur quelles permissions ce guide s'applique |
| `open_url` | URL | URL à ouvrir quand l'utilisateur tape `[o]` |
| `auto_install` | object | pour les guides non-humains, l'action à exécuter |

**Guides automatisables** :

```markdown
---
id: create-webhook
title: Create a Stripe webhook
requires_human: false
auto_install:
  action: webhooks.create
  inputs_from_prompt:
    url:
      prompt: "Webhook endpoint URL"
    events:
      prompt: "Events to subscribe to"
      default: "customer.created,customer.updated"
---
```

L'agent peut exécuter directement avec `one install stripe create-webhook --url X --events Y`.

## Process de contribution au catalogue

### 1. Fork le repo

```
git clone https://github.com/one-cli/catalog
```

### 2. Scaffold

```bash
one catalog scaffold my-service --lang ts
# crée services/my-service/{service.yaml, handlers/main.ts, package.json, tests/}
```

Ou manuellement, en copiant `services/_template/`.

### 3. Implémenter

- Remplir `service.yaml`
- Écrire les actions YAML (et handler WASM si nécessaire)
- Écrire `SKILL.md`
- Écrire les guides nécessaires

### 4. Tester localement

```bash
# Lint
one catalog lint my-service

# Build WASM si applicable
cd services/my-service && bun run build

# Tests handler
bun test

# Integration test
one catalog test my-service          # invoque le binaire one en mode local
```

### 5. Ouvrir une PR

Template de PR pré-rempli. Checklist :

- [ ] `service.yaml` passe le lint
- [ ] Toutes les actions ont un schéma d'inputs valide
- [ ] SKILL.md respecte la structure recommandée
- [ ] Au moins un guide d'install (même `initial-setup`)
- [ ] Tests des handlers passent
- [ ] Allowlist URLs cohérente avec ce que le handler fait
- [ ] Permissions déclarées matchent les credentials utilisées

### 6. Review et merge

Les maintainers du catalog reviewent. Critères :

- **Sécurité** : pas de credentials hardcodées, allowlist URL strict, pas d'écritures destructives sans warning
- **Qualité du skill** : compréhensible par un agent qui ne connaît pas le service
- **Robustesse** : gestion des erreurs courantes, hints actionnables
- **Cohérence** : naming des permissions, structure des inputs

Une fois mergée, la CI publie automatiquement la nouvelle version sur l'index.

## Versioning des services

Chaque service a un `version` SemVer dans son `service.yaml` (en plus de `version: 1` du format).

```yaml
version: 1
name: notion
service_version: 1.4.0
```

Versions :

- **Patch** (1.4.0 → 1.4.1) : fix de bug dans un handler, clarification de doc, hint ajusté
- **Minor** (1.4.0 → 1.5.0) : nouvelle action ajoutée, nouvelle permission, nouveau provider d'auth
- **Major** (1.4.0 → 2.0.0) : breaking change (action renommée, signature d'input changée, code d'erreur renommé)

Les utilisateurs pinnent la version dans `.onerc.lock`. Une major bump requiert une action manuelle (`one lock --update notion`).

## Sécurité et review

### Ce qui est interdit dans un service.yaml

- URLs `http://` (sauf localhost pour les tests)
- Credentials hardcodées (`access_token: "sk_..."`)
- Headers contenant des références à `${env.SECRET_*}` (utiliser le mécanisme de credentials)
- Permissions sans `description`
- Actions sans `errors` mapping (au minimum pour 401/403/404/429)

### Ce qui est vérifié automatiquement

- Schéma JSON Schema sur le YAML
- Chaque action référence une permission déclarée
- Chaque guide référencé dans `required_setup` existe
- Chaque code d'erreur référencé dans les guides existe dans une action
- Pour les WASM : URLs hit ⊆ `calls:` déclarées
- Pour les WASM : credentials lues ⊆ `credentials:` déclarées
- Pour les WASM : codes d'erreur retournés ⊆ `errors:` déclarés

### Reviewers

Pour les services à fort impact (auth complexe, secteurs sensibles, gros catalogues) : double review. Pour les services simples : single review suffit.

## Exemples complets

Voir le repo `one-cli/catalog/examples/` pour :

- `simple-rest/` : service trivial avec API key + GET/POST
- `oauth-paginated/` : OAuth + pagination cursor
- `wasm-graphql/` : GraphQL via WASM
- `wasm-sigv4/` : signature de requête AWS via WASM
- `multi-account/` : service avec multi-accounts complexes

Chaque exemple est commenté pour servir de référence.

---

*Pour proposer une évolution du format `service.yaml`, ouvrir un RFC dans `one-cli/rfcs` plutôt qu'une PR directe sur le catalog.*
