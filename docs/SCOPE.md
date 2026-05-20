# SCOPE.md

> Référence complète du scope file (`.onerc.yaml`) : grammaire, layering, lock file, commandes. Pour les permissions disponibles par service, voir le catalogue ou `one info <service>`.

## Vue d'ensemble

Le scope file est un fichier YAML versionné à la racine du projet qui déclare **ce qu'un agent (ou un dev) a le droit de faire sur ce projet via One CLI**. C'est l'unité centrale de gouvernance.

Trois propriétés non-négociables :

1. **Default deny strict.** Tout ce qui n'est pas explicitement autorisé est refusé.
2. **Lisible sans documentation.** Un dev qui ouvre le fichier doit comprendre ce qui est autorisé en 30 secondes.
3. **Versionné dans la repo.** Reviewable en PR, partagé par l'équipe, reproductible entre machines.

## Les trois fichiers

| Fichier | Rôle | Statut Git |
|---|---|---|
| `.onerc.yaml` | Source de vérité du projet | commité |
| `.onerc.local.yaml` | Overrides personnels | gitignored |
| `.onerc.lock` | Résolution catalogue figée | commité |

## Grammaire complète

### Squelette minimal

```yaml
version: 1

services:
  github:
    allow:
      - issues.*
      - pulls.read
  notion:
    allow:
      - pages.*
```

C'est suffisant pour la majorité des projets.

### Grammaire complète

```yaml
version: 1

project:
  name: kaampus-backend
  description: Backend API for Kaampus marketplace

defaults:
  github: work
  notion: kaampus
  stripe: test

profile: default                    # profile actif (si profiles définis)

profiles:
  default:
    defaults:
      stripe: test
    services:
      stripe:
        allow: [customers.*, subscriptions.*]

  production:
    extends: default
    defaults:
      stripe: live
    services:
      stripe:
        deny: [customers.delete, subscriptions.cancel]

services:
  github:
    account: work                   # surcharge defaults.github
    allow:
      - issues.*
      - pulls.read
      - pulls.review
      - repos.read
    deny:
      - issues.delete
      - repos.delete

  notion:
    account: kaampus
    allow:
      - pages.*
      - blocks.*
      - databases.read

  stripe:
    allow:
      - customers.read
      - subscriptions.read
```

### Champs détaillés

#### `version` (obligatoire)

Version du format. Toujours `1` actuellement. Permet d'évoluer le format dans le futur sans casser les projets existants.

#### `project` (optionnel)

Métadonnées humaines. Affichées dans `one info` quand on est dans ce projet. Utile pour distinguer plusieurs projets sur la même machine.

```yaml
project:
  name: kaampus-backend
  description: Backend API for the Kaampus marketplace
```

#### `defaults` (optionnel)

Compte par défaut par service. Si non spécifié, l'alias `default` est utilisé.

```yaml
defaults:
  github: work
  notion: kaampus
  stripe: test
```

Surchargeable par service (`services.<name>.account`) ou ad-hoc (`one --as perso github ...`).

#### `services` (obligatoire si on veut autoriser quoi que ce soit)

Permissions par service.

```yaml
services:
  github:
    account: work              # optionnel, sinon hérite de defaults.github
    allow: [...]               # liste de patterns autorisés
    deny: [...]                # liste de patterns refusés (override allow)
```

#### `profiles` (optionnel, v1.1+)

Profils nommés pour gérer plusieurs environnements (dev, staging, prod).

```yaml
profiles:
  default:
    defaults:
      stripe: test
    services:
      stripe:
        allow: [customers.*]

  production:
    extends: default
    defaults:
      stripe: live
    services:
      stripe:
        deny: [customers.delete]
```

Sélection :

- Par défaut : profil `default`
- Via env var : `ONE_PROFILE=production one ...`
- Via local : `profile: production` dans `.onerc.local.yaml`

L'héritage (`extends`) merge récursivement les champs du parent, le child surchargeant les conflits.

## Les globs

Une seule règle, minimaliste :

| Pattern | Match |
|---|---|
| `pages.read` | exactement cette permission |
| `pages.*` | tout ce qui commence par `pages.` (un seul niveau) |
| `*` | toutes les permissions du service |

