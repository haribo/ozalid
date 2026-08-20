# ADR 0014 — Server-generated case identity, and a catalogue tree

## Status

Accepted — 2026-08-20

Completes [ADR 0008](0008-no-migration-frozen-predecessor.md), which removed
path-and-title identity without saying what replaces it.

## Context

[ADR 0008](0008-no-migration-frozen-predecessor.md) established that a case
carries a server-side identity, with file path and title becoming ordinary
mutable attributes. It left the operative question open: **what does the client
send** so the server finds the right case on the next edition?

A second question was never asked at all. The predecessor derived its
categories from the e2e folder layout — which is exactly the kind of runner
knowledge [ADR 0003](0003-runner-and-tracker-agnostic-boundaries.md) keeps out
of the server. Two hundred and forty cases in a flat list is not a catalogue.

## Decision

### Identity

**The server generates the identity.** Creating a case takes a title and an
optional description, and returns a short uuid. The client stores that id and
sends it on every subsequent edition.

Title and description are mutable and carry no identity. A short uuid rather
than a sequential integer, so the catalogue's size is not exposed and cases
cannot be enumerated.

The server refuses a manifest where the same case id appears twice — a copied
test that kept its id would otherwise silently corrupt a case.

### Catalogue

**Categories form a tree of unrestricted depth**, per project. A case belongs to
**exactly one** category.

The client creates, renames, moves and deletes categories through the API, and
attaches a case to one. Deleting a category is allowed **only when it is
empty** — no sub-category, no case.

**A case is archived, never deleted.** An archived case leaves the catalogue and
the filters, stops blocking intake, and stays readable with its captures, its
comments and its journal.

## Alternatives rejected

**A stable key written by a human in the test** (`ozalid.case('checkout-guest')`).
Rejected by the decision-maker: a key is generated, never authored. It removes
any naming convention to enforce and any collision to arbitrate, at the cost of
an opaque identifier in the test and one round trip when a case is created.

**Path and title as identity.** Rejected in
[ADR 0008](0008-no-migration-frozen-predecessor.md), on evidence: renaming
invalidated every stored identity and needed a repair script.

**Several categories per case.** Rejected: the catalogue becomes a graph,
"where is this case?" loses a single answer, and no breadcrumb can be drawn.

**Categories derived from the client's folder layout.** Rejected: it is runner
knowledge, and [ADR 0003](0003-runner-and-tracker-agnostic-boundaries.md) keeps
it out of the server.

**Deleting a non-empty category**, with the cases moving up to the parent.
Deferred, not refused: it is a real need, but "delete" would then mean two
different things — removing a filing drawer, and moving what was inside it. It
deserves its own decision, with a confirmation naming exactly what moves and
where.

**Hard-deleting a case.** Rejected: it takes the captures, the comments and the
review journal with it, and destroys the answer to "what did we say about this
screen?". Archiving costs a flag and a clause.

## Consequences

- The client needs a way to create a case and collect its id before the first
  edition — a CLI command, not a manual step.
- The client stores the id somewhere durable. Where is the client's problem;
  ozalid never reads it back.
- Renaming or moving a case is free, and breaks nothing.
- An empty category can be deleted outright, which keeps the common case
  simple; anything else is refused with what blocks it.
- The archive flag enters every catalogue query from the start. Retrofitting it
  later would mean auditing them all.

## Cross-references

- [ADR 0003](0003-runner-and-tracker-agnostic-boundaries.md) — why categories
  are not derived from a folder layout
- [ADR 0008](0008-no-migration-frozen-predecessor.md) — the identity question
  this closes
