# HANDLERS.md

> Référence complète pour écrire des handlers WASM utilisés par les services du catalogue One CLI. Pour le format `service.yaml`, voir [CATALOG.md](./CATALOG.md). Pour le threat model, voir [SECURITY.md](./SECURITY.md).

## Pourquoi des handlers WASM

Le YAML déclaratif couvre 95% des cas (REST simple, auth Bearer, passthrough du body). Les handlers WASM existent pour le 5% restant : signature de requête, plusieurs appels chaînés, GraphQL avec interpolation typée, transformation de réponse non triviale, pagination automatique complexe.

**Décider entre YAML et WASM** :

| Cas | YAML pur | WASM |
|---|:---:|:---:|
| Un endpoint, body passthrough, headers fixes | ✓ | |
| Auth Bearer, API key dans un header | ✓ | |
| Pagination cursor simple (param + token) | ✓ | |
| Headers calculés (hash, signature, JWT) | | ✓ |
| ≥ 2 requêtes HTTP par action | | ✓ |
| Transformation de la réponse au-delà du passthrough | | ✓ |
| GraphQL avec interpolation des inputs dans la query | | ✓ |
| Logique conditionnelle (if X then call Y else Z) | | ✓ |
| Rollback en cas d'échec partiel | | ✓ |

En cas de doute, **commencer par le YAML**. Migrer vers WASM uniquement si l'option déclarative n'est pas atteignable.

## Modèle d'isolation

