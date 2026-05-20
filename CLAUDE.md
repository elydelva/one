# CLAUDE.md

> Instructions opérationnelles pour les agents IA (Claude Code, Cursor, Aider) qui contribuent au binaire **One CLI**. Lis ce fichier en premier. Pointe vers les autres docs pour les détails. Si tu as 5 minutes avant de coder, c'est ici qu'elles vont.

## Le projet en 30 secondes

**One CLI** est un binaire Go qui unifie l'accès aux APIs tierces pour les agents IA. Trois piliers structurants :

1. Un **vault local** multi-comptes chiffré (jamais en SaaS)
2. Un **scope file** `.onerc.yaml` versionné qui rend explicite ce qu'un agent peut faire
3. Un **catalogue** open source de services, distribué via Git + CDN

Quatre verbes côté agent : `one <service> <action>`, `one capabilities`, `one info`, `one can`.

**Statut actuel** : pre-alpha. Voir `ROADMAP.md` pour la phase en cours.

## Que faire avant de coder

### 1. Lire les bons docs

Selon ta tâche, lis dans cet ordre :

| Tâche | Docs à lire |
|---|---|
| Comprendre le projet | DESIGN.md (10 min) |
| Toucher au binaire Go | ARCHITECTURE.md (20 min) puis TESTING.md (15 min) |
| Ajouter un service au catalogue | CATALOG.md (15 min) |
| Écrire un handler WASM | HANDLERS.md (15 min) puis SECURITY.md (sandbox section) |
| Modifier le scope file format | SCOPE.md (10 min) puis ouvrir un RFC avant |
| Toucher à l'auth | AUTH.md (15 min) puis SECURITY.md |
| Modifier la CLI | CLI.md (10 min) |

**Ne lis pas tout systématiquement.** Cible le doc qui matche ta tâche.

### 2. Setup local

```bash
go version                    # doit être 1.23+
make build                    # produit ./bin/one
make test                     # doit passer (sinon stop, signale)
./bin/one --version
```

Si `make test` échoue sur main, ne commence pas ta task. Signale-le, c'est probablement un fix qui doit passer d'abord.

### 3. Identifier le périmètre

Avant d'écrire du code, **dis clairement quel use case tu touches** et **quel adapter ou port**. Si tu ne peux pas localiser ton changement précisément dans le layout d'ARCHITECTURE.md, tu n'es pas prêt à coder.

## Architecture en image

```
cmd/one/main.go                  composition root (≤100 lignes)
    │
    ▼
internal/cli/                    cobra commands, parsing, exit codes
    │
    ▼
internal/app/                    use cases (ExecuteAction, Login, ShowInfo, ...)
    │
    ▼
internal/core/  ◄── internal/ports/  ◄──  internal/adapters/<port>/
(domaine pur,        (interfaces)         (implémentations concrètes)
 zéro deps)
```

**La règle d'or** : `internal/core/` n'importe que la stdlib Go. Aucune exception. Si ton code dans `core/` a besoin de YAML, HTTP, crypto externe, c'est qu'il appartient ailleurs.

## Workflows types

### Tu ajoutes une nouvelle commande CLI

1. Définir le use case dans `internal/app/<usecase>.go`
2. Définir la commande cobra dans `internal/cli/<command>.go`
3. Wire dans `cmd/one/main.go` (composition root)
4. Tests : unit du use case avec fakes, integration via `internal/app/<usecase>_test.go`, et un cas E2E si critique
5. Update `CLI.md` avec la nouvelle commande
6. Si la commande est accessible aux agents : update le `one skill`

### Tu ajoutes un nouvel adapter (genre une 2ème implémentation de Vault)

1. Crée le fichier dans `internal/adapters/<port>/<impl>.go`
2. Implémente l'interface du port
3. **Run les contract tests** : `portstest.Run<Port>Tests(t, "<Impl>", factory)`
4. Wire dans la composition root si tu veux qu'il soit utilisé
5. Documente brièvement dans le doc concerné (AUTH, ou autre)

Les contract tests sont l'élément clé. Si tu ne les fais pas tourner, tu n'as pas fini.

### Tu ajoutes un nouveau port (besoin domaine nouveau)

**Stop.** C'est probablement le signe qu'il faut une discussion avant. Ouvre une issue, propose le port et son interface, attends un feedback. Un nouveau port = nouvelle responsabilité dans le domaine, ça mérite réflexion.

### Tu fix un bug

1. Reproduit le bug avec un test qui fail
2. Fix le code
3. Vérifie que le test pass
4. Vérifie que la suite complète passe : `make test`
5. Si le bug venait d'un manque dans la doc, update la doc

### Tu changes un comportement décrit dans une doc

