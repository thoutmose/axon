# Architecture

Comment Axon est construit, sur quoi il tourne et comment le configurer. Retour au [README](../README.fr.md) · [English](architecture.md).

## Architecture

Axon est d’abord un monolithe modulaire en Go, avec PostgreSQL 16 + pgvector comme source de vérité (le travail de fond tourne dans le processus, coordonné par PostgreSQL), et SvelteKit + TypeScript pour l’application web. Les agents appellent des outils bornés via une couche d’orchestration qui respecte l’approbation.

### Mermaid — contexte du système

```mermaid
flowchart LR
  user(["Person: owns the data, approves every AI change"])
  axon["Axon: self-hosted life OS"]
  ai["AI provider of each person's choice: Anthropic, OpenAI, Mistral, Gemini, DeepSeek, or a local model"]
  cat["Catalogues: Open Library, TMDB, Open Food Facts, USDA"]
  anki["anki-sync-server"]
  vault["Markdown vault, e.g. Obsidian"]
  user -->|"plans, captures, learns, reviews"| axon
  axon -->|"approved prompts and tools, with consent"| ai
  axon -->|"lookups, with consent"| cat
  axon -->|"flashcards"| anki
  axon <-->|"notes and items, both ways"| vault
```

### Mermaid — conteneurs

```mermaid
flowchart TB
  user(["Browser or phone"])
  subgraph host["One host per environment, Docker Compose"]
    caddy["Caddy: HTTPS front, headers"]
    web["Web: SvelteKit + TypeScript"]
    api["API: Go + chi, REST and OpenAPI 3.1"]
    workers["In-process workers: agents, vault mirror, demo reset, media prune"]
    bridge["Anki bridge: official Anki library"]
    db[("PostgreSQL 16 + pgvector: system of record")]
    backup["Backup: encrypted dumps"]
  end
  user --> caddy
  caddy --> web
  caddy -->|"/api"| api
  web -->|"server-side calls"| api
  api --- workers
  api --> db
  workers --> db
  api --> bridge
  backup --> db
```

### Mermaid — déploiement

```mermaid
flowchart LR
  dev["Your machine: make up, make check"] -->|"deploy.sh staging, green CI only"| staging["srv-staging: Axon + PostgreSQL"]
  staging -->|"deploy.sh prod, only what staging serves"| prod["srv-prod: Axon + PostgreSQL"]
  prod -->|"Anki sync, encrypted backups, tailnet"| services["srv-services: anki-sync-server, backups"]
  prod <-->|"notes-sync -watch, tailnet"| brain["srv-secondbrain: Markdown vault"]
```

### Mermaid — modèle de données et liens universels

```mermaid
erDiagram
  USER ||--o{ ENTITY : owns
  ENTITY ||--o{ LINK : "source or target"
  ENTITY ||--o{ EVENT : "history"
  ENTITY ||--o| NOTE : "is a"
  ENTITY ||--o| TASK : "is a"
  ENTITY ||--o| GOAL : "is a"
  GOAL ||--o{ MILESTONE : has
  ENTITY ||--o| PROJECT : "is a"
  ENTITY ||--o| HABIT : "is a"
  ENTITY ||--o| ROADMAP : "is a"
  ROADMAP ||--o{ SKILL : has
  ENTITY ||--o| CARD : "is a"
  CARD ||--o{ CARD_REVIEW : "reviewed in"
  ENTITY ||--o| MEDIA_ITEM : "is a"
  ENTITY ||--o| VAULT_NOTE : "mirrored as"
  ENTITY {
    uuid id PK
    uuid user_id FK
    string type
    string title
  }
  LINK {
    uuid id PK
    uuid source_id FK
    string relation
    uuid target_id FK
  }
  EVENT {
    uuid id PK
    string kind
    string fields
  }
```

### Mermaid — boucle d’apprentissage et de mémorisation

