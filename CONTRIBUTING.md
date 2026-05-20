# CONTRIBUTING.md

> Guide pour contribuer au projet One CLI. Si tu contribues au catalogue (ajouter un service), va directement à [CATALOG.md](./CATALOG.md). Ce document couvre les contributions au **binaire Go**.

## Premier setup

### Pré-requis

- **Go 1.23+** : `go version`
- **Git** : `git --version`
- **make** : ou rebuild via `go build` directement
- **golangci-lint** : pour le linting (`brew install golangci-lint`)
- **tinygo** : seulement si tu touches aux tests handlers Go (`brew install tinygo`)
- **bun** ou **node** : seulement si tu touches aux tests handlers TypeScript

Optionnel :

- **wasmtime** ou **wasmer** CLI : pour debugger des modules WASM
- **age** : pour tester le vault chiffré

### Cloner et builder

```bash
git clone https://github.com/one-cli/one
cd one
make build                            # produit ./bin/one
./bin/one --version                   # vérifie
```

Ou sans make :

```bash
go build -o bin/one ./cmd/one
```

### Lancer les tests

```bash
make test                             # unit + integration + contract
make test-security                    # tests sécurité (long, ~30s)
make test-e2e                         # tests E2E (lent, ~2min)
make bench                            # benchmarks
make lint                             # golangci-lint
```

Ou sans make :

```bash
go test ./...
go test -tags=security ./tests/security/...
go test -tags=e2e ./tests/e2e/...
go test -bench=. ./...
golangci-lint run
```

### Faire tourner en dev

```bash
# Build et utilise en local
make install                          # met le binaire dans $GOBIN
which one                             # vérifie

# Ou directement
go run ./cmd/one -- --version
```

## Workflow de contribution

### 1. Trouver une issue

Trois bonnes manières de commencer :

- **Issues taggées `good-first-issue`** : conçues pour découvrir le code
- **Issues taggées `help-wanted`** : besoin réel d'aide, scope moyen
- **RFC ouvertes** : feature en discussion, ton avis bienvenu

Avant de coder, **commente l'issue pour signaler ton intérêt**. Évite de dupliquer le travail.

### 2. Fork et branch

```bash
# Fork via GitHub UI, puis :
git clone https://github.com/<toi>/one
cd one
git remote add upstream https://github.com/one-cli/one
git checkout -b feat/add-bitbucket-provider
```

**Nom de branche** : `<type>/<courte-description>`. Types : `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `chore`.

### 3. Coder

Voir les sections plus bas pour le code style et les conventions.

### 4. Tester

```bash
make test
make lint
```

CI fait tourner les deux. Si ça passe localement, ça passe en CI (sauf cas cross-platform).

### 5. Commit

Format : **conventional commits**.

```
feat(auth): add bitbucket OAuth provider

Implements the OAuth 2.0 user-flow for Bitbucket Cloud, including
PKCE and refresh token rotation. Tested via fakes against the
ports.AuthProvider contract.

Closes #142
```

Types acceptés : `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `chore`, `ci`, `build`. Scope optionnel mais recommandé : `auth`, `catalog`, `vault`, `runtime`, `cli`, `core`, etc.

**Pas de commits "WIP"** dans la PR finale (rebase pour squash si tu en as).

### 6. Push et PR

```bash
git push origin feat/add-bitbucket-provider
```

Ouvre la PR via GitHub UI. Le template pré-rempli demande :

- Description du changement
- Issue liée
- Tests ajoutés
- Breaking changes (si applicable)
- Screenshots (si UI/TTY changée)

### 7. Review

Un maintainer review dans les 7 jours. Possibles itérations :

- Demandes de changement → push sur la même branche, pas besoin de rouvrir la PR
- Discussions sur le design → on tranche dans la PR, ou on ouvre une RFC si trop large

### 8. Merge

Merge en **squash and merge** par défaut. Ton ou tes commits deviennent un seul commit sur main. Message final édité par le maintainer pour respecter conventional commits.

Une fois mergé, ton nom apparaît dans `CHANGELOG.md` de la prochaine release.

## Code style

### Go : conventions standards

