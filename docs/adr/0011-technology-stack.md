# ADR 0011 — Technology stack

## Status

Accepted — 2026-08-19

Settles the technology-stack question left open in
`docs/design/product.md` § 11.

## Context

The specification was written to depend on no particular technology, which is
why this decision was left last. It cannot be left open any longer: nothing can
be built without it.

What the product actually demands of a stack, read from the specification:

- **A journal that is append-only and queryable** — every transition, with its
  actor, its cause and a fingerprint of its inputs
  ([ADR 0002](0002-server-owns-the-review-lifecycle.md)). This is the load
  bearing requirement, and it is a database requirement.
- **Filterable stored state** — `GET /cases?state=…&freshness=…` answers without
  scanning (`product.md` § 9).
- **Large binary payloads on intake**, hashed and deduplicated
  ([ADR 0004](0004-content-addressed-capture-storage.md)).
- **A server-sent event stream**, held open per connected client (`product.md`
  § 9), which means many long-lived idle connections.
- **A dense, stateful review UI** — a grid of captures across steps and
  variants, keyboard-driven, with image comparison.
- **A distributable client binary** with no runtime to install on a CI machine.

One more constraint, and it is not technical: this is a single-maintainer
product. A stack the maintainer already knows deeply is worth more than a
marginally better-suited one they would learn on the job.

## Decision

**Backend and CLI in Go. Web client in Vue 3 + TypeScript. PostgreSQL for
state. Captures on the filesystem. OpenAPI as the contract between them.**

| Layer | Choice |
| --- | --- |
| Server, CLI | Go |
| HTTP contract | OpenAPI 3, spec-first ([backend ADR 0002](../backend/adr/0002-spec-first-openapi.md)) |
| State | PostgreSQL ([backend ADR 0003](../backend/adr/0003-postgres-stack.md)) |
| Capture bytes | Filesystem, content-addressed ([backend ADR 0004](../backend/adr/0004-filesystem-blob-store.md)) |
| Web client | Vue 3 (script setup) + TypeScript + Vite + Tailwind ([frontend ADR 0001](../frontend/adr/0001-frontend-stack.md)) |
| Tests | Go stdlib testing; Vitest; Playwright |
| Task runner | `just` |

The detailed sub-decisions — layering, code generation, persistence tooling,
frontend architecture — each get their own ADR under `docs/backend/adr/` and
`docs/frontend/adr/`. This record decides the frame only.

## Rationale

**Go for the server.** Long-lived idle connections are the SSE stream's whole
cost model, and goroutines make thousands of them cheap without an async
runtime. Streaming multipart intake to disk while hashing it is standard library
work. Static binaries remove the deployment runtime from the operations budget
— relevant for a product whose hosting story is still open (`product.md` § 11).

**Go for the CLI, and the same language for both.** A test-runner adapter must
run on any CI machine with no runtime installed; a single static binary is the
whole distribution story. Sharing the language with the server means the
manifest types, the hashing rule and the API client are written once
([ADR 0010](0010-monorepo-with-server-web-and-cli.md)).

**Vue 3 for the web client.** The review surface is a stateful grid — the
composition API handles that with less ceremony than the alternatives, and the
maintainer's fluency is the deciding factor. This is an honest reason, not a
technical superiority claim: at this scale no mainstream framework would fail at
the task, and the one that ships fastest is the one already known.

**PostgreSQL.** The journal is the product's audit trail and must be queryable,
exportable and transactional: a fact and the transition it causes are recorded
together or not at all. `inputs` fingerprints are semi-structured, which JSONB
holds natively without a second store. `LISTEN`/`NOTIFY` feeds the SSE stream
without introducing a broker.

**OpenAPI as the contract.** Three consumers share one API
([ADR 0010](0010-monorepo-with-server-web-and-cli.md)). Making the document the
source, and generating from it, turns a contract break into a compile error in
every consumer rather than a runtime failure in one.

## Alternatives considered

**TypeScript end to end (Node server).** One language, shared types with no
generation step, and the largest ecosystem. Rejected: the CLI would ship a Node
runtime or a bundled binary onto CI machines, and the server would trade
goroutines for an event loop on exactly the workload — many idle open
connections plus streaming uploads — where Go is simplest. The single-language
argument is real but the two boundaries it simplifies are already covered by
code generation.

**Rust for the server and CLI.** Better on paper for the hashing and streaming
path. Rejected: compile-time cost and ecosystem friction on ordinary web
plumbing buy nothing here — the bottleneck is disk and network, not CPU — and
the maintainer would be learning while designing.

**SQLite instead of PostgreSQL.** Genuinely tempting: a single file, no service
to operate, and the journal is append-only. Rejected on
[ADR 0001](0001-standalone-multi-project-service.md) — a hosted service with
concurrent reviewers and exclusive locks
([ADR 0005](0005-exclusive-case-locking.md)) means concurrent writers, which is
where SQLite's single-writer model turns into a product constraint. It stays the
right answer for a single-user tool; that is the product ozalid is replacing.

**Object storage (S3-compatible) for captures from v1.** Rejected for v1 and
only for v1: it adds a service, credentials and slower tests before there is any
deployment requiring it. The decision is isolated behind a port
([backend ADR 0004](../backend/adr/0004-filesystem-blob-store.md)) so it costs
one adapter to revisit.

**React for the web client.** Rejected on the same grounds Vue was chosen: no
decisive technical advantage at this scale, and a lower fluency.

**Server-rendered HTML with no SPA framework** (Go templates plus a light
interactivity layer). Not unreasonable for a catalogue, and it would remove the
frontend build entirely. Rejected: the review surface is stateful — a lock
heartbeat, an event stream, per-cell verdicts accumulated before a single save,
image comparison — which is application behaviour, not page navigation.

## Consequences

- Every open question in `product.md` § 11 that depended on the stack can now
  be settled; the stack question itself is closed by this record.
- The OpenAPI document becomes the artifact under the strictest discipline in
  the repository: it is the only place where all three applications meet.
- Two toolchains and two lint pipelines enter scope, and the CI is gated by path
  so that neither runs when it is not concerned
  ([ADR 0010](0010-monorepo-with-server-web-and-cli.md)).
- Replacing any of these choices is a project, not an edit. Each is isolated
  behind its own ADR so the blast radius is known before it is paid.

## Cross-references

- `docs/design/product.md` § 11 — the question this closes
- [ADR 0010](0010-monorepo-with-server-web-and-cli.md) — the layout this fills
- `docs/backend/adr/`, `docs/frontend/adr/` — the sub-decisions
