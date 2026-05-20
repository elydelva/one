# One CLI

> Un binaire local qui donne à tes agents IA un accès unifié, gouverné et auditable aux services tiers.

```bash
one notion pages.create --parent_page_id abc-123 --properties '{"title":[{"text":{"content":"Hello"}}]}'
```

Pas besoin d'apprendre dix SDKs. Pas besoin de gérer dix vaults. Pas besoin de coder dix workflows d'auth. Un seul binaire, un catalogue communautaire, et un fichier `.onerc.yaml` versionné qui dit exactement ce que tes agents ont le droit de faire sur ton projet.

## En 30 secondes

- **Un binaire** Go cross-platform, installation en une ligne, cold start <30ms.
- **Un catalogue** de services open source, contribué via PR (~40 services au lancement).
- **Un vault** local multi-comptes : keychain natif de l'OS, jamais en SaaS, multi-providers (OAuth, PAT, API key, AWS).
- **Un scope file** `.onerc.yaml` versionné qui rend explicite, reviewable et default-deny ce qu'un agent peut faire.
- **Des install guides** first-class pour les setups qui ne s'automatisent pas (genre "partage cette page Notion avec l'intégration").
- **Un audit log** local, `one trace` pour voir ce qui a été fait.

## Quick start

### Installer

```bash
curl -fsSL https://one-cli.dev/install.sh | sh
one --version
```

Ou via Homebrew :

```bash
brew install one-cli/tap/one
```

### Setup d'un projet

```bash
cd mon-projet
one init                              # crée .onerc.yaml

one login github                      # OAuth flow, browser
one login notion --as kaampus         # OAuth, alias pour multi-comptes

one scope add github "issues.*"       # globs supportés
one scope add github pulls.read
one scope add notion "pages.*"

one lock                              # fige les versions du catalog
git add .onerc.yaml .onerc.lock
git commit -m "init: setup One CLI"
```

### Exécuter une action

```bash
one github issues.create \
  --repo me/myrepo \
  --title "Bug: foo" \
  --body "Detailed description"
```

Output JSON :

```json
{
  "ok": true,
  "data": { "id": 42, "url": "https://github.com/me/myrepo/issues/42" },
  "trace_id": "01HXYZ..."
}
```

### Workflow pour un agent IA

```bash
one capabilities --scope-only         # qu'est-ce que je peux faire ?
one info github                       # comment ça marche ?
one can github issues.create          # est-ce que c'est dans le scope ?
one github issues.create ...          # exécution
```

Voir [`one skill`](./CLI.md#one-skill) pour installer le skill dans Claude Code, Cursor, etc.

## Vs MCP

| Dimension | MCP | One CLI |
|---|---|---|
| Forme | Serveur HTTP/stdio par service | Un binaire local |
| Auth | Géré par chaque serveur | Vault unifié multi-comptes |
| Scope par projet | Pas couvert | `.onerc.yaml` versionné |
| Setup humain | Pas couvert | `one install` first-class |
| Distribution | Un package par serveur | Catalogue centralisé reviewable |
| Audit | Logs par serveur | Audit log unifié `one trace` |

One CLI n'est pas anti-MCP. Les deux coexistent. La différence de fond est la **gouvernance par projet** via scope file et la **vault locale unifiée**.

## Documentation

| Document | Pour |
|---|---|
| [DESIGN.md](./DESIGN.md) | Comprendre le **pourquoi** : pitch, positionnement, concepts, philosophie |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Contribuer au binaire Go : layout, ports & adapters, patterns |
| [CATALOG.md](./CATALOG.md) | Ajouter un service au catalogue : format service.yaml, process PR |
| [HANDLERS.md](./HANDLERS.md) | Écrire un handler WASM : contrat, host functions, SDKs TS/Go/Rust |
| [SCOPE.md](./SCOPE.md) | Maîtriser le scope file : grammaire, layering, lock file |
| [AUTH.md](./AUTH.md) | Comprendre l'auth : providers, flows OAuth, vault, multi-comptes |
| [SECURITY.md](./SECURITY.md) | Threat model, sandbox WASM, disclosure policy |
| [TESTING.md](./TESTING.md) | Stratégie de test : pyramide, contract tests, fakes vs mocks |
| [CLI.md](./CLI.md) | Référence des commandes, exit codes, env vars |
| [CONTRIBUTING.md](./CONTRIBUTING.md) | Premier setup, workflow, code style |

## Services supportés (v0.5 cible)

GitHub, Notion, Linear, Stripe, Resend, Slack, Google Drive, Google Calendar, Gmail, AWS S3, Cloudflare, OpenAI, Anthropic, Vercel, Supabase, PostgreSQL, Discord, Twilio, SendGrid, HubSpot, Airtable, Asana, Trello, Figma, Bitbucket, GitLab, Sentry, PagerDuty, Datadog, MongoDB Atlas, Mailchimp, Calendly, Zoom, Microsoft Graph, Reddit, Twitter/X, Webflow, Shopify.

Catalogue complet : `one catalog search` ou https://one-cli.dev/catalog.

## Status

**Pre-alpha** : v0.1 en cours de développement. Pas stable. Voir [ROADMAP.md](./ROADMAP.md) pour le sequencing.

Ne pas utiliser en production. Open issue ou contribute si quelque chose vous frustre.

## License

Apache 2.0. Voir [LICENSE](./LICENSE).

## Crédits

Conçu et développé par [@elydelva](https://github.com/elydelva). Inspiré par : Homebrew formulae (modèle de catalogue), [Inngest](https://inngest.com) (philosophie agent-native), MCP (positionnement), git (philosophie de versioning local).

Contributions bienvenues : voir [CONTRIBUTING.md](./CONTRIBUTING.md).

---

*Questions ? Idées ? [GitHub Discussions](https://github.com/one-cli/one/discussions).*
