# CLI.md

> Référence exhaustive des commandes du binaire `one`. Pour les concepts (scope, vault, catalog), voir les docs dédiés.

## Conventions générales

### Format des commandes

```
one [global flags] <command> [args] [command flags]
one <service> <action> [inputs]                # forme courte pour l'exec
```

### Flags globaux

Persistants sur la racine cobra :

| Flag | Description |
|---|---|
| `--json` | Force la sortie JSON même en TTY (défaut : auto-détecté) |
| `--account <alias>` | Compte à utiliser (équivalent de `--as` côté exec) |
| `--dry-run` | Exécute sans effet de bord (pour les mutations) |
| `--project <dir>` | Dossier de projet (défaut : cwd) |
| `--help`, `-h` | Affiche l'aide |
| `--version`, `-v` | Affiche la version |

Profil : se contrôle via la variable d'environnement `ONE_PROFILE` (pas un flag). Pas de `--tty`, `--quiet`, `--debug`, `--trace`, `--no-color`, `--catalog-dir` en v0.4.

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

Crée un `.onerc.yaml` minimal dans le répertoire courant et ajoute `.onerc.local.yaml` au `.gitignore` (créé ou complété).

```bash
one init
```

Pas de `--from-template` en v0.4.

#### `one doctor`

Diagnostic complet de l'installation et de la config.

```bash
one doctor
```

Sortie : une ligne par check, préfixée par `✓` (ok), `!` (warn) ou `✗` (fail).

```
✓ scope    2 service(s) in scope
✓ catalog:github
! vault:github   no accounts (run `one login github`)
✓ lock     /path/.onerc.lock
✓ home     /Users/x/.one
```

Exit 0 si aucun fail, 1 dès qu'un check `fail` apparaît (les `warn` ne font pas échouer).

#### `one upgrade`

Reporté à v0.5.

### Authentification

#### `one login <service>`

Authentifie au service. Provider par défaut : `pat`.

```bash
one login github                              # provider pat
one login github --provider oauth2_user       # OAuth user-flow
one login github --provider oauth2_device     # device flow (headless)
one login github --as perso                   # crée/écrase l'alias "perso"
```

