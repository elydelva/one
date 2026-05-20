# CLI.md

> Référence exhaustive des commandes du binaire `one`. Pour les concepts (scope, vault, catalog), voir les docs dédiés.

## Conventions générales

### Format des commandes

```
one [global flags] <command> [args] [command flags]
one <service> <action> [inputs]                # forme courte pour l'exec
```

### Flags globaux

| Flag | Description |
|---|---|
| `--profile <name>` | Profil de scope à utiliser (override de `.onerc.yaml`) |
| `--as <alias>` | Compte à utiliser pour cette commande (override de `defaults`) |
| `--dry-run` | Exécute sans effet de bord (pour les actions mutations) |
| `--json` | Force la sortie JSON même en TTY |
| `--tty` | Force la sortie TTY même en pipe |
| `--quiet`, `-q` | Réduit la verbosité (warnings seulement) |
| `--debug` | Verbose, affiche les requêtes HTTP, durées, traces |
| `--trace <id>` | Préfixe un trace ID custom au lieu d'en générer un |
| `--no-color` | Désactive les couleurs ANSI |
| `--catalog-dir <path>` | Override le catalog local |
| `--help`, `-h` | Affiche l'aide de la commande |
| `--version` | Affiche la version et exit 0 |

### Détection TTY

Par défaut :

- **stdout est un TTY** : sortie colorée et formatée
- **stdout est piped** : sortie JSON

Override possible via `--json` ou `--tty`. Important : si tu pipe l'output dans `jq`, tu auras du JSON automatiquement.

## Exit codes

Convention stable, **API publique pour les agents**.

| Code | Signification | Quand |
|---|---|---|
| 0 | Succès | L'action s'est exécutée et a réussi |
| 1 | Erreur générique | Bug interne, erreur réseau non spécifique, validation d'inputs |
| 2 | Non authentifié | Aucun credential pour ce service/account, ou refresh impossible |
| 3 | Hors scope | L'action n'est pas autorisée par `.onerc.yaml` |
| 4 | Setup requis | Une étape humaine est nécessaire (`one install` proposé) |
| 5 | Service ou action inconnu | Pas dans le catalogue |

Tout exit code >5 est un bug. L'agent peut s'appuyer sur ces codes sans parser stderr.

## Format de sortie JSON

Schéma stable pour `one <service> <action>` en mode JSON :

```json
{
  "ok": true,
  "data": { ... },                  // output de l'action
  "trace_id": "01HXYZ...",
  "warnings": []
}
```

En cas d'erreur :

```json
{
  "ok": false,
  "error": {
    "code": "not_in_scope",
    "message": "Permission github.repos.delete not allowed",
    "hint": "Run `one scope add github repos.delete` to allow",
    "install": null
  },
  "trace_id": "01HXYZ..."
}
```

Avec `install` quand pertinent :

```json
{
  "ok": false,
  "error": {
    "code": "setup_required",
    "message": "Page not accessible",
    "install": {
      "service": "notion",
      "guide": "share-page",
      "command": "one install notion share-page",
      "requires_human": true
    }
  }
}
```

## Commandes par catégorie

### Initialisation

#### `one init`

Crée un `.onerc.yaml` vide dans le répertoire courant.

```bash
one init                              # crée .onerc.yaml minimal
one init --from-template readonly     # template pré-configuré
```

Effet :

```yaml
# .onerc.yaml créé
version: 1
services: {}
```

Ajoute aussi `.onerc.local.yaml` à un `.gitignore` créé/modifié.

#### `one doctor`

Diagnostic complet de l'installation et de la config.

```bash
one doctor
```

Sortie :

```
✓ Binary version: v0.3.2
✓ Catalog: 47 services indexed (last update: 2h ago)
✓ Vault: keychain (macOS)
  ├─ github:work     authenticated, refresh in 23m
  ├─ github:perso    authenticated, refresh in 1h2m
  └─ notion:kaampus  authenticated, refresh in 4h
✓ Scope file: .onerc.yaml v1 valid
✓ Lock file: .onerc.lock matches installed catalog
⚠ Project has 3 services in scope, but only 2 are authenticated
  → Run `one login stripe` to complete setup

Recommendations:
  - Update catalog: `one catalog update` (last: 2h ago)
```

