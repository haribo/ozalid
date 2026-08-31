# ADR 0010 — One repository holding server, web client and CLI

## Status

Accepted — 2026-08-19. **Partly superseded by
[ADR 0015](0015-no-cli-clients-call-the-api.md)** on 2026-08-21: the monorepo
layout stands, the CLI is withdrawn. The reasoning below about a CLI being
structurally necessary is the part that did not hold — see ADR 0015 for why.

## Context

ozalid ships three artifacts, and they are not independent.

- **A server.** It owns the review lifecycle
  ([ADR 0002](0002-server-owns-the-review-lifecycle.md)) and the capture store
  ([ADR 0004](0004-content-addressed-capture-storage.md)).
- **A web client.** The book a reviewer reads and judges.
- **A CLI.** [ADR 0003](0003-runner-and-tracker-agnostic-boundaries.md) puts the
  runner adapter and the tracker knowledge outside the server, on the client
  side. Without the CLI, nothing can push an edition — the product has no input
  at all, and no end-to-end path to test.

All three are bound by one contract: the HTTP API. A change to it must reach the
three of them, or one of them breaks silently.

## Decision

**One repository, three applications.**

```
apps/server      Go   — the API, the lifecycle, the store
apps/web         Vue  — the review book
apps/cli         Go   — intake, runner adapters, issue-link reporting   (withdrawn, ADR 0015)
```

The OpenAPI document is authored once under `apps/server/api/src/` and consumed
by the three: the server generates its handler interfaces from it, the CLI
generates its client, the web client generates its typed client. A contract
change that breaks a consumer breaks the build of that consumer, in the same
commit, in the same CI run.

`apps/cli` is built from day one, minimally. It is the only way an edition
enters the system.

## Alternatives rejected

**One repository per artifact.** Rejected: the contract is shared, so version
alignment becomes manual work paid on every API change — a cross-repository
dance to make an incompatibility visible, when a single CI run does it for free.
Three CI pipelines, three release cycles, one team.

**Server and web client only, CLI later.** Rejected: without an intake client
nothing can be pushed, so nothing is testable end to end and the first real
review session waits for a component that was deferred precisely because it
looked secondary. It is the component that produces all the data.

**A single Go binary embedding the built web assets, no separate app.** Not
rejected on merit — this is a *packaging* choice, and it stays open. Embedding
the compiled frontend into the server binary is compatible with this layout;
what is decided here is the source layout, not the number of deliverables.

## Consequences

- Shared Go code between server and CLI (API client, capture hashing, manifest
  types) lives in a common Go module; the two binaries are two `cmd/` entries,
  not two modules.
- CI is gated by path filters: touching `apps/web` never runs the Go pipeline,
  and touching only `docs/` runs neither. A doc-only pull request with no
  backend or frontend check scheduled is intended, not a missing step.
- The repository root holds no application code — only tooling configuration and
  `README.md`.
- All documentation lives under `docs/`, never inside `apps/`
  ([ADR 0009](0009-documentation-strategy.md)).

## Cross-references

- [ADR 0003](0003-runner-and-tracker-agnostic-boundaries.md) — why the runner
  adapter is a client concern, hence why a CLI must exist
- [ADR 0011](0011-technology-stack.md) — what each application is written in