```mermaid
sequenceDiagram
  actor P as Person
  participant N as Notes
  participant T as Teacher agent
  participant R as Review engine, FSRS
  participant L as Learn: today's plan
  participant A as anki-sync-server
  P->>N: captures fleeting notes, rewrites them as literature and permanent notes
  P->>N: marks what to remember: q :: a, ==term==, **Term**: definition
  N-->>R: Make cards: drafts from the marks, added by the person
  P->>T: illustrated cards from these notes
  T->>N: reads only literature and permanent notes
  T-->>P: proposes cards with the why, an example, a vivid image, an SVG drawing
  P->>R: approves, then places cards in a memory palace
  loop Every day
    L-->>P: cards due, a few new ones, decks mixed, and notes due for recall
    P->>R: writes the answer, then sees it with the why, the example, and the image, and rates it
    R->>R: schedules the next review at the chosen retention
    P->>N: recalls a whole note on a blank page
    N-->>L: ideas recalled and missed, next recall further away when recalled well
  end
  L-->>P: weekly mixed test, and leeches to practise and rewrite
  R->>A: syncs approved cards
```

### Mermaid — évaluation et parcours du Tuteur

```mermaid
sequenceDiagram
  actor P as Person
  participant As as Assessment
  participant Tu as Tutor agent
  participant L as Curated library
  participant Ro as Roadmaps
  P->>As: situation, job, goals, current level
  As-->>Ro: places you on roadmaps
  P->>Tu: what to learn, time, format
  Tu->>L: searches the best resources
  L-->>Tu: books, courses, papers, videos
  Tu-->>P: draft roadmap, every skill cites its source
  P->>Ro: approves, edits, or removes
```

### Mermaid — miroir du coffre, dans les deux sens

```mermaid
sequenceDiagram
  participant I as Item, e.g. a task
  participant M as Vault mirror worker
  participant N as Note under Axon/
  participant V as Markdown vault
  I->>M: created, changed, or deleted
  M->>N: writes front matter and sections
  N->>V: notes-sync carries the file
  V->>N: you edit, add, move, or delete the file
  N->>M: change in the event log
  M->>I: updates, creates, archives, or deletes the item
```

### Mermaid — série quotidienne et gels

```mermaid
stateDiagram-v2
  [*] --> Idle
  Idle --> Active: first qualifying action of the day
  Active --> Active: an action on each following day
  Active --> AtRisk: a day passes without an action
  AtRisk --> Active: act today
  AtRisk --> Frozen: a freeze bought with earned points covers the day
  Frozen --> Active: act on the next day
  AtRisk --> Restart: two days without an action
  Restart --> Active: start again, without shame
```

## Pile technique

| Couche | Choix | Pourquoi |
|---|---|---|
| Backend | Go + chi | Peu de mémoire, concurrence explicite, déploiement simple, très bons outils. |
| API | REST + OpenAPI 3.1 ; GraphQL éventuellement plus tard | Cache, observabilité, clients générés ; pas de flexibilité prématurée. |
| Frontend | SvelteKit + TypeScript, jetons de design + CSS limité aux composants ([ADR 0007](adr/0007-styling-without-tailwind.md)) | Une interface petite et rapide, accessible, en amélioration progressive. |
| Données | PostgreSQL 16 + pgvector | Un cœur relationnel durable : texte intégral, vecteurs, contraintes, extensions. |
| Travail de fond | Workers dans le processus, lisant PostgreSQL (curseur du journal d’événements, tables de file) | Aucun service de plus ; la base reste l’unique source de vérité ([ADR 0022](adr/0022-no-valkey-yet.md)). |
| Vue graphe | D3 ou Sigma.js | Visualisation interactive des compétences et des relations, qui passe à l’échelle. |
| Authentification | WebAuthn + TOTP intégrés | Clés d’accès, MFA, contrôle auto-hébergé. |
| Entrée réseau | Caddy ou Traefik | TLS automatique et routage clair. |
| Exploitation | Prometheus, Grafana, Loki, OpenTelemetry | Métriques, tableaux de bord, journaux, traces. |
| Livraison | GitOps ou Compose + Ansible/OpenTofu | Une promotion reproductible adaptée aux petits parcs. |
| Sécurité | SOPS/age, Trivy, SBOM, Renovate, pratiques SLSA | Réduire les risques de chaîne d’approvisionnement et de secrets. |