Un handler tourne dans un module WASM via [wazero](https://wazero.io/). WASI minimal, **rien n'est exposé par défaut** :

- Pas de filesystem
- Pas d'env vars
- Pas de réseau direct
- Pas d'horloge directe
- Pas de random direct
- Pas d'exec de process
- Pas de stdin/stdout sauvages

Le handler ne peut interagir avec le monde que via les **host functions** exposées explicitement par le binaire One CLI.

Cette isolation est ce qui rend le catalogue *communautaire-safe* : un handler `stripe.wasm` ne peut pas lire `~/.ssh/`, exfiltrer le vault, ou faire un appel HTTP arbitraire vers un serveur externe.

## Le flow d'invocation

```
┌──────────────┐   1. invoke handler          ┌─────────────────┐
│   One CLI    │ ──────────────────────────►  │  WASM handler   │
│   (host)     │   inputs (JSON)              │  (sandboxed)    │
│              │                              │                 │
│              │  ◄── 2. host functions ──── │                 │
│              │      creds.get               │                 │
│              │      http.request            │                 │
│              │      crypto.*                │                 │
│              │      time.*                  │                 │
│              │      log.*                   │                 │
│              │      fail                    │                 │
│              │                              │                 │
│              │  ──── 3. responses ───────► │                 │
│              │                              │                 │
│              │  ◄── 4. return ────────────  │                 │
│              │      output (JSON)           │                 │
│              │      or structured error     │                 │
└──────────────┘                              └─────────────────┘
```

## Le contrat I/O

### Ce que le handler reçoit (sur invocation)

```json
{
  "action": "pages.create",
  "inputs": {
    "parent": { "page_id": "abc-123" },
    "properties": { "title": [...] }
  },
  "config": {
    "api_version": "2025-09-03"
  },
  "context": {
    "account": "kaampus",
    "dry_run": false,
    "trace_id": "01HXYZ..."
  }
}
```

**Les credentials ne sont pas dans ce payload.** Le handler les demande via `host.creds.get`.

### Ce que le handler renvoie

Soit un objet JSON arbitraire (validé contre `output:` du YAML si défini) :

```json
{
  "id": "page_abc",
  "url": "https://notion.so/...",
  "created_at": "2026-05-20T..."
}
```

Soit termine via `host.fail.withCode(...)` (équivalent à un throw structuré).

## Les host functions

### `host.creds`

```ts
namespace host.creds {
  /**
   * Récupère une credential par son nom logique (déclaré dans service.yaml > credentials).
   * Échoue si la clé n'est pas déclarée.
   * La valeur retournée est marquée sensible et redactée dans les logs.
   */
  function get(key: string): string;
}
```

**Exemple** :

```ts
const token = host.creds.get('access_token');
// throws if not declared in service.yaml > credentials
```

### `host.http`

```ts
namespace host.http {
  function request(req: HttpRequest): HttpResponse;
}

interface HttpRequest {
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE' | 'HEAD';
  url: string;
  headers?: Record<string, string>;
  body?: string | Uint8Array;
  timeout_ms?: number;        // default 30000, max 120000
  follow_redirects?: boolean; // default true
  max_redirects?: number;     // default 5, max 10
}

interface HttpResponse {
  status: number;
  headers: Record<string, string>;
  body: Uint8Array;
}
```

**Allowlist obligatoire**. Avant l'invocation, le host parse `service.yaml > actions > calls:` et construit une liste de patterns d'URL autorisés. Si le handler appelle `host.http.request` avec une URL qui ne matche aucun pattern, l'host échoue avec `url_not_allowed`.

**Auto-injection des credentials standards**. Pour les schémas auth courants (Bearer, Basic), si le `service.yaml > auth.providers.X.injection` déclare `inject: auto`, l'host ajoute le header automatiquement. Le handler n'a pas besoin de toucher au token. Sécurité accrue.

Pour les schémas custom (SigV4, JWT, signing), `inject: manual` et le handler forge lui-même via `host.creds.get` + `host.crypto.*`.

**Audit log**. Chaque requête est loggée (méthode, host, path, status, durée). Pas le body. Visible via `one trace`.

### `host.crypto`

Crypto **primitive uniquement**, pas de high-level "encrypt(data, password)".

```ts
namespace host.crypto {
  function sha256(data: Uint8Array): Uint8Array;
  function sha512(data: Uint8Array): Uint8Array;
  function hmacSha256(key: Uint8Array, data: Uint8Array): Uint8Array;
  function hmacSha512(key: Uint8Array, data: Uint8Array): Uint8Array;
  function randomBytes(n: number): Uint8Array;
  function uuidV4(): string;
  function base64Encode(data: Uint8Array, urlSafe?: boolean): string;
  function base64Decode(s: string, urlSafe?: boolean): Uint8Array;
  function hexEncode(data: Uint8Array): string;
  function hexDecode(s: string): Uint8Array;
}
```

Suffisant pour tous les schémas d'auth modernes (SigV4, JWT signing, webhook verification). Le reste de la logique se fait en pure code dans le handler.

### `host.time`

```ts
namespace host.time {
  function now(): number;          // unix ms
  function sleep(ms: number): void; // bloquant, capé à 30s par appel, 60s cumulés
}
```

Pas d'accès à un clock monotonic. Pas de timezone. Si le handler a besoin d'une string ISO 8601, il la construit lui-même depuis le timestamp.

### `host.log`

```ts
namespace host.log {
  function debug(msg: string, attrs?: Record<string, unknown>): void;
  function info(msg: string, attrs?: Record<string, unknown>): void;
  function warn(msg: string, attrs?: Record<string, unknown>): void;
}
```

Pas de `error` : utiliser `host.fail`.

Les logs handler-side sortent dans la sortie structurée d'One CLI quand `--debug` ou `--trace` est activé. Sinon ignorés silencieusement.

**Ne pas spam.** Cap de 1000 lignes de log par invocation.

### `host.fail`

```ts
namespace host.fail {
  function withCode(code: string, message: string, hint?: string): never;
}
```

Termine l'exécution avec une erreur structurée. Le `code` **doit** matcher une entrée du bloc `errors:` de l'action YAML. Sinon, l'host le mappe sur `unknown_error` et émet un warning au reviewer.

```ts
const res = host.http.request({ method: 'GET', url: '...' });
if (res.status === 404) {
  host.fail.withCode(
    'not_found',
    'Page does not exist or integration lacks access',
    'one install notion share-page'
  );
}
```

L'agent reçoit côté CLI :

```json
{
  "error": {
    "code": "not_found",
    "message": "Page does not exist or integration lacks access",
    "hint": "one install notion share-page",
    "install": {
      "service": "notion",
      "guide": "share-page",
      "requires_human": true
    }
  }
}
```

## Cycle de vie

**Une invocation = un module instancié.** Pas de réutilisation entre appels. Plus lent (~10ms d'overhead pour wazero AOT-compiled), mais isole les bugs d'état.

**Pas de fonction `main()`.** Le handler exporte une ou plusieurs fonctions par nom, dispatch via `handler_entry` du YAML :

```ts
// handler.ts compilé en handler.wasm
export function upload(inputs: UploadInputs): UploadOutput {
  // ...
}

export function bucket_create(inputs: BucketCreateInputs): BucketCreateOutput {
  // ...
}
```

Le host invoque `handler.upload(deserialize(stdin_json).inputs)`. Les signatures sont typées via les schémas YAML : le SDK génère les types depuis le `service.yaml`.

### Limites de ressources

```
memory:     64 MB par défaut, max 256 MB (configurable via service.yaml)
cpu time:   30s wall-clock par invocation (max 120s)
stack:      1 MB
http calls: 50 max par invocation
log lines:  1000 max par invocation
sleep:      30s par appel, 60s cumulés
```

Au-delà, kill brutal et erreur `resource_exhausted` côté agent.

## SDKs par langage

Trois toolchains officiellement supportées au début.

### TypeScript (recommandé pour débuter)

Compilé via [Javy](https://github.com/bytecodealliance/javy) (QuickJS embarqué dans WASM).

```bash
bun add -d @one-cli/handler-sdk-ts
bun add -d @bytecodealliance/javy
```

**`handlers/main.ts`** :

```ts
import { host } from '@one-cli/handler-sdk-ts';
import type { Inputs, Output } from './generated/pages-create.types';

export function pages_create(inputs: Inputs): Output {
  host.log.info('Creating Notion page', { parent: inputs.parent });

  const res = host.http.request({
    method: 'POST',
    url: 'https://api.notion.com/v1/pages',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      parent: inputs.parent,
      properties: inputs.properties,
      children: inputs.children,
    }),
  });

  if (res.status === 404) {
    host.fail.withCode(
      'not_found',
      'Parent page not accessible',
      'one install notion share-page'
    );
  }

  if (res.status >= 400) {
    const body = JSON.parse(new TextDecoder().decode(res.body));
    host.fail.withCode('api_error', body.message || 'Unknown error');
  }

  return JSON.parse(new TextDecoder().decode(res.body));
}
```

**Build** :

```bash
bun run build
# génère handlers/main.wasm
```

**Tests** (avec fake host) :

```ts
import { test, fakeHost } from '@one-cli/handler-test';
import { pages_create } from './main';

test('pages_create POSTs to /v1/pages', async () => {
  const host = fakeHost();
  host.creds.set('access_token', 'secret_token');
  host.http.expect({
    method: 'POST',
    url: 'https://api.notion.com/v1/pages',
    headers: { 'Content-Type': 'application/json' },
  }).respond({
    status: 200,
    body: { id: 'page_123', url: '...' },
  });

  const result = pages_create({
    parent: { page_id: 'parent_abc' },
    properties: { title: [{ text: { content: 'Test' } }] },
  });

  expect(result.id).toBe('page_123');
});
```

### Go (via tinygo)

```bash
tinygo build -o handlers/main.wasm -target=wasi handlers/main.go
```

**`handlers/main.go`** :

```go
package main

import (
    "encoding/json"
    "one.dev/handler"
)

//export pages_create
func pages_create() {
    var inputs struct {
        Parent     map[string]string      `json:"parent"`
        Properties map[string]interface{} `json:"properties"`
    }
    handler.ReadInputs(&inputs)

    body, _ := json.Marshal(inputs)
    res, err := handler.Host.HTTP.Request(handler.HTTPRequest{
        Method: "POST",
        URL:    "https://api.notion.com/v1/pages",
        Headers: map[string]string{
            "Content-Type": "application/json",
        },
        Body: body,
    })
    if err != nil {
        handler.Host.Fail("api_error", err.Error(), "")
    }

    if res.Status == 404 {
        handler.Host.Fail("not_found", "Parent not accessible",
            "one install notion share-page")
    }

    var out map[string]interface{}
    json.Unmarshal(res.Body, &out)
    handler.WriteOutput(out)
}

func main() {} // requis par tinygo
```

### Rust (pour les besoins performants)

```bash
cargo build --target wasm32-wasi --release
```

**`src/lib.rs`** :

```rust
use one_handler::{host, Inputs, Output};

#[no_mangle]
pub extern "C" fn pages_create() {
    let inputs: Inputs = host::read_inputs();
    let token = host::creds::get("access_token");

    let body = serde_json::to_vec(&inputs).unwrap();
    let res = host::http::request(host::HttpRequest {
        method: "POST".into(),
        url: "https://api.notion.com/v1/pages".into(),
        headers: [("Content-Type", "application/json")].into(),
        body: Some(body),
        ..Default::default()
    });

    if res.status == 404 {
        host::fail::with_code("not_found", "Parent not accessible",
            Some("one install notion share-page"));
    }

    let out: serde_json::Value = serde_json::from_slice(&res.body).unwrap();
    host::write_output(&out);
}
```

## Patterns courants

### Auth Bearer avec validation

```ts
const token = host.creds.get('access_token');
const res = host.http.request({
  method: 'GET',
  url: 'https://api.service.com/v1/me',
  headers: { Authorization: `Bearer ${token}` },
});
```

Souvent inutile si `injection: auto` est configuré.

### Signature SigV4 pour AWS

```ts
import { signSigV4 } from '@one-cli/handler-sdk-ts/aws';

const ak = host.creds.get('access_key_id');
const sk = host.creds.get('secret_access_key');
const region = host.creds.get('region');

const signed = signSigV4({
  method: 'PUT',
  url: `https://${inputs.bucket}.s3.${region}.amazonaws.com/${inputs.key}`,
  headers: { 'x-amz-content-sha256': '...' },
  body: inputs.body,
  service: 's3',
  region,
  credentials: { accessKeyId: ak, secretAccessKey: sk },
});

const res = host.http.request(signed);
```

Le helper `signSigV4` est fourni par le SDK, utilise `host.crypto.*` en interne.

### Chains d'appels avec rollback

```ts
// stripe customer.full-create : 3 appels avec rollback best-effort
export function customer_full_create(inputs: CustomerFullCreateInputs) {
  const apiKey = host.creds.get('api_key');
  const headers = { Authorization: `Bearer ${apiKey}` };

  // 1. Create customer
  const cust = postJSON('https://api.stripe.com/v1/customers', {
    email: inputs.email,
  }, { 'Idempotency-Key': inputs.idempotency_key });

  try {
    // 2. Attach payment method
    postJSON(`https://api.stripe.com/v1/payment_methods/${inputs.payment_method_id}/attach`, {
      customer: cust.id,
    });

    // 3. Set default
    postJSON(`https://api.stripe.com/v1/customers/${cust.id}`, {
      invoice_settings: { default_payment_method: inputs.payment_method_id },
    });

    return { customer_id: cust.id };
  } catch (e) {
    // Rollback best-effort
    try {
      host.http.request({
        method: 'DELETE',
        url: `https://api.stripe.com/v1/customers/${cust.id}`,
        headers,
      });
    } catch {}

    host.fail.withCode('partial_failure',
      'Customer was created but setup failed; rolled back',
      `Customer ID: ${cust.id}`);
  }
}
```

### Pagination auto

```ts
export function* messages_list_all(inputs: ListAllInputs) {
  let cursor: string | undefined;
  let count = 0;
  while (count < inputs.max_total) {
    const url = new URL('https://gmail.googleapis.com/gmail/v1/messages');
    url.searchParams.set('q', inputs.query);
    if (cursor) url.searchParams.set('pageToken', cursor);

    const res = host.http.request({ method: 'GET', url: url.toString() });
    const page = JSON.parse(new TextDecoder().decode(res.body));

    for (const msg of page.messages || []) {
      yield msg;
      count++;
      if (count >= inputs.max_total) return;
    }

    if (!page.nextPageToken) return;
    cursor = page.nextPageToken;
  }
}
```

Le SDK gère la sérialisation du generator en NDJSON pour l'host.

## Le `service.yaml > calls:` (allowlist URL)

C'est le mécanisme qui rend la sandbox réelle. Sans allowlist, un handler malveillant pourrait exfiltrer des credentials vers un serveur tiers.

```yaml
calls:
  - method: POST
    url: "https://api.notion.com/v1/pages"
  - method: GET
    url_pattern: "^https://api\\.notion\\.com/v1/pages/[a-f0-9-]+$"
  - method: PATCH
    url_pattern: "^https://api\\.notion\\.com/v1/pages/[a-f0-9-]+$"
```

**Patterns supportés** :

- `url` : match exact
- `url_pattern` : regex (validée à l'enregistrement)

**Bonnes pratiques** :

- Utiliser des patterns aussi spécifiques que possible. Pas de `^https://api\\.notion\\.com/.*` qui autorise tout.
- Listés à granularité fine (un endpoint = une entrée), pour faciliter le review.
- Méthode HTTP toujours explicitée.

**Vérifié en CI** : un linter analyse statiquement le handler et vérifie que toutes les URLs hit par le code matchent au moins un pattern.

## Versionnement du contrat host

Le `host_api_version` du service.yaml déclare la version d'API host attendue :

```yaml
host_api_version: 1
```

Le binaire vérifie au load. Si le handler attend une v2 mais le binaire n'expose que v1, refuse le load avec un message clair : "upgrade `one` to use this service".

Permet l'évolution du contrat host sans casser les anciens handlers. Les changements compatibles (ajout de fonctions) ne bumpent pas la version majeure ; les changements incompatibles (renommage, signature changée) sont rares mais bumpent.

## Distribution

Compilé par la CI du repo catalog, joint au tarball du service. Hash SHA-256 dans l'index, vérifié au download par `one`. Le source du handler est dans le repo, donc auditable, et le binaire est reproductible.

**Le contributeur ne commit jamais le `.wasm`.** C'est la CI qui le construit. Empêche la dérive entre source et binaire.

## Tests

### Pattern recommandé

Pour chaque handler :

1. Un test "happy path" : input valide, mock l'HTTP, vérifie le résultat.
2. Un test par code d'erreur déclaré dans `errors:` : assert que `host.fail.withCode` est appelé avec le bon code.
3. Un test pour chaque branche conditionnelle non triviale du handler.

### Exemple

```ts
import { test, fakeHost } from '@one-cli/handler-test';
import { pages_create } from './main';

test('happy path', async () => {
  const host = fakeHost();
  host.creds.set('access_token', 'tok');
  host.http.expect({
    method: 'POST',
    url: 'https://api.notion.com/v1/pages',
  }).respond({ status: 200, body: { id: 'page_123' } });

  const result = pages_create({ parent: { page_id: 'p' }, properties: {} });
  expect(result.id).toBe('page_123');
});

test('404 maps to not_found with hint', async () => {
  const host = fakeHost();
  host.creds.set('access_token', 'tok');
  host.http.expect({}).respond({ status: 404, body: { message: 'Not found' } });

  expect(() => pages_create({ parent: { page_id: 'p' }, properties: {} }))
    .toThrow(/not_found/);
  expect(host.fail.lastCall).toMatchObject({
    code: 'not_found',
    hint: 'one install notion share-page',
  });
});
```

### Lint en CI

Le repo catalog tourne un lint statique sur chaque PR :

- Toutes les URLs hit par le handler matchent une entrée de `calls:`
- Toutes les `host.creds.get(key)` matchent une entrée de `credentials:`
- Tous les `host.fail.withCode(code, ...)` matchent une entrée de `errors:`
- Pas d'`import` interdit (pas de `fs`, `child_process`, etc.)

C'est ce qui rend la communauté capable de contribuer sans casser. Le linter attrape 80% des erreurs avant code review.

## Anti-patterns

### Stocker un token en variable globale

```ts
// MAUVAIS
let cachedToken: string | null = null;
export function action(inputs) {
  if (!cachedToken) cachedToken = host.creds.get('access_token');
  // ...
}
```

Une invocation = un module fresh. Le cache ne survit pas et c'est une source de confusion. Toujours `host.creds.get` à chaque invocation.

### Faire des appels HTTP non déclarés

```ts
// MAUVAIS : crash à l'exécution avec url_not_allowed
host.http.request({ url: 'https://my-personal-server.com/log' });
```

Toutes les URLs doivent être dans `service.yaml > calls:`.

### Logger des secrets

```ts
// MAUVAIS
host.log.info('Auth', { token: host.creds.get('access_token') });
```

Les `Secret` côté host sont redactés, mais les valeurs retournées par `host.creds.get` côté handler sont des strings. Le host n'a pas de moyen sûr de détecter la fuite. **Ne logge jamais ce que tu reçois de `creds.get`.**

### Bypasser l'injection auto

```yaml
# service.yaml
auth:
  providers:
    oauth:
      injection:
        header: Authorization
        format: "Bearer {access_token}"
        inject: auto
```

```ts
// MAUVAIS : injection auto déjà active, tu duplifies
const token = host.creds.get('access_token');
host.http.request({
  url: '...',
  headers: { Authorization: `Bearer ${token}` },
});
```

Avec `inject: auto`, le handler n'a pas besoin de toucher au token. Plus simple et plus sûr.

### Pas gérer les erreurs

```ts
// MAUVAIS : le handler renvoie un crash sale au lieu d'une erreur typée
const res = host.http.request({ ... });
return JSON.parse(new TextDecoder().decode(res.body));
// Si res.status != 200, le JSON sera potentiellement invalide
```

Toujours brancher sur le status code et appeler `host.fail.withCode` avec un code mappé.

## Performance

### Cold start d'un handler

Un module WASM est compilé à la première utilisation puis caché. Le cache est partagé entre invocations du même processus, mais pas entre processus (chaque `one ...` est un process neuf).

Cold start handler typique : 10-30ms (compilation) + 5ms (instanciation). Acceptable pour la majorité des cas. Si critique, le binaire peut pré-compiler les handlers fréquents en AOT (`wazero compile`) au moment du `one catalog update`.

### Memory footprint

Un handler simple consomme ~5-10 MB. Un handler complexe avec beaucoup de string manipulation peut monter à 30-50 MB. Le cap à 64 MB par défaut est large.

Si un handler approche les 64 MB, c'est probablement un bug (fuite, boucle, accumulation). Augmenter le cap est rarement la bonne solution.

### Throughput

Pas le bottleneck d'un CLI. Si un handler fait 10ms de logique + 200ms de latence HTTP, le total est dominé par le HTTP. WASM ajoute <5ms d'overhead.

## Sécurité (résumé)

Le détail est dans [SECURITY.md](./SECURITY.md). Les points clés pour un auteur de handler :

1. **Allowlist URL stricte.** Pas d'évasion possible.
2. **Credentials lues uniquement via `host.creds.get`.** Déclarées dans `service.yaml > credentials`.
3. **Pas d'I/O hors host functions.** Pas de filesystem, pas d'env vars, pas d'exec.
4. **Codes d'erreur sortants typés.** Pas de string-matching côté utilisateur.
5. **Pas de logging de secrets.** Toujours.

Si tu suis ces règles, ton handler peut être mergé sans inquiétude.

---

*Pour proposer une évolution du contrat host (nouvelle fonction, nouveau type), ouvrir un RFC dans `one-cli/rfcs`. Toute évolution doit être backward-compatible avec les handlers existants.*