On suit [Effective Go](https://go.dev/doc/effective_go) et [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments). Quelques points spécifiques :

#### Naming

- Types exportés : `PascalCase` (`Credential`, `ServiceID`)
- Types non exportés : `camelCase` (`vaultState`)
- Fonctions de constructor : `New<Type>(...)` (`NewScope`, `NewCachedCatalog`)
- Interfaces : nom sans `I` prefix, plutôt avec un suffixe descriptif (`Catalog`, `AuthProvider`, pas `ICatalog`)
- Variables courtes pour scope court (`ctx`, `err`, `i`), explicites pour scope long (`servicesByName`)

#### Imports

Ordonnés en 3 blocs séparés par une ligne vide :

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/zalando/go-keyring"
    "golang.org/x/oauth2"

    "one/internal/core"
    "one/internal/ports"
)
```

Stdlib | externe | local.

#### Errors

- Wrap avec contexte : `fmt.Errorf("fetch credential: %w", err)`
- Pas de "failed to" devant : `"fetch credential"` suffit
- Pas de capitalisation du premier caractère du message
- Pas de point final
- Types d'erreurs sentinelles définis dans `core/errors.go`, jamais dispersés

#### Comments

- Toutes les fonctions/types exportés ont un comment doc Go
- Format : `// FuncName does X. Returns Y when Z.`
- Pas de commentaires dans des cas évidents (`// increment i`)
- Préfère des noms clairs à des commentaires explicatifs

#### Context

- Toujours premier argument : `func Foo(ctx context.Context, ...)`
- Jamais stocké dans une struct
- Propagé partout, même quand "pas utilisé maintenant"

#### Concurrence

- Préfère les channels aux mutexes quand possible
- Si mutex, doc le invariant qu'il protège
- Toujours `defer mu.Unlock()` après `mu.Lock()`
- Pas de `time.Sleep` pour synchroniser ; utiliser channels ou waitgroups

### Layout du repo