Là où le tableau laisse un choix, l’[ADR 0002](adr/0002-initial-technology-stack.md) en propose un : WebAuthn + TOTP intégrés, Caddy, et Compose + Ansible.

**Pourquoi Go et pas Rust ?** Rust apporterait des garanties plus fortes à la compilation et un peu moins de mémoire. Mais le travail d’Axon est surtout de l’entrée-sortie (PostgreSQL, HTTP, fournisseurs d’IA), où Go est aussi rapide en pratique. Go compile en quelques secondes, dispose de SDK officiels pour les services utilisés par Axon, et garde le code facile à lire pour un mainteneur seul comme pour des agents IA. Réécrire l’API existante arrêterait les nouvelles fonctionnalités pendant des mois, sans gain visible pour les utilisateurs. Le choix est consigné dans l’ADR 0002, et le [journal des changements](BACKLOG.md#change-log) note qu’une réécriture a été envisagée sans être planifiée.

## Matériel requis

Un seul hôte fait tout tourner : Caddy, l’API et ses tâches de fond, l’application web, PostgreSQL avec pgvector, la passerelle Anki et les sauvegardes ([ADR 0003](adr/0003-single-host-first.md), [ADR 0004](adr/0004-hosts-and-tailnet-only-access.md)). `srv-prod` sert d’exemple :

| | Minimum | `srv-prod` (l’exemple) | Ce qu’Axon y utilise (mesuré le 17 octobre 2026) |
|---|---:|---:|---|
| Processeur | 2 vCPU | 8 vCPU | Moins de 1 % au repos ; une récupération de la veille ou un import occupe un cœur quelques secondes. |
| Mémoire | 4 Go | 16 Go | Environ 120 Mo pour les six conteneurs au repos ; 300 Mo en charge ([ADR 0021](adr/0021-single-host-measured.md)). |
| Disque | 40 Go SSD | 120 Go SSD | 1,8 Go d’images, une base de 19 Mo, 1 Mo de sauvegardes chiffrées ; les images des notes et les couvertures s’ajoutent à la base. |

Mesuré le 11 octobre 2026, un hôte sert 700 requêtes par seconde. Un hôte de base de données séparé ne se justifie que si une mesure le montre. Le serveur de synthèse vocale facultatif (Kokoro, pour des voix naturelles dans le podcast) demande environ 8 Go de disque et 2 Go de mémoire en plus ; Whisper, pour les vidéos sans sous-titres, davantage : faites-les tourner sur une autre machine via `AXON_TTS_URL` et `AXON_WHISPER_URL` si l’hôte est petit.

> [!NOTE]
> Les petits hôtes suffisent parce que les tâches de fond sont bornées, les connexions mutualisées, les listes paginées, et les modèles tournent ailleurs : les appels d’IA vont au fournisseur que chacun choisit, un serveur Ollama local compris, limités en débit et dans un budget mensuel.

## Configuration

| Variable | Obligatoire | Exemple | Rôle |
|---|---|---|---|
| `AXON_BASE_URL` | Oui | `https://axon.example` | URL canonique et origine des retours. |
| `DATABASE_URL` | Oui | `postgres://...` | Connexion PostgreSQL ; à stocker comme secret. |
| `AXON_ENCRYPTION_KEY` | Prod | `openssl rand -hex 32` | Chiffre les secrets TOTP au repos ; ne jamais la committer. |
| `AXON_REGISTRATION` | Non | `first-user` | `first-user` : seul le premier compte peut s’inscrire ; `open` : toute personne qui atteint Axon. |
| `AXON_DEMO` | Non | `1` | Propose un compte de démonstration partagé, avec des données d’exemple remises à zéro chaque nuit, et montre la bande-annonce d’une minute sur la page d’accueil. |
| `WEBAUTHN_RP_ID` | Prod | `axon.example` | Identifiant de la partie de confiance pour les clés d’accès. |
| `ANTHROPIC_API_KEY` | Facultatif | secret | Clé Anthropic par défaut du serveur pour les agents (chacun peut aussi choisir son fournisseur). |
| `AXON_ANTHROPIC_OAUTH` | Facultatif | config | `1` pour utiliser un profil `ant auth login` au lieu d’une clé. |
| `TMDB_API_KEY` | Facultatif | secret | Recherche des films et séries (votre propre clé TMDB non commerciale, clé v3 ou jeton de lecture v4). |
| `USDA_API_KEY` | Facultatif | secret | Recherche d’aliments dans USDA FoodData Central (clé data.gov gratuite). |
| `AXON_CALENDAR_PRIVATE` | Facultatif | valeur | `1` autorise des agendas suivis sur votre propre réseau (un Nextcloud sur le réseau local ou le tailnet) ; sinon, seules les adresses HTTPS publiques sont lues. |
| `AXON_SMTP_URL` | Facultatif | secret | E-mails de rappel par votre serveur de messagerie : `smtp://user:pass@host:587` ou `smtps://user:pass@host:465`. |
| `AXON_SMTP_FROM` | Facultatif | valeur | L’adresse d’expédition des e-mails de rappel. |
| `AXON_GOOGLE_CLIENT_ID`, `AXON_GOOGLE_CLIENT_SECRET` | Facultatif | secret | Connexion avec Google ; aussi `AXON_MICROSOFT_*` (avec `AXON_MICROSOFT_TENANT`), `AXON_DISCORD_*`, `AXON_SLACK_*`. Adresse de retour : `AXON_BASE_URL/api/v1/auth/oauth/<fournisseur>/callback`. |
| `AXON_APPLE_CLIENT_ID`, `AXON_APPLE_TEAM_ID`, `AXON_APPLE_KEY_ID`, `AXON_APPLE_PRIVATE_KEY` | Facultatif | secret | Connexion avec Apple : un Services ID et la clé `.p8` (ou `AXON_APPLE_PRIVATE_KEY_FILE`). |
| `AXON_OIDC_ISSUER`, `AXON_OIDC_CLIENT_ID`, `AXON_OIDC_CLIENT_SECRET`, `AXON_OIDC_NAME` | Facultatif | secret | Connexion avec n’importe quel fournisseur OpenID Connect (Authentik, Keycloak, GitLab…). |
| `AXON_CONTACT` | Facultatif | `vous@example.org` | Comment Open Library peut joindre l’opérateur (envoyé dans le User-Agent) ; par défaut `AXON_BASE_URL`. |
| `ANKI_SYNC_URL` | Facultatif | URL privée | Adresse suggérée du serveur anki-sync-server pour le pont Anki (chacun saisit son propre compte). |
| `AXON_PORT` | Non | `3000` | Le port que Caddy publie sur 127.0.0.1 (compose) ; une seconde pile, comme les tests de bout en bout, tourne à côté de la première sur un autre. |
| `AXON_VERSION` | Non | `dev` | L’étiquette des images que compose construit et lance ; `deploy.sh` y met le commit qu’il déploie. |
| `AXON_TRUSTED_PROXIES` | Non | `100.95.163.114/32` | Adresses d’un proxy placé devant Tailscale Serve (celui du domaine d’Axon), pour que Caddy voie l’adresse de chaque visiteur et non celle du proxy. |
| `AXON_ADDR`, `AXON_METRICS_ADDR` | Non | `:8080`, `:9464` | Où l’API écoute, et où elle sert les métriques Prometheus, à part des routes publiques. |

Ne mettez jamais de secrets dans du Markdown, des tickets, des prompts, des captures d’écran ou des journaux. Utilisez SOPS/age et un gestionnaire de secrets adapté au déploiement.

## Sécurité et RGPD

Axon suit l’OWASP ASVS comme base de vérification et les recommandations CIS pour les hôtes. Le trafic entre serveurs passe par WireGuard ou Tailscale ; l’entrée publique se limite au proxy inverse. Les clés d’accès WebAuthn et le TOTP protègent les comptes ; les sessions tournent et se révoquent. Les contrôles d’appartenance par utilisateur s’appliquent à chaque frontière de l’API.

Les contrôles de confidentialité comprennent les registres de consentement, la limitation des finalités, les durées de conservation, l’export, les demandes d’effacement, les journaux d’audit, l’inventaire des sous-traitants et le masquage des prompts et des journaux. Les sauvegardes sont chiffrées, avec des délais de suppression documentés. Les données de santé et d’apprentissage sont sensibles ; la visibilité par défaut est privée.

> [!NOTE]
> La conformité au RGPD est un travail d’exploitation continu, pas un badge. Demandez un avis juridique adapté à votre déploiement et à votre juridiction.

## Tests et exploitation

- **Tests :** tests unitaires Go, tests d’intégration PostgreSQL, tests de contrat OpenAPI, tests navigateur Playwright, contrôles d’accessibilité, évaluations des agents, répétition des migrations et exercices de restauration des sauvegardes.
- **Observabilité :** métriques Prometheus, tableaux de bord Grafana, journaux Loki, traces OpenTelemetry, identifiants de corrélation, profondeur des files, latence des fournisseurs, coût en jetons et décisions d’agents masquées.
- **Fiabilité :** définir SLO, alertes, RPO/RTO, limites de débit, clés d’idempotence, disjoncteurs et procédures avant de déclarer une fonctionnalité terminée.
- **Chaîne d’approvisionnement :** Renovate, SBOM, analyses Trivy, images épinglées, artefacts signés quand c’est possible, et provenance inspirée de SLSA.

## Méthode de développement

1. Prendre le prochain ticket prêt dans [BACKLOG.md](BACKLOG.md) ; ne pas travailler sur plusieurs Epics à la fois.
2. Créer une branche ciblée et écrire d’abord les tests ou les exemples d’acceptation.
3. Livrer une petite tranche verticale ; garder les migrations réversibles et les API documentées.
4. Lancer formatage, lint et tests unitaires, d’intégration, de contrat, d’accessibilité et de sécurité.
5. Ouvrir une PR/MR en Conventional Commits avec preuves, risques, retour arrière et documentation.
6. Demander une revue indépendante et une approbation humaine, fusionner, déployer en préproduction et observer avant la promotion.

## Feuille de route

| Phase | Résultat | Backlog, dans l’ordre |
|---|---|---|
| Fondations | Déploiement sécurisé, CI, observabilité, authentification, confidentialité. | EP-00 → 01 → 02 → 03 → 04 |
| Graphe personnel | Liens, tâches, notes, objectifs, tableau de bord, sans IA. | EP-05 → 06 → 07 → 09 → 11 → 13 → 08 |
| Croissance guidée | Agents, habitudes, projets, parcours, espace de notes, agents de livraison et Tuteur. | EP-19 → 12 → 22 → 23 → 24 → 25 → 10 → 26 → 14 → 27 |
| Usage quotidien | Démo, docs bilingues, tableau de bord, réglages, raccourcis, avatars, Atomic Habits, Pomodoro, apprentissage et Anki, gamification, coffre en direct, graphe de connaissances, espaces thématiques, finances, portabilité. | EP-28 → 29 → 30 → 41 → 31 → 32 → 33 → 34 → 35 → 16 → 42 → 15 → 36 → 43 → 39 → 38 → 37 → 40 |
| Bêta vie entière | Médias, santé, nutrition, analyses. | EP-17 → 18 → 45 → 46 → 44 → 20 |
| 1.0 | Durcissement, restauration, documentation, chemin de mise à jour. | EP-21 |
| 1.1 | Demandes du propriétaire : graphe Zettelkasten, tout dans le coffre, diagrammes, images dans les notes, moteur d’apprentissage et de mémoire, agents sur chaque page, application installable, rappels, saisie rapide, agendas externes, revue hebdomadaire, imports, foyer, recherche par le sens, Aujourd’hui personnalisable, actions groupées, corbeille et annulation, partager vers Axon, rappels de tâches, code propre. | EP-47 → 48 → … → 69 |

État actuel : **toutes les Epics de [doc/BACKLOG.md](BACKLOG.md#delivery-order) jusqu’à EP-169 sont livrées et publiées en 1.0.0.** Pour découvrir Axon sans compte, utilisez **Essayer la démo** sur la page d’accueil : un compte partagé avec des données d’exemple, réinitialisé chaque nuit. Exploitation : [doc/operations.md](operations.md).
