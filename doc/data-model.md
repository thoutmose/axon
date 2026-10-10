# Data model

Conventions: [ADR 0005](adr/0005-entity-conventions.md). Entities and links: [ADR 0006](adr/0006-entity-registry-and-links.md). Classification and retention: [privacy.md](privacy.md).

```mermaid
erDiagram
  users ||--o{ passkeys : has
  users ||--o{ sessions : has
  users ||--o{ recovery_codes : has
  users ||--o{ audit_events : has
  users ||--o{ consents : gives
  users ||--o{ entities : owns
  users ||--o{ events : owns
  entities ||--o{ links : "source"
  entities ||--o{ links : "target"
  entities ||--o| domain_table : "is (tasks, notes, goals, …)"

  entities {
    uuid id PK
    uuid user_id FK
    text type
    text title
    text search_text
    tsvector search "generated"
  }
  links {
    uuid id PK
    uuid user_id "FK with source and target"
    uuid source_id FK
    text relation
    uuid target_id FK
  }
  events {
    uuid id PK
    uuid user_id FK
    text actor
    text entity_type
    uuid entity_id "no FK: outlives the entity"
    text kind
    text_array fields "names only"
  }
```

## Rules for a new entity type

1. Create the domain table with `id uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE`; no `user_id` column is needed (ownership lives on `entities`).
2. Register it with `graph.RegisterType` (name, table, export function) and any typed relations with `graph.RegisterRelation`.
3. Create rows with `graph.CreateEntity` and the domain insert in one transaction; update with `graph.UpdateEntity`, listing changed field names; delete with `graph.DeleteEntity`.
4. Put searchable text in `title` and `search_text`.

`TestEveryEntityTableIsRegistered` fails if step 2 is missed, and `axon integrity` (run on every deployment) fails if step 3 is.