Pas de `--client-id` flag en v0.4 (les `client_id` sont résolus depuis le catalogue / variables d'env documentées par le service). Voir [AUTH.md](./AUTH.md).

#### `one logout <service>`

Supprime un compte du vault.

```bash
one logout github                     # supprime "default"
one logout github --as perso          # supprime "perso"
```

Pas de `--all` en v0.4.

#### `one accounts <service>`

Liste les comptes registrés pour un service (arg obligatoire).

```bash
one accounts github
```

Sortie : une ligne `<service>:<alias>` par compte, ou `no accounts`.

#### `one rotate <service> <account>`

Re-run le login flow et écrase la credential dans le vault.

```bash
one rotate github work
```

La révocation côté OAuth provider est reportée à v0.5.

#### `one refresh <service> <account>`

Force un refresh manuel sans attendre l'expiration.

```bash
one refresh github work
```

Utile pour diagnostiquer un problème de refresh.

#### `one vault export`

Dump JSON **plaintext** des credentials in-scope sur stdout. Pipe à `age -p` (ou équivalent) avant de persister.

```bash
one vault export | age -p > backup.age
```

#### `one vault import`

Restaure depuis un JSON bundle lu sur stdin. Écrase systématiquement les entrées existantes (pas de flag `--overwrite` : c'est le comportement par défaut).

```bash
age -d backup.age | one vault import
```

#### `one vault status`

JSON : compte de credentials par service en scope.

```bash
$ one vault status
{
  "services": { "github": 2, "notion": 1 },
  "total": 3
}
```

### Scope et permissions

#### `one scope show [service]`

Affiche le scope effectif (merge `.onerc.yaml` + `.onerc.local.yaml`, et override `.onerc.<profile>.yaml` si `ONE_PROFILE` est défini), au format JSON.

```bash
one scope show
one scope show github
```

#### `one scope add <service> <permission>`

Ajoute une permission à `allow` (ou à `deny` avec `--deny`). Écrit toujours dans `.onerc.yaml`.

```bash
one scope add github issues.read
one scope add github "issues.*"
one scope add github issues.delete --deny
```

Pas de `--local` en v0.4.

#### `one scope remove <service> <permission>`

Retire une permission (cherche dans `allow` et `deny`).

```bash
one scope remove github issues.read
```

#### `one scope check <service> <action>`

Exit 0 si autorisé, exit 3 (`ErrNotInScope`) sinon.

```bash
one scope check github issues.delete
```

#### `one scope explain <service> <action>`

Affiche en JSON `{allowed, reason, service, action}` puis exit 0 si autorisé, non-zéro avec la raison sinon.

`one scope use`, `--strict`, `--raw` : reportés à v0.5.

### Catalogue et lock

#### `one catalog ...`

`one catalog update`, `search`, `lint`, `scaffold`, `test` : reportés à v0.5. En v0.4, le catalogue HTTP est piloté par `ONE_CATALOG_URL` (cache 15 min) et la chaîne FS → HTTP. Cf. [CATALOG.md](./CATALOG.md).

#### `one lock`

Génère ou met à jour `.onerc.lock` (schema v1).

```bash
one lock                              # (re)génère depuis le scope courant
one lock --update notion              # refresh un service
one lock --update-all                 # refresh tous les services en scope
one lock --check                      # exit 1 (ErrLockDrift) si drift
```

`--check` retourne une erreur `lock drift detected: ...` listant les services divergents, avec hint `run \`one lock --update-all\` to refresh`.

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

**4. stdin pour piping** : reporté à v0.5.

#### Options pour les mutations

```bash
--dry-run                             # validation sans side effect
```

`--idempotency-key` / `--confirm` : reportés à v0.5.

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

#### `one install <service> [guide]`

Affiche un guide d'install. Sans `[guide]`, requiert `--list`.

```bash
one install notion share-page          # affiche le guide
one install notion --list              # liste tous les guides du service
```

Sortie TTY (guide simple) :

```
# <title>

<content markdown>

Verify: one <service> <action>          # si le frontmatter a `verify.action`
```

`--list` affiche `<id>\t<title>` par ligne. Pas de `--verify`, ni d'exécution automatique en v0.4.

### Skill et intégration IDE

#### `one skill`

Stub en v0.4: retourne `not implemented`. Le flag `--install` est déclaré mais inerte. Reporté à v0.5.

### Audit et debug

#### `one trace`

Câblé côté CLI mais l'implémentation retourne `not implemented`. Persistence de l'audit log reportée à v0.5.

#### `--debug`

Reporté à v0.5. Le logger interne tourne à niveau `warn` en sortie texte sur stderr.

## Variables d'environnement

| Variable | Description |
|---|---|
| `ONE_CATALOG_URL` | Active la couche HTTP du catalogue (sinon FS seul) |
| `ONE_CATALOG_ROOT` | Override le dossier local du catalogue (défaut `$HOME/.one/catalog`) |
| `ONE_AGE_VAULT_PATH` | Path du fichier age vault (défaut `$HOME/.one/vault.age`) |
| `ONE_AGE_PASSPHRASE` | Passphrase du vault age (requise pour activer la couche age) |
| `ONE_PROFILE` | Profil de scope actif (charge `.onerc.<profile>.yaml`) |
| `ONE_CREDS_<SERVICE>_<ACCOUNT>` | Credential inline (JSON storage shape) — vault read-only |
| `ONE_CERT_<SERVICE>_<ACCOUNT>_CERT` | Chemin PEM cert client (provider `certificate`) |
| `ONE_CERT_<SERVICE>_<ACCOUNT>_KEY` | Chemin PEM clé privée (provider `certificate`) |
| `ONE_TRANSPORT_ALLOW_HTTP` | `1` pour tolérer `http://` (tests; refuse par défaut) |
| `ONE_TRANSPORT_ALLOWED_HOSTS` | Bypass SSRF pour ces hosts (CSV) |

`ONE_DEBUG`, `ONE_NO_COLOR`, `ONE_<SERVICE>_API_BASE`, `ONE_PPROF`, `ONE_HOME`, `XDG_CONFIG_HOME` : pas câblés en v0.4.

### Exemples d'utilisation

```bash
# CI: credentials via env, no keychain
export ONE_CREDS_GITHUB_DEFAULT='{"access_token":"ghp_xxx","provider":"pat","service":"github","account":"default"}'
one github issues.list --repo me/repo

# CI: vault age depuis un secret
export ONE_AGE_VAULT_PATH=/tmp/vault.age
export ONE_AGE_PASSPHRASE="$VAULT_PASSPHRASE"

# Profil restrictif
export ONE_PROFILE=production
one stripe customers.create --email ...
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
| `~/.one/catalog/` | Catalogue local FS (override via `ONE_CATALOG_ROOT`) |
| `~/.one/locks/<service>:<alias>.lock` | File locks de refresh (gofrs/flock, 10s timeout) |
| `~/.one/vault.age` | Vault age (si la couche age est activée) |
| `~/.one/cache/wasm/` | Cache des modules WASM compilés |

`audit.log`, `config.yaml`, convention XDG : reportés à v0.5.

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
export ONE_AGE_VAULT_PATH=/tmp/vault.age
export ONE_AGE_PASSPHRASE="$VAULT_PASSPHRASE"

aws s3 cp s3://my-secrets/vault.age $ONE_AGE_VAULT_PATH

one doctor                            # vérifie tout est OK
one lock --check                      # exit 1 (ErrLockDrift) si drift

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

`one catalog lint`, `scaffold`, `test`, ainsi que `one completion` (bash/zsh/fish) : reportés à v0.5.

---

*Toute commande ajoutée requiert : test d'intégration dans `internal/app/`, test E2E dans `tests/e2e/`, doc dans ce fichier, entrée dans le `one skill` si l'agent doit la connaître.*