**Interdit** :

- `**` (rejeté à la validation, trop ambigu)
- `pages.{read,write}` (pas d'expansion brace)
- `!pages.delete` (pas d'anti-pattern, utiliser `deny`)
- `pages.?` (pas de wildcard char)

Si tu veux exclure une permission précise, utilise `deny`. La grammaire reste lisible et prévisible.

### Précédence d'évaluation

L'ordre est figé et documenté :

```
1. deny exact         (deny: [pages.delete] vs perm pages.delete)
2. deny glob          (deny: [pages.*] vs perm pages.delete)
3. allow exact        (allow: [pages.read] vs perm pages.read)
4. allow glob         (allow: [pages.*] vs perm pages.read)
5. default deny       (rien ne match)
```

**Exemples** :

```yaml
allow: [pages.*]
deny: [pages.delete]
# → tout pages.X autorisé sauf pages.delete
```

```yaml
allow: [*]
deny: [databases.write]
# → tout autorisé sauf databases.write
```

```yaml
allow: [pages.delete]
deny: [pages.*]
# → pages.delete autorisé (deny glob bat allow exact ? NON, deny glob = 2, allow exact = 3)
# wait, refaisons: deny glob (pages.*) match pages.delete → DENY (étape 2 termine)
# → pages.delete REFUSÉ
```

L'exemple ci-dessus montre qu'**il faut comprendre la précédence**. La commande `one scope explain` (voir plus bas) trace l'évaluation pour débogger.

## Le layering : `.onerc.local.yaml`

Le `.onerc.local.yaml` est gitignored et permet à chaque dev d'ajuster pour sa machine. **Règle stricte** : il ne peut qu'**enlever ou changer de compte**, jamais ajouter une permission qui n'est pas dans le base.

### Ce qui est autorisé dans `.local`

```yaml
version: 1

# Changer de profil
profile: development

# Changer de compte par défaut
defaults:
  github: perso

# Restreindre encore
services:
  github:
    deny:
      - issues.delete
      - pulls.merge
```

### Ce qui est interdit dans `.local`

```yaml
# INTERDIT : ajouter une permission absente du base
services:
  github:
    allow:
      - repos.delete           # pas dans .onerc.yaml → ignoré + warning
  newservice:                  # service absent du base → ignoré + warning
    allow: [*]
```

### Algorithme de merge

```
result.profile = local.profile ?? base.profile

result.defaults = merge(base.defaults, local.defaults)
  # local override base par service

Pour chaque service présent dans base.services :
  result.allow = base.allow                              # local NE PEUT PAS étendre
  result.deny = base.deny ∪ local.deny                   # local PEUT restreindre
  result.account = local.account ?? base.account         # local PEUT changer

Pour chaque service présent uniquement dans local.services :
  → warning "service X in .onerc.local.yaml not in .onerc.yaml, ignored"
```

**Justification de la règle anti-extension** : si le local pouvait étendre, le scope file commité ne serait plus la source de vérité. Quelqu'un qui ouvre le repo ne verrait pas ce qui est *réellement* autorisé sur la machine du dev. Mauvais pour l'audit et la reproductibilité.

## Le lock file `.onerc.lock`

Fige les versions résolues du catalogue. Comme un `package-lock.json`.

### Format

```yaml
version: 1
generated_at: 2026-05-20T14:32:11Z

catalog:
  source: https://catalog.one-cli.dev
  index_sha256: 7a8b9c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b

services:
  github:
    version: 2.1.4
    tarball_sha256: 3f2a1b...
  notion:
    version: 1.4.0
    tarball_sha256: e9d8c7...
  stripe:
    version: 3.0.2
    tarball_sha256: 5c4b3a...
```

### Comportement strict

Si une version installée localement ne matche pas le lock, le binaire **refuse** d'exécuter l'action :

```
$ one notion pages.read --page_id ...
Error: notion version mismatch
  Lock:       1.4.0 (sha256:e9d8c7...)
  Installed:  1.5.2 (sha256:abc123...)

Run `one catalog update` to fetch the locked version, or
`one lock --update notion` to bump the lock.

Exit 1
```

Garantit la reproductibilité entre machines : tous les devs et CI utilisent les mêmes versions du catalogue.

### Commandes

```bash
one lock                              # génère le lock depuis les versions actuelles
one lock --update notion              # met à jour un service spécifique
one lock --update-all                 # met à jour tous les services
one lock --check                      # vérifie que le local matche (exit 0/1)
```

## Les commandes pour modifier le scope

L'agent et l'humain ne touchent **jamais** le fichier à la main, sauf en review de PR. La CLI propose les commandes :

```bash
one scope show                        # affiche le scope effectif (merge inclus) en JSON
one scope add <service> <perm>        # ajoute une permission à allow
one scope add <service> <perm> --deny # ajoute à deny
one scope remove <service> <perm>     # retire une permission
one scope remove <service> <perm> --deny  # retire de deny
one scope use <service> --as <alias>  # change le compte par défaut
one scope check                       # valide la cohérence (schéma + catalogue)
one scope explain <service> <perm>    # trace l'évaluation, montre pourquoi un perm est autorisé/refusé
```

### `one scope show`

Affiche le scope effectif après merge base + local + profile :

```json
{
  "version": 1,
  "project": "kaampus-backend",
  "profile": "default",
  "services": {
    "github": {
      "account": "perso",
      "allow": ["issues.*", "pulls.read"],
      "deny": ["issues.delete"]
    },
    "notion": {
      "account": "kaampus",
      "allow": ["pages.*", "blocks.*"]
    }
  }
}
```

Utile pour les agents qui veulent introspecter avant d'agir.

### `one scope check`

Validation exhaustive du fichier :

```
$ one scope check
✓ Schema valid (.onerc.yaml v1)
✓ Schema valid (.onerc.local.yaml v1)
✓ Merge has no conflicts
✓ All services exist in catalog
✓ All permissions exist in their services
✓ Lock file matches installed catalog
⚠ Service 'discord' is in scope but not authenticated
    → Run `one login discord` to authenticate
✗ Permission 'github.actions.read' does not exist
    → Did you mean: 'actions.list'?

Errors: 1   Warnings: 1
```

Exit codes :

- 0 : aucun erreur ni warning
- 1 : warnings uniquement
- 2 : erreurs présentes

Idéal en pre-commit hook ou CI.

### `one scope explain`

Trace l'évaluation, ligne par ligne :

```
$ one scope explain github issues.delete
Permission: github.issues.delete
Result: DENIED

Evaluated in order:
  1. .onerc.local.yaml:deny - matched 'issues.delete' (exact) → DENY (final)

Rules not evaluated (would have applied):
  - .onerc.yaml:allow - 'issues.*' would have matched
  - .onerc.yaml:deny  - 'issues.delete' would have matched (redundant)

To allow:
  Remove the deny rule from .onerc.local.yaml:
    one scope remove github issues.delete --deny --local
```

C'est ce qui rend la grammaire utilisable au-delà du trivial. Quand un agent dit "j'ai pas la permission", le dev fait `one scope explain` et voit exactement pourquoi en 5 secondes.

## Exemple : workflow type sur un nouveau projet

### Première installation

```bash
cd mon-projet
one init                              # crée .onerc.yaml vide
```

`.onerc.yaml` est créé minimal :

```yaml
version: 1
services: {}
```

### Login aux services nécessaires

```bash
one login github                      # OAuth, alias "default"
one login notion                      # OAuth, alias prompted
```

### Définir le scope

```bash
one scope add github issues.*
one scope add github pulls.read
one scope add notion pages.*
one scope add notion blocks.*
```

`.onerc.yaml` devient :

```yaml
version: 1
services:
  github:
    allow:
      - issues.*
      - pulls.read
  notion:
    allow:
      - pages.*
      - blocks.*
```

### Verrouiller

```bash
one lock
git add .onerc.yaml .onerc.lock
git commit -m "init: add One CLI scope"
```

### Le collègue clone

```bash
git clone ...
cd mon-projet
one catalog update                    # fetch les versions lockées
one login github                      # crée son propre compte
one login notion
# Le scope est déjà défini, pas besoin de le redéfinir
```

## JSON Schema (extrait formel)

Référence pour la validation. Le schéma complet est dans `pkg/catalog/schema/onerc-v1.json`.

```yaml
$schema: https://json-schema.org/draft/2020-12/schema
type: object
required: [version]
additionalProperties: false
properties:
  version:
    const: 1
  project:
    type: object
    additionalProperties: false
    properties:
      name: { type: string }
      description: { type: string }
  profile:
    type: string
  profiles:
    type: object
    additionalProperties:
      $ref: "#/$defs/profile"
  defaults:
    type: object
    additionalProperties:
      type: string
  services:
    type: object
    additionalProperties:
      $ref: "#/$defs/serviceScope"

$defs:
  profile:
    type: object
    properties:
      extends:
        type: string
      defaults: { $ref: "#/properties/defaults" }
      services: { $ref: "#/properties/services" }

  serviceScope:
    type: object
    additionalProperties: false
    properties:
      account:
        type: string
        pattern: "^[a-z][a-z0-9_-]*$"
      allow:
        type: array
        items:
          $ref: "#/$defs/permPattern"
      deny:
        type: array
        items:
          $ref: "#/$defs/permPattern"

  permPattern:
    type: string
    pattern: "^[a-z][a-z0-9_]*(\\.[a-z0-9_*]+)*$"   # interdit `**` et `?`
```

## Recettes courantes

### Agent en read-only

```yaml
version: 1
services:
  github:
    allow:
      - "*.read"
      - "*.list"
      - "*.search"
      - search
```

### Limiter destructif sur tous les services

```yaml
version: 1
services:
  github:
    allow: ["*"]
    deny:
      - repos.delete
      - issues.delete
  notion:
    allow: ["*"]
    deny:
      - pages.archive
      - databases.delete
```

### Compte de test en dev, prod en CI

```yaml
# .onerc.yaml
version: 1
profile: default
profiles:
  default:
    defaults:
      stripe: test
  production:
    extends: default
    defaults:
      stripe: live
services:
  stripe:
    allow: [customers.*, charges.*]
```

```yaml
# .onerc.local.yaml (chez chaque dev)
# Rien, le défaut "test" est OK
```

```yaml
# Sur la CI prod : env var ONE_PROFILE=production
```

### Limiter agent IA strictement

Cas d'usage : un projet où un agent IA tourne en autonomie sur du code de production. On veut être très strict.

```yaml
version: 1
services:
  github:
    allow:
      - issues.read
      - issues.create
      - issues.comment
      - pulls.read
      - pulls.review            # autorisé à reviewer mais pas merger
    deny:
      - "*"                     # explicite, en bottom
  # pas de notion, pas de stripe, pas d'autre service
```

Le `deny: ["*"]` en bas est redondant à cause du default deny, mais rend explicite l'intent.

## Anti-patterns à refuser en code review

### Allow `["*"]` sans deny

```yaml
# DANGEREUX
services:
  github:
    allow: ["*"]
```

Tout est autorisé. C'est rarement ce qu'on veut. Soit on liste les permissions explicitement, soit on `allow: ["*"]` + `deny:` des destructeurs.

### Permissions sans namespace

```yaml
# MAUVAIS
services:
  github:
    allow:
      - "read"                  # de quoi ?
      - "*"                     # trop large
      - "issue.delete"          # singulier au lieu de pluriel
```

Respecter la convention `<resource>.<verb>` exactement comme déclaré par le service.

### Conflits non résolus

```yaml
# CONFUS
services:
  github:
    allow:
      - issues.*
      - issues.delete
    deny:
      - issues.delete
```

`issues.*` couvre déjà `issues.delete`, ajouter `issues.delete` en allow est redondant. Avec le deny qui suit, c'est ambigu pour le lecteur. `one scope check` warn dans ce cas.

### Mélanger profiles et top-level

```yaml
# CONFUS
services:                       # définit au top-level
  github:
    allow: [issues.read]
profiles:
  default:
    services:                   # ET dans le profile default
      github:
        allow: [issues.write]
```

Choisis l'un ou l'autre. Soit pas de profile, soit tout dans des profiles.

---

*Pour proposer une évolution de la grammaire du scope file, ouvrir un RFC dans `one-cli/rfcs`. Changement majeur = bump `version`. Le binaire doit supporter les versions précédentes pendant au moins 12 mois.*