**Update la doc dans la même PR.** Pas dans une PR séparée "plus tard". Une doc qui ment est pire qu'une doc absente.

## Règles non-négociables

Ces règles existent pour des raisons documentées dans DESIGN.md. Ne les enfreins pas sans ouvrir une RFC d'abord.

1. **Default deny strict.** Aucune permission, aucun accès, aucune URL n'est implicite.
2. **Pas d'I/O dans `internal/core/`.** Le domaine est pur.
3. **Tout secret est typé `core.Secret`.** Jamais de string en clair pour un token.
4. **Pas de panic en code applicatif.** Retourne une erreur typée.
5. **Pas de framework DI.** Composition explicite dans main.go.
6. **Pas de génération de code custom** (au-delà de `go generate` pour structures depuis JSON Schema).
7. **Allowlist URL stricte pour les handlers WASM.** Pas d'évasion possible.
8. **Tests cross-platform pour tout ce qui touche au keychain, aux paths, ou aux locks.**

Si une PR enfreint une de ces règles, elle sera rejetée même si le code est élégant.

## Anti-patterns courants

À éviter, vu fréquemment dans les premières contributions :

### Stocker un client HTTP global dans une variable de package
```go
var defaultClient = &http.Client{Timeout: 30 * time.Second}
```
**Non.** Toute config passe par les constructeurs. Pas de globals mutables.

### Logger des credentials
```go
log.Info("got token", "token", cred.AccessToken.Reveal())
```
**Non.** Le type `Secret` retourne `[REDACTED]` par défaut. `Reveal()` uniquement au point d'injection HTTP.

### Mocks avec 30 expectations
```go
mockVault.EXPECT().Fetch(gomock.Any(), gomock.Eq(ref1)).Return(...)
mockVault.EXPECT().Store(gomock.Any(), gomock.Any(), ...)
// ... 28 lignes
```
**Non.** Utilise les fakes dans `internal/testing/fake/`. Voir TESTING.md.

### Importer une lib externe dans `core/`
```go
// internal/core/credential.go
import "filippo.io/age"
```
**Non.** Le domaine ne sait pas qui chiffre. Mets ça dans `adapters/vault/age.go`.

### Faire une PR qui touche 20 fichiers
**Non.** Découpe. Une PR a un objectif précis et atteignable, idéalement en moins de 10 fichiers modifiés.

### Ajouter une dépendance Go sans discussion
**Non.** Chaque dépendance est un coût. Voir CONTRIBUTING.md pour les critères.

### `time.Sleep` pour synchroniser un test
```go
go startBackground()
time.Sleep(100 * time.Millisecond)
```
**Non.** Flaky en CI. Utilise channels ou `t.Cleanup`.

### Tests dépendant de l'ordre d'exécution
**Non.** Chaque test indépendant, setup + cleanup propres.

### Réponse evasive sur un trade-off
Si tu te dis "je sais pas, je fais les deux pour être safe" et que tu codes les deux options, **non**. Tranche. Documente le choix. Si tu hésites sincèrement, demande au mainteneur via l'issue avant de coder.

## Convention de commits et PRs

**Conventional commits.** Format :

```
feat(auth): add bitbucket OAuth provider

Implements OAuth 2.0 user-flow with PKCE. Tested against the
ports.AuthProvider contract.

Closes #142
```

