# DESIGN.md

> Le **north star** du projet. Ce document décrit *pourquoi* One CLI existe, *quoi* il fait, et *quels principes* tranchent les arbitrages quand un choix se présente. Si un futur contributeur ne lit qu'un seul document, c'est celui-ci.

## Pitch en 30 secondes

**One CLI est une couche d'abstraction unifiée entre les agents IA (ou les humains) et les services tiers.** Au lieu d'apprendre N SDKs et de gérer N modèles d'authentification, l'agent appelle `one <service> <action>`. Le binaire gère l'auth, applique des permissions définies par projet, exécute l'action, et retourne du JSON propre. Si l'action sort de son périmètre, il renvoie un guide actionnable.

Trois différenciateurs durables :

1. **Un vault local multi-comptes** chiffré dans le keychain de l'OS, jamais en SaaS.
2. **Un scope file versionné** dans la repo, qui rend explicite et reviewable ce qu'un agent a le droit de faire sur ce projet précis.
3. **Des install guides first-class** qui matérialisent honnêtement les frontières entre ce qui est automatisable et ce qui requiert un humain.

## Le problème

Les agents IA ont besoin d'agir dans le monde réel. Aujourd'hui ils le font de trois manières, toutes imparfaites :

**1. Via des SDKs spécifiques (OpenAI tools, Anthropic tools).** Chaque agent re-implémente l'appel à GitHub, Stripe, Notion. Pas de gouvernance, pas de réutilisation, credentials gérées ad hoc par chaque dev (souvent dans des env vars éparpillées).

**2. Via MCP (Model Context Protocol).** Standardisation prometteuse, mais orientée serveur, sans modèle de scope par projet, sans vault unifié multi-comptes, sans gestion des opérations humaines requises. Et MCP a tendance à pousser vers du "tout-en-conteneur" alors que beaucoup de devs préfèrent un binaire local sous leur contrôle direct.

**3. En écrivant du code à chaque fois.** L'agent génère du `curl` ou du fetch, l'humain lui passe ses tokens. Fragile, dangereux, non-auditable.

**Ce qui manque, et que personne ne fait bien aujourd'hui** :

- Un binaire local, unique, qui parle à N services avec un modèle unifié.
- Un fichier versionné dans la repo qui dit explicitement "sur ce projet, l'agent peut faire X mais pas Y".
- Un mécanisme propre pour les setups qui ne peuvent pas être automatisés (genre "partage cette page Notion avec l'intégration"), au lieu de boucles d'erreurs incompréhensibles.
- Un catalogue de services maintenu en open source, à la manière de Homebrew formulae, où la communauté ajoute ses intégrations.

One CLI est l'intersection de ces quatre besoins.

## Le positionnement vs MCP

MCP est l'alternative la plus proche conceptuellement. La question "pourquoi pas juste utiliser MCP ?" doit avoir une réponse claire.

| Dimension | MCP | One CLI |
|---|---|---|
| Forme | Serveur HTTP/stdio par service | Un seul binaire local |
| Auth | Géré par chaque serveur, ad hoc | Vault unifié, multi-comptes, multi-providers |
| Scope par projet | Pas d'équivalent | Fichier `.onerc.yaml` versionné |
| Setup humain | Pas couvert nativement | Verbe `install` first-class |
| Distribution | Chaque server est un package séparé | Catalogue centralisé reviewable |
| Audit | Logs par serveur | Audit log unifié `one trace` |
| Modèle mental | "L'agent parle à des serveurs" | "L'agent utilise un outil local" |

**One CLI n'est pas anti-MCP.** Les deux peuvent coexister. À long terme, One CLI pourra exposer son catalogue en mode "serveur MCP" pour qui veut l'intégrer dans ce protocole. Le différenciateur reste la **gouvernance par projet via scope file** et la **vault locale**, deux choses que MCP ne couvre pas et n'a pas vocation à couvrir.

## Les concepts centraux

Cinq concepts à maîtriser pour comprendre toute la suite. Ils s'articulent comme ceci :

```
     ┌─────────────────────────────────────────┐
     │              Agent ou humain            │
     └─────────────────┬───────────────────────┘
                       │  one <service> <action>
                       ▼
     ┌─────────────────────────────────────────┐
     │              Binaire `one`              │
     │                                         │
     │   ┌────────┐  ┌────────┐  ┌─────────┐   │
     │   │ Scope  │  │ Vault  │  │ Catalog │   │
     │   │ file   │  │        │  │         │   │
     │   └────────┘  └────────┘  └─────────┘   │
     │                                         │
     │     skill                install        │
     └─────────────────────────────────────────┘
```

