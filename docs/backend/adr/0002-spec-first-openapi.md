# ADR 0002 (backend) — Spec-first OpenAPI, generated on both sides

## Status

Accepted — 2026-08-19

## Context

Three applications share one HTTP contract
([ADR 0010](../../adr/0010-monorepo-with-server-web-and-cli.md)): the server
implements it, the web client and the CLI consume it. There are two ways to
maintain such a contract.

- **Code-first** — write the Go handlers, derive an OpenAPI document from
  annotations.
- **Spec-first** — write the OpenAPI document, generate the Go handler
  interfaces and the client types from it.

Under code-first the document is a by-product. A field renamed in Go regenerates
the document, both consumers keep compiling against their old generated types,
and the break surfaces as a runtime failure on real traffic.

The stakes are specific here. `product.md` § 9 states that **every fact enters
through the API and no path writes state behind it**. The API is not an access
layer over the product — it *is* the product's surface. A contract that can
drift from its implementation undermines the guarantee the whole lifecycle model
rests on.

## Decision

**The OpenAPI document is the source of truth**, and code is generated from it
on every side.

- Authored as several files under `apps/server/api/src/` — one per resource.
- Bundled into `apps/server/api/openapi.yaml`, committed, and **never edited by
  hand**.
- Server: `oapi-codegen` in **strict server mode**. Handlers satisfy a generated
  typed interface; a shape mismatch is a Go compile error.
- CLI: client types generated from the same document.
- Web client: typed TypeScript client generated from the same document.

A contract change is authored once, regenerated, and every consumer that no
longer matches fails to build — in the same commit, in the same CI run.

## Rationale

- A breaking change becomes a compile error in three places instead of a
  production incident in one.
- Strict server mode removes the class of bug where a handler writes a response
  that does not match its declared schema.
- The document being authored rather than derived means it can be reviewed as a
  design artifact: adding an endpoint that sets a case's state would be visible
  in the diff, and `product.md` § 9 forbids exactly that endpoint.

## Alternatives rejected

**Code-first with annotations.** Rejected: the contract becomes a derivative,
consumers cannot trust it, and nothing prevents internal drift.

**Handwritten handlers plus request validation at runtime.** Rejected: drift
surfaces only when traffic hits it. Compile-time checking is strictly stronger
for the same effort.

**gRPC or protobuf.** Rejected: a browser is a first-class client here, and
JSON over HTTP is what it speaks natively. The intake path also carries large
binary bodies, which is plain multipart, not a streaming RPC problem.

**No generation, hand-written clients on both sides.** Rejected: it reintroduces
by hand the duplication the shared document exists to remove — three
transcriptions of one contract, kept in sync by discipline.

## Consequences

- Adding an endpoint means editing `api/src/`, regenerating, and implementing
  the interface the generator produced. The order is not negotiable.
- Editing the bundled `openapi.yaml` by hand is forbidden — it silently desyncs
  from its sources.
- Generated files are committed so that a checkout builds without the code
  generators installed; CI verifies that regenerating produces no diff.
- The document is the contract's review surface: an API change is reviewed there
  before it is reviewed in Go.

## Cross-references

- `docs/design/product.md` § 9 — the API shape and the endpoint that must not
  exist
- [ADR 0010](../../adr/0010-monorepo-with-server-web-and-cli.md) — the three
  consumers