Exit 0 si tout OK, 1 si warnings, 2 si erreurs.

#### `one upgrade`

Upgrade le binaire vers la dernière version stable.

```bash
one upgrade                           # latest stable
one upgrade --to v0.3.2               # version spécifique
one upgrade --check                   # juste check, n'installe pas
```

### Authentification

#### `one login <service>`

Authentifie au service via le provider par défaut.

```bash
one login github                      # OAuth (provider par défaut)
one login github --provider pat       # provider spécifique
one login github --as perso           # crée un nouvel alias
one login github --device             # device flow (headless)
one login github --client-id ABC123   # BYOC
```

Voir [AUTH.md](./AUTH.md) pour le détail de chaque provider.

#### `one logout <service>`

Supprime un compte du vault.

```bash
one logout github                     # supprime "default"
one logout github --account perso     # supprime "perso"
one logout --all                      # supprime TOUS les comptes (confirmation requise)
```

#### `one accounts [service]`

Liste les comptes authentifiés.

```bash
one accounts                          # tous les services
one accounts github                   # un service spécifique
```

Sortie :

```
github
  work    elydelva@protonmail.com     valid (refresh in 1h2m)
  perso   ely.delvallee@gmail.com     valid (refresh in 23m)

notion
  kaampus elydelva                    valid

stripe
  test    sk_test_...                 valid
```

#### `one rotate <service> <account>`

Force un re-login (cas de fuite suspectée).

```bash
one rotate github work
```

Révoque l'ancien token via l'endpoint OAuth si supporté, puis ouvre le flow de login.

#### `one refresh <service> <account>`

Force un refresh manuel sans attendre l'expiration.

```bash
one refresh github work
```

Utile pour diagnostiquer un problème de refresh.

#### `one vault export`

Exporte le vault dans un fichier age chiffré.

```bash
one vault export --to /path/to/vault.age
# prompt for passphrase
```

Pour backup ou pour transférer vers une CI.

#### `one vault import`

Importe un vault depuis un fichier age.

```bash
one vault import /path/to/vault.age
# prompt for passphrase
```

Le vault courant est fusionné (les comptes existants ne sont pas écrasés sauf flag `--overwrite`).

#### `one vault status`

Affiche quelle source de vault est active.

```bash
one vault status
```

Sortie :

```
Active source: keychain (macOS)
Env var override: none

Other sources available:
  - age file: ~/.one/vault.age (would be used if keychain unavailable)
```

### Scope et permissions

#### `one scope show`

Affiche le scope effectif après merge.

```bash
one scope show                        # JSON par défaut
one scope show --tty                  # format lisible
one scope show --raw                  # affiche les deux fichiers sans merge
```

#### `one scope add <service> <permission>`

Ajoute une permission à l'`allow`.

```bash
one scope add github issues.read
one scope add github "issues.*"       # glob, quote pour le shell
one scope add github issues.delete --deny
one scope add github issues.read --local  # dans .onerc.local.yaml
```

#### `one scope remove <service> <permission>`

Inverse de `add`.

```bash
one scope remove github issues.read
one scope remove github issues.delete --deny
```

#### `one scope use <service> --as <alias>`

Change le compte par défaut pour un service.

```bash
one scope use github --as perso
# modifie defaults.github ou services.github.account
```

#### `one scope check`

Validation exhaustive.

```bash
one scope check
one scope check --strict              # warnings deviennent erreurs
```