### Catalogue

Un répertoire structuré (distribué via un repo Git public + index JSON sur CDN) qui contient la définition de chaque service supporté. Pour chaque service : ses actions, ses permissions, ses providers d'auth, ses install guides, son skill markdown.

Le catalogue est **purement déclaratif** (YAML + Markdown). Quand un service nécessite plus que du déclaratif (signature de requête, GraphQL complexe, chains d'appels), un handler **WASM sandboxé** est attaché.

Le catalogue est **reviewable par PR** : ajouter un service = ouvrir une pull request. Voir [CATALOG.md](./CATALOG.md).

### Vault

Stockage local sécurisé des credentials. Trois implémentations chaînées par priorité :

1. **Variables d'environnement** (`ONE_CREDS_*`) pour CI et override
2. **Keychain natif de l'OS** (macOS Keychain, libsecret Linux, Credential Manager Windows) pour les machines desktop
3. **Fichier chiffré age** (`~/.one/vault.age`) pour les contextes headless sans keychain

Le vault stocke des `Credential` typés (access token, refresh token, expiration, scopes, metadata). Jamais en clair dans des fichiers de config, jamais transmis ailleurs que dans les requêtes HTTP des handlers.

**Multi-comptes natif.** Un service peut avoir plusieurs accounts (`github:work`, `github:perso`). L'utilisateur choisit lequel utiliser par projet via le scope file ou via `--as <alias>` au runtime.

Voir [AUTH.md](./AUTH.md) pour les détails.

### Scope file

Un fichier `.onerc.yaml` versionné à la racine du projet qui déclare *ce qu'un agent peut faire sur ce projet précis*. Format minimaliste :

```yaml
version: 1
services:
  github:
    allow: [issues.*, pulls.read]
    deny: [issues.delete]
  notion:
    allow: [pages.*, blocks.*]
```

**Default deny strict** : tout ce qui n'est pas explicitement autorisé est refusé.

**Layering** : un fichier `.onerc.local.yaml` (gitignored) peut restreindre encore ou changer de compte, mais jamais étendre. Cette règle est non-négociable : le scope file commité est la source de vérité.

**Lock file** `.onerc.lock` fige les versions du catalogue résolues, comme un `package-lock.json`.

Voir [SCOPE.md](./SCOPE.md) pour la grammaire complète.

### Skill

Un document markdown intégré au binaire (`one skill`) qui dit à un agent IA comment utiliser One CLI. Il décrit le discovery flow, les quatre verbes, les exit codes, les patterns idiomatiques, et les anti-patterns.

Le skill est **installable dans l'IDE de l'agent** (`one skill --install`) qui détecte Claude Code, Cursor, Aider, etc., et écrit le skill au bon endroit du projet.

Chaque service a aussi son propre skill (`one info <service>`), focalisé sur le mental model et les gotchas de ce service.

### Install guides

Des recettes markdown qui décrivent les étapes humaines nécessaires pour qu'un service fonctionne. Exemples :

- Partager une page Notion avec l'intégration
- Créer un webhook Stripe
- Configurer un IAM role AWS

Chaque guide a un mode interactif (TTY) et un mode JSON (agent). Il peut déclarer une `verify` action qui prouve que l'install a fonctionné.

Le mécanisme central : quand une action échoue avec un code mappé à un guide, l'erreur sortante inclut directement `install.command: "one install <service> <guide>"`. L'agent reconnaît qu'il doit demander à l'humain, plutôt que de boucler.

## Les quatre verbes de l'agent

L'API surface offerte aux agents est volontairement minimaliste :

```
one <service> <action> [args]    # exécuter une action
one capabilities [<service>]     # introspection JSON (qu'est-ce qui existe)
one info [<service> [<action>]]  # documentation markdown (comment l'utiliser)
one can <service> <action>       # precheck de permission (exit 0/3)
```

Plus une commande utilitaire de hint :

```
one install <service> <guide>    # affiche un guide humain requis
```

Tout le reste (login, scope, accounts, lock, doctor, trace) est pour l'humain qui *configure* le binaire, pas pour l'agent qui *l'utilise*.

## Philosophie

Six règles non-négociables qui tranchent les arbitrages :

### 1. Déclaratif d'abord, code en dernier recours

Le YAML décrit 95% des services (REST classique avec auth). WASM n'arrive que quand le déclaratif est insuffisant (signature SigV4, GraphQL avec interpolation typée, chains d'appels avec rollback).

