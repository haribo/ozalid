# ADR 0015 — No CLI; clients call the API directly

## Status

Accepted — 2026-08-21

Partly supersedes [ADR 0010](0010-monorepo-with-server-web-and-cli.md), which
made the CLI one of the repository's three applications. The monorepo layout it
decided stands; only the third application is withdrawn.

## Context

[ADR 0010](0010-monorepo-with-server-web-and-cli.md) argued that a CLI was
structurally necessary: "[ADR 0003](0003-runner-and-tracker-agnostic-boundaries.md)
puts the runner adapter and the tracker knowledge outside the server, on the
client side. Without the CLI, nothing can push an edition."

The premise is right and the conclusion does not follow. ADR 0003 says
translating a runner's output is the **client's** job. It does not say ozalid
must ship a program that does it — it says the opposite of shipping runner
knowledge in the product.

What a client actually needs to fill the book is now built and proven:

- `HEAD /blobs/{hash}` to learn what the store already holds
- `PUT /blobs/{hash}` for what it does not
- `POST /projects/{slug}/editions` with the manifest

A Playwright teardown does that in a few dozen lines, in the language the suite
is already written in. A CLI would wrap those three calls in a binary that has
to be built for every platform, released, versioned, and kept in step with the
contract — while the contract is already published as an OpenAPI document any
client can generate from.

`apps/cli` has been an eighteen-line shell since the skeleton and has never done
anything. Keeping it reserved a name and a slot in the CI for work nobody had
asked for.

## Decision

**ozalid ships a server and a web client. Nothing else.**

Clients call the API. The manifest shape, the content-address format and the
hashing algorithm are published in the OpenAPI document, which is what a client
generates from — the same source the server implements.

`apps/cli` is deleted.

## Alternatives rejected

**Keep the CLI and fill it in.** Rejected: it buys convenience nobody has asked
for, at the cost of a second binary to release and keep current. The moment a
real client exists and finds the raw calls painful, this can be revisited with
evidence instead of anticipation.

**Ship a client library instead**, in TypeScript for Playwright users.
Rejected *for now*, not on merit: it is a better answer than a binary, since it
lives in the client's language and needs no distribution channel. But it is
decided when someone is writing that integration, not before — and it belongs
in the client's repository, not necessarily in this one.

**Keep the empty package as a placeholder.** Rejected: an empty package that
does nothing still appears in the layout, the CI filters and the justfile, and
reads as a commitment. Deleting it is free; git remembers.

## Consequences

- Two applications, not three. CI path filters and the justfile lose a target.
- `internal/contract` keeps its manifest and hashing types, but they are the
  server's own. The claim that they are "shared by the server and the CLI" is
  removed — the shapes are shared through the OpenAPI document now, which is
  where a second implementation reads them.
- The hashing algorithm is still a contract, and still changes only through a
  new ADR ([ADR 0004](0004-content-addressed-capture-storage.md)): a client
  computing addresses differently would re-upload the whole catalogue.
- Nothing about the product's boundaries changes.
  [ADR 0003](0003-runner-and-tracker-agnostic-boundaries.md) holds exactly as
  written: the server knows no runner, and the translation happens on the
  client side — wherever the client chooses to put it.

## Cross-references

- [ADR 0003](0003-runner-and-tracker-agnostic-boundaries.md) — the boundary this
  respects, and whose reading ADR 0010 stretched
- [ADR 0010](0010-monorepo-with-server-web-and-cli.md) — the record this partly
  supersedes
