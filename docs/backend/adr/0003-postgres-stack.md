# ADR 0003 (backend) — PostgreSQL with pgx, sqlc and goose

## Status

Accepted — 2026-08-19

## Context

[ADR 0011](../../adr/0011-technology-stack.md) settled PostgreSQL as the state
store. Three tooling decisions follow and are tied together: the driver, the way
queries are written, and the way the schema evolves. Choosing them separately
invites impedance mismatch — an ORM that hides the driver, a migration tool that
needs its own runtime.

What the schema has to hold is unusual enough to matter. The journal
([ADR 0002](../../adr/0002-server-owns-the-review-lifecycle.md)) is append-only,
carries a semi-structured `inputs` fingerprint, and is the audit trail: it must
be queryable and exportable, not a log file. Facts and the transitions they
cause are written together or not at all.

## Decision

**PostgreSQL + pgx + sqlc + goose.**

- **pgx** as the driver, on its native interface — not through `database/sql`.
- **sqlc** for queries: SQL is written by hand under
  `apps/server/db/queries/*.sql` and compiled into typed Go functions.
- **goose** for migrations: one SQL file per migration, `-- +goose Up` and
  `-- +goose Down` in the same file.

## Rationale

| Choice | Why |
| --- | --- |
| PostgreSQL | JSONB for `inputs` fingerprints with no second store; real transactions so a fact and its transition commit together; `LISTEN`/`NOTIFY` feeding the SSE stream with no broker; partial and expression indexes for the state and freshness filters of `product.md` § 9 |
| pgx | Native protocol, typed values instead of `interface{}`, first-class JSONB and array support, and the connection-level `LISTEN` support the event stream needs |
| sqlc | The SQL is the source and Go is generated — the same philosophy as the OpenAPI document ([ADR 0002 backend](0002-spec-first-openapi.md)). A query that drifts from the schema is a generation error, not a runtime one |
| goose | One readable file per migration; a binary is enough to run it, no Go runtime at migration time |

Writing SQL by hand is a deliberate choice, not a concession. The queries that
matter here — filtering cases by stored state and freshness, walking a
transition journal, resolving which captures moved between two editions — are
exactly the ones a query builder makes harder to read and harder to index
deliberately.

## Alternatives rejected

**GORM or ent.** Rejected: an ORM hides the SQL behind a builder, and the type
safety it offers is the type safety sqlc already gives without giving up the
query.

**sqlx.** Rejected: it adds struct mapping over `database/sql` but queries stay
strings checked at runtime. Half the boilerplate removed, none of the checking
gained.

**Raw `database/sql`.** Rejected: `rows.Scan` boilerplate on every query, no
protection against a query drifting from the struct it fills.

**golang-migrate.** Rejected: two files per migration doubles the file count for
no readability gain.

**An event-sourced store, the journal as the only state.** Genuinely close to
the product's model, since transitions are already append-only. Rejected: the
product needs both — a journal *and* a directly queryable current state, which
is the whole point of
[ADR 0002](../../adr/0002-server-owns-the-review-lifecycle.md). A projection
rebuilt on read is the derived-on-read model that ADR rejected. Storing the
current state in a table alongside the journal is the simpler half of the same
guarantee.

## Consequences

- Writing a query means writing SQL, running the generator, and calling the
  generated function. Generated code is committed; CI verifies regeneration
  produces no diff.
- One feature, one migration. A migration in development may be iterated
  (down, edit, up) until it ships; once merged it is never modified.
- Transactions are opened in the repository, never in a use case — `app` cannot
  import an adapter ([backend ADR 0001](0001-hexagonal-layers.md)) and therefore
  cannot know a transaction exists. A write spanning several tables becomes one
  repository method.
- Blob bytes are **not** in Postgres. The database stores hashes and pointers
  ([backend ADR 0004](0004-filesystem-blob-store.md)).

## Cross-references

- [ADR 0002](../../adr/0002-server-owns-the-review-lifecycle.md) — the journal
  this schema must serve
- [ADR 0011](../../adr/0011-technology-stack.md) — where this sits in the stack