Types : `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `chore`, `ci`, `build`.

Scopes : `core`, `auth`, `vault`, `catalog`, `runtime`, `cli`, `scope`, `skill`, etc.

**Une PR = un objectif.** Pas de "PR fourre-tout" mélangeant un fix, un refactor, et une feature.

**Titre de PR descriptif et impératif** : "Add bitbucket OAuth provider", pas "Bitbucket stuff".

**Description de PR** : ce qui change, pourquoi, ce qui n'est pas couvert, breaking changes éventuels.

## Où trouver quoi

Quelques pointers rapides pour ne pas perdre du temps à chercher :

| Question | Réponse |
|---|---|
| Comment créer une nouvelle erreur typée ? | `internal/core/errors.go` |
| Comment ajouter un exit code ? | `internal/cli/exit.go`, mapping depuis erreur typée |
| Format du `service.yaml` ? | CATALOG.md + `pkg/catalog/schema/v1.json` |
| Comment écrire un fake ? | `internal/testing/fake/` (modèles existants) |
| Comment écrire un contract test ? | `internal/testing/portstest/` (modèles existants) |
| Comment debugger un handler WASM ? | HANDLERS.md, section tests + `ONE_DEBUG=1` |
| Où sont stockées les credentials en local ? | Keychain natif (macOS/Linux/Windows), voir AUTH.md |
| Comment ajouter une fixture de catalog pour tests ? | `internal/testing/fixture/catalog/v1-<name>/` |
| Comment générer un benchmark ? | `make bench`, voir TESTING.md section perf |
| Comment release une nouvelle version ? | `make release`, tag SemVer, CI fait le reste |

## Comment toi (agent) tu utilises One CLI

Quand tu travailles sur le repo, **tu as `one` installé**. Tu peux et dois t'en servir au lieu de coder à la main des appels API.

```bash
one capabilities --scope-only      # qu'est-ce que je peux faire ?
one info <service>                 # comment utiliser un service
one can <service> <action>         # vérifie avant de tenter
one <service> <action> ...         # exécute
```

Si tu te demandes comment interagir avec un service tiers (GitHub, Linear, etc.), **utilise `one`** plutôt que de générer du `curl` ou de demander à l'utilisateur ses credentials.

Si une action te renvoie `setup_required`, l'output JSON contient `install.command`. Affiche-le à l'utilisateur, attends sa confirmation, puis poursuis.

Si une action te renvoie `not_in_scope` (exit 3), propose `one scope add <service> <perm>` à l'utilisateur. **Ne contourne jamais le scope file en passant par autre chose.**

## Style de réponse à l'utilisateur

L'utilisateur principal (Ely) préfère :

- **Direct et pragmatique.** Pas de "Great question!", pas de "Let me think about this carefully". Va à l'essentiel.
- **Anti-théâtral.** Pas de langage pompeux. Pas de "embark on this journey", pas de "let's dive deep".
- **Push back constructif accepté.** Si tu vois un trade-off non discuté, tu le signales avec ton avis.
- **Pas de tirets cadratins.** Utilise `:`, `(`, ou phrases courtes.
- **Français** (sauf code, commits, et noms d'entités techniques).
- **Concis.** Si une réponse de 5 lignes suffit, ne fais pas 30 lignes.

Quand tu proposes un changement non trivial, **anticipe les objections** : si tu vois 2 options viables, dis lesquelles et tranche en argumentant brièvement.

## Trois choses à toujours faire

1. **Lis le doc pertinent avant de coder.** 15 minutes économisées en lecture évitent 2h de débogage.
2. **Run `make test && make lint` avant de proposer un commit.** Pas après "j'ai fini", pendant.
3. **Pose des questions quand c'est ambigu.** Mieux vaut une question de 30 secondes qu'une PR refusée après 3h de travail.

## Trois choses à ne jamais faire

1. **Ne modifie jamais `internal/core/` pour ajouter une dépendance externe.** Refactor en port + adapter d'abord.
2. **Ne contourne jamais une règle de sécurité** documentée dans SECURITY.md, même temporairement, même pour "débug".
3. **Ne refactor pas hors de ton scope.** Si tu vois 4 trucs à améliorer en passant, **note-les dans une issue**, mais ne les fais pas dans la PR courante. Le focus est plus précieux que la complétude.

## Cas particuliers à connaître

### Tu travailles sur le repo `one` mais une question concerne le catalogue

Le catalogue (services, handlers WASM, install guides) est dans un **repo séparé** : `one-cli/catalog`. Si la question concerne le format `service.yaml` ou un handler concret, redirige vers ce repo. Le binaire ne contient pas la définition des services.

### Tu vois une décision dans le code qui semble contredire la doc

Demande avant d'agir. Soit la doc ment (à fix), soit le code a un bug (à fix), soit il y a un contexte que tu ignores. Les trois cas méritent une issue avant un changement.

### Tu hésites entre faire un truc "proprement" et "rapidement"

Par défaut, fais proprement. Le projet est jeune, c'est l'inverse d'une codebase legacy : chaque shortcut pris maintenant coûtera 10x plus cher dans 6 mois. Si l'utilisateur veut un quick fix, il te le dira explicitement.

### Tu trouves un bug critique de sécurité

**Ne l'ouvre pas en issue publique.** Voir SECURITY.md > Disclosure policy. Email direct au mainteneur.

## Ressources

- **Docs** dans `/docs` : DESIGN.md, ARCHITECTURE.md, CATALOG.md, HANDLERS.md, SCOPE.md, AUTH.md, SECURITY.md, TESTING.md, CLI.md, CONTRIBUTING.md, ROADMAP.md
- **Code** : `cmd/`, `internal/`, `pkg/`
- **Tests** : `*_test.go` colocalisés, `tests/e2e/`, `tests/security/`
- **Fixtures** : `internal/testing/fixture/`
- **Issues GitHub** : tagging `good-first-issue`, `help-wanted`, `bug`, `feat`
- **RFC** : repo séparé `one-cli/rfcs` pour les gros changements

---

*Si quelque chose dans ce fichier te semble obsolète ou contradictoire avec le reste de la doc, c'est probablement vrai. Signale-le.*