Voir [ARCHITECTURE.md](./ARCHITECTURE.md#layout-du-repo) pour le layout complet. Règle d'or :

- **Domaine** dans `internal/core/` : zéro dépendance externe
- **Ports** dans `internal/ports/` : interfaces uniquement
- **Adapters** dans `internal/adapters/<port>/` : implémentations
- **Use cases** dans `internal/app/` : orchestration
- **CLI** dans `internal/cli/` : adapter UI (cobra)

Si ta PR met du HTTP dans `core/`, refus immédiat.

### Patterns récurrents

#### Constructeurs

Constructeurs explicites, pas de "init magic" :

```go
// BON
func NewExecuteAction(
    catalog ports.Catalog,
    vault ports.Vault,
    runtime ports.Runtime,
    log ports.Logger,
    clock ports.Clock,
) *ExecuteAction {
    return &ExecuteAction{...}
}

// PAS BIEN
type ExecuteAction struct { ... }
func (e *ExecuteAction) Init(...) { ... }
```

#### Options patterns pour les structs complexes

```go
type Server struct { ... }

type ServerOption func(*Server)

func WithTimeout(d time.Duration) ServerOption {
    return func(s *Server) { s.timeout = d }
}

func NewServer(opts ...ServerOption) *Server {
    s := &Server{timeout: 30 * time.Second}  // défauts
    for _, opt := range opts { opt(s) }
    return s
}
```

#### Interface segregation

Une interface = une responsabilité. Si tu as besoin de 3 méthodes, fais peut-être 2-3 interfaces et compose-les.

```go
// BON
type CatalogReader interface {
    GetService(ctx, ServiceID) (*Service, error)
    ListServices(ctx) ([]Service, error)
}

type CatalogWriter interface {
    UpdateService(ctx, Service) error
}

type Catalog interface {
    CatalogReader
    CatalogWriter
}
```

## Que faire et ne pas faire

### À faire

- **Écrire des tests pour chaque changement.** Niveau approprié : unit pour la logique métier, contract pour les adapters, integration pour les use cases.
- **Documenter les fonctions exportées.** Si elles le sont, c'est qu'on les utilise ; doc + doc test idéalement.
- **Update les docs en parallèle du code.** Si tu changes un comportement décrit dans CLI.md, update CLI.md dans la même PR.
- **Bénéficier des fakes existants.** `internal/testing/fake/` a déjà ce dont tu as besoin pour 80% des cas.
- **Préférer ajouter à modifier.** Si tu peux ajouter un adapter sans toucher au domaine, c'est mieux.
- **Discuter avant de coder pour les gros changements.** Ouvre une issue ou une RFC.

### À ne pas faire

- **Pas de PR géantes.** Une PR = un changement focused. Si tu touches 30 fichiers, découpe.
- **Pas de breaking changes silencieux.** Si tu casses une API publique, mentionne-le en titre de PR et propose un path de migration.
- **Pas de "fix unrelated typo in passing".** Si tu vois une typo, ouvre une PR dédiée. Ça facilite la review.
- **Pas de dépendance ajoutée sans discussion.** Chaque nouvelle dépendance externe est un coût. Si elle se justifie, mentionne-le explicitement dans la PR.
- **Pas de magie.** Pas de génération de code custom, pas de réflection abusive, pas de panic dans le code applicatif.

## Ajouter une nouvelle dépendance externe

Critères avant d'ajouter un package :

1. **Pas trouvable en stdlib** : la stdlib Go est riche, vérifie d'abord.
2. **Maintenu activement** : dernier commit < 6 mois, ou si abandonné, justifie pourquoi c'est OK.
3. **License compatible** : MIT, Apache 2.0, BSD-3, MPL-2.0. Pas GPL, pas custom.
4. **Pas de dépendances transitives explosives** : check `go mod graph`.
5. **Surface d'API minimale** : si tu n'utilises qu'une fonction sur 50, considère l'écrire toi-même.

Liste indicative des dépendances acceptables au moment de l'écriture :

- `github.com/spf13/cobra` : CLI (déjà en place)
- `github.com/spf13/viper` : config (déjà en place)
- `github.com/zalando/go-keyring` : keychain (déjà en place)
- `github.com/tetratelabs/wazero` : WASM runtime (déjà en place)
- `filippo.io/age` : vault chiffré (déjà en place)
- `github.com/goccy/go-yaml` : YAML parser (déjà en place)
- `github.com/santhosh-tekuri/jsonschema/v6` : JSON Schema (déjà en place)
- `golang.org/x/oauth2` : OAuth helpers (déjà en place)
- `github.com/charmbracelet/lipgloss` : styling TTY (déjà en place)
- `github.com/charmbracelet/bubbletea` : TUI interactif (déjà en place pour les flows interactifs)

Si tu veux en ajouter une autre, motive dans la PR.

## RFC : pour les gros changements

Si tu veux changer :

- L'API du `service.yaml`
- La grammaire du scope file
- Le contrat des host functions WASM
- Une convention de naming structurante
- Un mécanisme de sécurité

Ouvre une RFC dans le repo `one-cli/rfcs`. Format dans `rfcs/0000-template.md`.

Processus :

1. Fork `rfcs`, copie le template, rename `0000-` en un nombre libre.
2. Édite, push, ouvre une PR.
3. Discussion ouverte 14 jours minimum.
4. Décision : accept, defer, reject. Documentée par le mainteneur.
5. Si accepté, l'implémentation suit normalement via une PR sur le repo concerné.

## Sécurité

Si tu trouves une vulnérabilité, **n'ouvre pas une issue publique**. Voir [SECURITY.md > Disclosure policy](./SECURITY.md#disclosure-policy).

## License

En ouvrant une PR, tu acceptes que ton code soit licencié sous la même license que le projet (Apache 2.0 ou MIT, voir LICENSE). Pas de CLA, signature implicite via le merge.

## Communauté

- **GitHub Discussions** : questions générales, design ideas, retours d'usage
- **GitHub Issues** : bugs et features concrètes
- **Discord** (à venir) : chat en temps réel

Code of conduct : standard, [Contributor Covenant 2.1](https://www.contributor-covenant.org/version/2/1/code_of_conduct/). En substance : sois respectueux, prudent dans les désaccords, focalisé sur le projet.

## Récompenses

Le projet est open source, pas de paiement direct. Mais :

- Ton nom dans le `CHANGELOG.md` et `CONTRIBUTORS.md`
- Reconnaissance publique sur les annonces de release
- Mentorship : si tu débutes en Go ou en open source, les mainteneurs prennent le temps de t'aider en review
- Pour les contributeurs réguliers : commit access possible après 3-5 PRs de qualité

## Premier issue suggérée

Si tu veux contribuer mais ne sais pas par où commencer :

1. **Lis `DESIGN.md` et `ARCHITECTURE.md`** : 20 minutes pour la vue d'ensemble.
2. **Run `make build && make test`** : assure-toi que ton setup marche.
3. **Pick une issue `good-first-issue`** : généralement ajout d'un test, fix d'un message d'erreur, amélioration d'un guide.
4. **Suis le workflow ci-dessus** : pas de surprise.

Bienvenue !

---

*Maintenu par [@elydelva](https://github.com/elydelva) et la communauté. Toute proposition d'amélioration de ce document est bienvenue via PR.*