Voir [SCOPE.md](./SCOPE.md#one-scope-check) pour la liste des checks.

#### `one scope explain <service> <permission>`

Trace l'évaluation d'une permission.

```bash
one scope explain github issues.delete
```

Sortie :

```
Permission: github.issues.delete
Result: DENIED

Evaluated in order:
  1. .onerc.local.yaml > deny: matched 'issues.delete' (exact) → DENY (final)

Rules not evaluated (would have applied if no prior match):
  - .onerc.yaml > allow: 'issues.*' would have matched

To allow:
  Remove the deny rule:
    one scope remove github issues.delete --deny --local
```

### Catalogue et lock

#### `one catalog update`

Fetch les dernières versions du catalogue depuis le CDN.

```bash
one catalog update                    # interactif
one catalog update --pin              # update les services et regénère le lock
one catalog update --check            # juste check, n'installe pas
```

#### `one catalog search <query>`

Recherche dans le catalogue.

```bash
one catalog search payment
# affiche stripe, paypal, square, ...
```

#### `one lock`

Génère ou met à jour `.onerc.lock`.

```bash
one lock                              # snapshot des versions actuelles
one lock --update notion              # update un service spécifique
one lock --update-all                 # update tous
one lock --check                      # vérifie matche actuel (exit 0/1)
```

### Exécution d'actions

#### `one <service> <action>`

Forme principale.

```bash
one github issues.read --issue 42
one notion pages.create \
  --parent_page_id abc-123 \
  --properties '{"title":[{"text":{"content":"Hello"}}]}'
one stripe customers.create \
  --email user@example.com \
  --name "Jane Doe" \
  --idempotency_key cust-2026-05-20-001
```

#### Passer des inputs

Trois modes possibles selon le type :

**1. Flags simples (string, number, bool)** :

```bash
one github issues.read --issue 42 --include_pull_requests true
```

**2. Flags JSON pour les objets** :

```bash
one notion pages.create --properties '{"title":[...]}'
```

**3. `@file` pour les fichiers** :

```bash
one notion blocks.append --children @blocks.json
one s3 objects.put --bucket mybucket --key file.pdf --body @./file.pdf
```

**4. stdin pour piping** :

```bash
echo '{"properties": {...}}' | one notion pages.create --stdin
cat data.json | one bigservice action --stdin
```

#### Options pour les mutations

```bash
--dry-run                             # validation sans side effect
--idempotency-key <key>               # passage explicite (sinon généré)
--confirm                             # requis pour destructive en TTY
```

### Introspection (verbes agent)

#### `one capabilities`

JSON listant les services et actions disponibles.

```bash
one capabilities                      # tous les services + actions
one capabilities github               # juste github
one capabilities --scope-only         # seulement ce qui est dans le scope courant
```

Sortie typique :

```json
{
  "services": [
    {
      "name": "github",
      "version": "2.1.4",
      "actions": [
        {
          "id": "issues.read",
          "permission": "issues.read",
          "kind": "query",
          "in_scope": true,
          "summary": "Read an issue by number"
        },
        {
          "id": "issues.create",
          "permission": "issues.write",
          "kind": "mutation",
          "side_effects": "write",
          "in_scope": false,
          "summary": "Create a new issue"
        }
      ]
    }
  ]
}
```

#### `one info`

Documentation markdown.

```bash
one info                              # skill global "onecli"
one info github                       # skill du service github
one info github issues.create         # doc d'une action spécifique
```

Sortie : markdown pour les humains et les agents qui aiment le markdown. Pour parsing structuré, utiliser `capabilities`.

#### `one can <service> <action>`

Precheck rapide de permission, sans exécuter.

```bash
one can github issues.delete
# exit 0 si OK, exit 3 si pas dans le scope, exit 2 si pas authentifié
```

Utile pour les agents qui veulent vérifier avant de tenter, ou pour scripter.

### Install et setup

#### `one install <service> <guide>`

Affiche un guide d'install.

```bash
one install notion share-page
one install notion share-page --json  # output structuré pour agent
one install notion share-page --verify --page_id abc
```

En mode TTY, render le markdown joliment avec une checklist navigable. En mode JSON, retourne le frontmatter + content :

```json
{
  "guide": {
    "id": "share-page",
    "title": "Share a Notion page with your integration",
    "estimated_time": "30s",
    "requires_human": true,
    "content_md": "Notion's permission model is opt-in...",
    "verify_command": "one install notion share-page --verify --page_id <PAGE_ID>",
    "open_url": "https://notion.so"
  }
}
```

### Skill et intégration IDE

#### `one skill`

Affiche le skill markdown du binaire.

```bash
one skill                             # affiche le skill onecli
one skill <service>                   # skill d'un service spécifique
one skill --install                   # détecte l'IDE et installe le skill
one skill --install --ide claude-code # force un IDE
```

`--install` :

- Détecte Claude Code, Cursor, Aider, etc.
- Crée le fichier au bon endroit (genre `.claude/skills/onecli.md`)
- Affiche un message de confirmation

### Audit et debug

#### `one trace`

Affiche le journal d'audit.

```bash
one trace                             # 50 dernières entrées
one trace --since=1h                  # depuis 1h
one trace --since=2026-05-20          # depuis une date
one trace --auth                      # seulement les events d'auth
one trace <trace_id>                  # détail d'une exécution spécifique
one trace --service github            # filtré par service
```

Format : NDJSON par défaut, TTY tabulaire en interactif.

#### `one trace <trace_id>` (détail)

Trace complète d'une exécution :

```
Trace: 01HXYZ123ABC
Service: notion
Action: pages.read
Account: kaampus
Start: 2026-05-20T14:32:11Z
Duration: 234ms
Result: success

Scope check:
  Permission: pages.read
  Result: ALLOWED via .onerc.yaml > allow: ['pages.*']

Auth:
  Provider: oauth
  Refreshed: no (valid 47m23s)

HTTP calls:
  1. GET https://api.notion.com/v1/pages/abc-123
     Status: 200
     Duration: 198ms

Output: 1.4 KB JSON
```

#### `--debug`

Active la verbosité maximale.

```bash
one --debug github issues.read --issue 42
```

Affiche sur stderr :

- Logs slog en JSON (toutes les étapes du use case)
- Requêtes HTTP émises (méthode, URL, status, durée)
- Trace ID
- Source du vault utilisée
- Décisions de scope

Utile pour debugger un comportement inattendu.

## Variables d'environnement

| Variable | Description |
|---|---|
| `ONE_CATALOG_URL` | Override l'URL du CDN du catalogue |
| `ONE_CATALOG_DIR` | Override le dossier local du catalogue |
| `ONE_VAULT_FILE` | Path du fichier age vault (fallback si pas de keychain) |
| `ONE_VAULT_PASSPHRASE` | Passphrase du vault age (sinon prompt interactif) |
| `ONE_PROFILE` | Profil de scope actif |
| `ONE_DEBUG` | Active le mode debug (équivalent à `--debug`) |
| `ONE_NO_COLOR` | Désactive les couleurs |
| `ONE_<SERVICE>_CLIENT_ID` | Client ID OAuth pour un service (BYOC) |
| `ONE_<SERVICE>_API_BASE` | Override l'URL de base d'une API (testing) |
| `ONE_CREDS_<SERVICE>_<ACCOUNT>` | Credentials inline (JSON) pour override |
| `ONE_PPROF` | Adresse pour le serveur pprof (debug, ex: `:6060`) |
| `ONE_HOME` | Override `~/.one/` (par défaut : XDG ou home) |

### Exemples d'utilisation

```bash
# CI : credentials via env, no keychain
export ONE_CREDS_GITHUB_DEFAULT='{"access_token":"ghp_xxx","provider":"pat","service":"github","account":"default"}'
one github issues.list --repo me/repo

# Override d'URL pour tests E2E
export ONE_GITHUB_API_BASE=http://localhost:8080
one github issues.read --issue 1

# Profile production
export ONE_PROFILE=production
one stripe customers.create --email ...

# Debug avec pprof
export ONE_PPROF=:6060
one github issues.read --issue 1 &
go tool pprof http://localhost:6060/debug/pprof/profile
```

## Fichiers utilisés

### Par projet (cwd)

| Fichier | Description |
|---|---|
| `.onerc.yaml` | Scope file principal (commité) |
| `.onerc.local.yaml` | Override personnel (gitignored) |
| `.onerc.lock` | Versions du catalogue figées (commité) |

### Globaux (`~/.one/`)

| Fichier/dossier | Description |
|---|---|
| `~/.one/catalog/` | Cache du catalogue téléchargé |
| `~/.one/catalog/_index.json` | Index des services |
| `~/.one/catalog/services/<name>/` | Définition d'un service |
| `~/.one/audit.log` | Journal d'audit local (NDJSON, rotation 30j) |
| `~/.one/locks/` | File locks pour les opérations concurrentes |
| `~/.one/vault.age` | Vault age (si non keychain) |
| `~/.one/config.yaml` | Config globale (catalog URL, log level, etc.) |
| `~/.one/cache/` | Caches divers (réponses HTTP courtes, etc.) |

### Convention XDG

Si `XDG_CONFIG_HOME` est défini, utilise `$XDG_CONFIG_HOME/one/` au lieu de `~/.one/`. Idem `XDG_DATA_HOME` pour le cache et l'audit.

## Workflow complet

### Première installation

```bash
# Installer le binaire
curl -fsSL https://one-cli.dev/install.sh | sh

# Vérifier
one --version
one doctor

# (Optionnel) Installer le skill dans Claude Code
one skill --install
```

### Premier projet

```bash
cd mon-projet
one init                              # crée .onerc.yaml

# Login aux services
one login github
one login notion --as kaampus

# Définir le scope
one scope add github issues.*
one scope add github pulls.read
one scope add notion pages.*

# Verrouiller
one lock

# Commit
git add .onerc.yaml .onerc.lock .gitignore
git commit -m "init: setup One CLI"
```

### Workflow agent

```bash
# Discovery
one capabilities --scope-only          # qu'est-ce que je peux faire ?
one info github                        # comment ça marche ?

# Pre-check
one can github issues.create          # est-ce autorisé ?

# Exécution
one github issues.create \
  --repo me/myrepo \
  --title "Bug: foo" \
  --body "Detailed description"
# stdout: {"ok":true,"data":{"id":42,"url":"https://github.com/..."}}

# En cas d'erreur de setup
# stdout: {"ok":false,"error":{"code":"setup_required",...,"install":{"command":"one install ..."}}}
# L'agent sait quoi faire : suggérer à l'humain de lancer le install
```

### Workflow CI

```bash
# Dans le pipeline
export ONE_VAULT_FILE=/tmp/vault.age
export ONE_VAULT_PASSPHRASE="$VAULT_PASSPHRASE"

aws s3 cp s3://my-secrets/vault.age $ONE_VAULT_FILE

one doctor                            # vérifie tout est OK
one lock --check                      # vérifie le catalog matche

# Run l'agent ou les actions
one github issues.list --repo me/repo
```

## Comportement détaillé

### Resolution de la commande `one <service> <action>`

Algorithme :

1. Parser les flags globaux
2. Identifier `<service>` : s'il matche une commande built-in (login, scope, info, etc.), router vers celle-ci
3. Sinon, charger `<service>` depuis le catalogue
4. Charger `<action>` depuis ce service
5. Parser les flags de l'action selon son schéma d'inputs
6. Charger le scope file
7. Vérifier que `action.permission` est dans le scope
8. Résoudre le compte (default → local → --as)
9. Fetch credentials du vault
10. Refresh si nécessaire
11. Si setup requis : retourner ErrSetupRequired (exit 4)
12. Invoquer le runtime (déclaratif ou WASM)
13. Render output

Chaque étape peut court-circuiter avec un exit code spécifique.

### Comportement sur erreur de validation d'inputs

```bash
one github issues.read --issue not-a-number
# stderr: Error: invalid input 'issue': expected integer, got "not-a-number"
# exit 1
```

L'erreur est validée **avant** tout appel réseau. Pas de gaspillage de quota.

### Comportement sur action inconnue

```bash
one github does.not.exist
# stderr: Error: unknown action 'does.not.exist' in service 'github'
#         Did you mean: 'issues.delete'?
# exit 5
```

Suggestion via distance de Levenshtein si une action proche existe.

### Comportement sur service inconnu

```bash
one unknown-service action
# stderr: Error: unknown service 'unknown-service'
#         Run `one catalog search <query>` to find available services.
# exit 5
```

### Comportement piped

```bash
one github issues.read --issue 42 | jq .data.title
```

Le binaire détecte que stdout est piped, switch en JSON automatiquement. Stderr reste lisible pour les warnings.

### Comportement TTY interactif

```bash
one stripe customers.delete --customer cus_xxx
# Confirm: This will permanently delete customer cus_xxx. Type 'yes' to confirm: _
```

Pour les actions `side_effects: destructive`, prompt de confirmation en TTY. Bypass via `--confirm` ou en pipe (où le prompt n'a pas de sens).

## Codes d'erreur sortants (au-delà des exit codes)

Codes stables dans le champ `error.code` du JSON output :

| Code | Sens |
|---|---|
| `not_authenticated` | aucun credential ou refresh impossible |
| `not_in_scope` | permission refusée par scope file |
| `setup_required` | action humaine requise (avec install hint) |
| `unknown_service` | service pas dans le catalogue |
| `unknown_action` | action pas dans le service |
| `invalid_input` | inputs ne matchent pas le schéma |
| `api_error` | erreur retournée par le service tiers |
| `rate_limited` | service tiers rate limit (avec retry-after) |
| `not_found` | ressource inexistante |
| `forbidden` | les credentials sont valides mais pas le droit |
| `network_error` | timeout, DNS fail, connexion refusée |
| `internal_error` | bug du binaire (à reporter) |
| `lock_violation` | lock file ne matche pas le catalog installé |
| `url_not_allowed` | un handler a tenté un appel hors allowlist (bug catalog) |
| `resource_exhausted` | timeout/memory/calls limits hit |

L'agent peut switcher selon le code sans parser le message.

## Conventions de naming des actions

Les noms d'actions suivent `<resource>.<verb>`, en minuscules :

| Verbe | Sémantique |
|---|---|
| `read`, `get`, `retrieve` | lecture d'une ressource par ID |
| `list` | liste paginée de ressources |
| `search`, `query` | recherche avec filtres |
| `create` | création |
| `update` | update partiel |
| `replace` | update total |
| `delete`, `archive` | suppression (destructive) |
| `enable`, `disable` | toggle d'état |
| `attach`, `detach` | gestion de liens |
| `subscribe`, `unsubscribe` | événements |

Si tu as un doute en contribuant un service, regarde des services existants du même domaine.

## Commandes pour les contributeurs au catalogue

#### `one catalog lint <path>`

Validation d'un service local.

```bash
one catalog lint ./services/my-service
```

#### `one catalog scaffold <name>`

Crée la structure d'un nouveau service.

```bash
one catalog scaffold my-service --lang ts
# crée services/my-service/{service.yaml, handlers/main.ts, package.json, tests/}
```

#### `one catalog test <path>`

Run les tests d'un service local (handlers + integration).

```bash
one catalog test ./services/my-service
```

## Bash completion

```bash
one completion bash >> ~/.bashrc
one completion zsh >> ~/.zshrc
one completion fish > ~/.config/fish/completions/one.fish
```

Génère les fichiers de completion pour le shell. Inclut les services et actions du catalogue installé.

---

*Toute commande ajoutée requiert : test d'intégration dans `internal/app/`, test E2E dans `tests/e2e/`, doc dans ce fichier, entrée dans le `one skill` si l'agent doit la connaître.*