**Bénéfice** : un reviewer voit ce qu'un service va faire en lisant le YAML. Pas de "lis le code pour comprendre".

### 2. Default deny partout

Le scope file est deny par défaut. Le vault refuse les credentials non déclarées. Les handlers WASM ne peuvent hit que les URLs allowlistées. Aucune permission n'est implicite.

**Bénéfice** : un dev qui ouvre `.onerc.yaml` voit *exactement* ce qui est autorisé, sans avoir à connaître les défauts du système.

### 3. L'audit est first-class, pas un add-on

Chaque exécution est traçable via `one trace`. Chaque requête HTTP émise par un handler est loggée (méthode, URL, status, durée, sans body). Les credentials sont redactées via le type `Secret`.

**Bénéfice** : quand un agent fait quelque chose d'inattendu, on remonte la trace en quelques minutes.

### 4. Pas de magie réseau

Pas de daemon background, pas d'auto-discovery, pas de mDNS, pas de polling caché. Le binaire fait ce que l'utilisateur ou l'agent lui demande explicitement, rien de plus.

**Bénéfice** : la latence, la consommation de batterie, la sécurité réseau sont prévisibles.

### 5. La doc fait partie du livrable

Un service sans `SKILL.md` à jour n'est pas mergé dans le catalogue. Une commande non documentée dans le skill `onecli` n'est pas releasée. Une feature qui n'a pas d'exemple dans la doc n'existe pas.

**Bénéfice** : les agents qui n'ont pas la doc en contexte n'utilisent pas le binaire correctement. La doc est l'interface, pas un bonus.

### 6. Le code est obstinément simple

Pas de framework DI. Pas de génération de code custom. Pas de DSL. Si on hésite entre une abstraction et 50 lignes dupliquées, on prend les 50 lignes.

**Bénéfice** : un contributeur peut comprendre le code en une journée. Aucun magicien à embaucher pour faire évoluer le projet.

## Décisions structurantes et leur rationale

Documenté ici pour que les futurs Ely (et contributeurs) sachent pourquoi ces choix ont été faits, et puissent les remettre en question avec contexte.

### Go pour le binaire

**Pourquoi** : cold start <30ms, cross-compilation native single-binary, écosystème keychain mature, distribution triviale (pas de runtime à installer).

**Pas Rust** : surcoût de complexité non justifié, temps de compilation tueurs de vélocité, pas d'invariants critiques où le borrow checker sauverait.

**Pas TypeScript/Bun** : cold start trop lent pour un binaire appelé 50 fois par session, runtime à installer ou embarquer (50+ MB), keychain bindings moins propres.

### Catalogue en repo Git séparé + index JSON sur CDN

**Pourquoi** : reviewable par PR comme Homebrew, gratuit à héberger (Pages ou R2), pas d'infra serveur à maintenir.

**Pas un registry custom HTTP** : single point of failure, coût d'infra, complexité de maintenance.

**Pas décentralisé (un repo par service)** : discovery cassée, qualité variable, pas de listing centralisé.

### WASM (wazero) pour les handlers complexes

