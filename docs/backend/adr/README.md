# Backend architecture decision records

Decisions confined to the server and the CLI. Cross-cutting and product
decisions live in [`docs/adr/`](../../adr/); the lifecycle rules — append-only,
never deleted, a reversal is a new ADR carrying links on both sides — are set by
[ADR 0009](../../adr/0009-documentation-strategy.md).

| # | Decision |
| --- | --- |
| [0001](0001-hexagonal-layers.md) | Hexagonal layers, enforced by linter |
| [0002](0002-spec-first-openapi.md) | Spec-first OpenAPI, generated on both sides |
| [0003](0003-postgres-stack.md) | PostgreSQL with pgx, sqlc and goose |
| [0004](0004-filesystem-blob-store.md) | Capture bytes on the filesystem, behind a port |