**Pourquoi** : sandbox par défaut (WASI = rien d'exposé), polyglotte (TS via Javy, Go via tinygo, Rust direct), distribuable dans le tarball du service.

**Pas de plugin natif Go** : pas de sandbox, sécurité catastrophique, binaire non-reproductible.

**Pas d'exécution de scripts arbitraires (Lua, JS V8)** : même problème de sandbox, ou alors ajoute des dépendances natives qui cassent la cross-compilation.

### Scope file versionné dans la repo

**Pourquoi** : code review des permissions, reproductibilité entre devs, source de vérité auditable.

**Pas dans un cloud config** : créerait une dépendance, casserait l'offline-first, créerait un single point of failure.

**Pas dans le vault** : le scope est public (committable), pas secret. Mélanger les deux est une erreur de design.

### Install guides en markdown avec frontmatter

**Pourquoi** : lisible par humains, parsable par machines, versionnable dans le catalogue, traduisible.

**Pas en YAML pur** : illisible pour un humain qui suit les étapes.

**Pas en script exécutable** : ramène la sandbox dans les guides, vecteur d'attaque.

### Exit codes typés (0/1/2/3/4/5)

**Pourquoi** : l'agent peut écrire de la logique conditionnelle propre (`if exit==3 then ask scope`) sans parser des messages d'erreur en langage naturel.

**Pas juste 0/1** : trop pauvre, force le parsing de stderr.

**Pas 7-bit ASCII full** (genre 0-127) : convention Unix dit que >128 sont signaux, on reste dans la norme.

## Non-buts

Ce que One CLI **ne fait pas** et n'a pas vocation à faire. Important à clarifier pour éviter les dérives.

**One CLI n'est pas un orchestrateur de workflows.** Pas de DAGs, pas de retries automatiques entre actions, pas de scheduling. Tu veux ça ? Utilise Temporal, Trigger.dev, ou un script.

**One CLI n'est pas un proxy API public.** Ce n'est pas un service qu'on déploie, c'est un binaire local. Si tu veux un proxy avec rate limiting, route vers `kong` ou `nginx`.

**One CLI n'est pas un LLM gateway.** Anthropic, OpenAI, etc. peuvent être *appelés* via le catalogue, mais ce n'est pas un point d'entrée unifié pour les complétions (style portkey, OpenRouter). Si tu veux ça, utilise ces outils dédiés.

**One CLI ne gère pas le code généré par les agents.** Pas d'exécution de code dans un sandbox. C'est un outil pour *appeler des services*, pas pour *lancer du code*.

**One CLI ne fait pas de pricing par usage.** Open source pur. Les credentials sont à toi, les appels API sont facturés par les services tiers comme d'habitude.

**One CLI n'est pas un agent.** C'est un *outil* utilisable par des agents. Il ne décide pas, il exécute.

## Cible utilisateur

Trois publics, dans cet ordre de priorité :

**1. Le dev qui build des agents IA en 2026.** Il utilise Claude Code, Cursor, ou un agent custom. Il veut shipper des intégrations rapidement sans gérer 10 SDKs. Il préfère un binaire local sous son contrôle à un service tiers.

**2. L'agent IA lui-même.** Il consomme le skill `onecli`, fait des `capabilities`/`info`/`exec`, gère les exit codes. C'est l'utilisateur le plus *fréquent* du binaire mais pas celui qui le configure.

**3. Le team lead qui veut gouverner.** Il définit le scope file pour son équipe, il review les PRs sur `.onerc.yaml`, il s'assure qu'aucun agent ne va `delete repos` par accident en prod.

**Pas ciblés au début** : les utilisateurs grand public non-techniques, les entreprises avec compliance SOC2 strict (viendront en v1.0+).

## Métriques de succès du design

Comment savoir si le design tient ses promesses ? Quatre signaux à surveiller :

1. **Temps pour ajouter un service simple au catalogue** (genre Resend) : <2h pour un dev qui découvre. Si c'est 8h, le format service.yaml est trop complexe.

2. **Temps pour qu'un agent comprenne One CLI** : <30s de lecture du skill avant de pouvoir l'utiliser. Si l'agent se trompe systématiquement après avoir lu le skill, la doc est mal écrite.

3. **Nombre de questions "comment je fais X" sur GitHub Discussions** : devrait diminuer au cours du temps à mesure que la doc s'enrichit. Si ça reste stable, le projet est moins bien documenté qu'il devrait.

4. **Surface des breaking changes par version** : 0 entre versions mineures, listés explicitement entre majeures. Si on en a 3 par mineure, l'API n'était pas mûre.

## Évolution prévue

Ce design est figé pour v0.1 à v1.0. Après v1.0, les évolutions possibles :

- **Mode serveur MCP** : exposer le catalogue One CLI comme un serveur MCP pour l'interop.
- **Audit log distant** : envoi opt-in des traces vers un endpoint pour les équipes qui veulent du monitoring centralisé.
- **Templates de scope** : `.onerc.yaml.template` partagés par communauté (genre "scope pour un agent customer support", "scope pour un agent DevOps").
- **Tier entreprise** : support payant pour les organisations, sans changer le core open source.

Aucune de ces évolutions ne casse les concepts décrits dans ce document.

---

*Document maintenu en parallèle du code. Si une décision décrite ici devient obsolète, il faut soit mettre à jour ce document avec un changelog clair, soit reverter le code.*
