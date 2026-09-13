# ADR 0025 — A case reads at an edition that captured it

## Status

Accepted — 2026-09-13

Amends [ADR 0024](0024-the-pin-follows-the-lock.md): the fallback it named —
the project's latest edition — becomes the latest edition **of that case**.

## Context

ADR 0024 made the displayed edition derived rather than stored: a live lock
pins the edition stamped at claim, and otherwise the case reads at the
project's latest edition.

That fallback assumes every run covers the whole book. Nothing states the
assumption and nothing enforces it: intake accepts a manifest naming two cases
as readily as one naming forty (`product.md` §7), and a client with a targeted
suite has no reason to send more.

When a run skips a case, the case keeps its cycle state — it is the server's,
computed from facts, and no fact changed — so the catalogue counts it and the
review queue promises it (§3.6). Its grid, read at an edition that never
captured it, is empty. Observed against `882b301`: `state="to-review"`,
`gridSteps=0`, `inQueue=0` (#253).

The same hole sat one layer deeper: the claim stamped the project's latest
edition too, so opening such a case pinned the emptiness for the whole
session.

## Decision

**A case is never read at an edition that did not capture it.** Wherever the
server answers "which edition", the answer is the latest edition holding at
least one capture of that case:

- the free case's display;
- the edition a claim stamps onto the lock;
- the edition a delivery re-stamps it onto;
- the edition a settling review re-derives against.

A live lock still wins over all of it — freezing the bytes under a reviewer is
ADR 0024's decision and stands. A case no edition ever captured still reads
empty: `not-instrumented` is a legitimate state (ADR 0012), and an empty grid
is the honest answer to a case outside the funnel.

## Consequences

- A partial run stops blanking the rest of the book. What a case shows changes
  only when a run captures that case.
- Two cases on one screen may come from different runs. That was already true
  under a lock (ADR 0024) and is the price of not hiding evidence.
- Completeness (§3.4) is unaffected: it is measured at an edition, against
  what that edition declared for that case.
- The queue's count stops promising captures no screen can show (#204, #205).

## Rejected alternatives

**Recomputing a skipped case's state at intake**, so it stops saying the
reviewer is needed. Rejected: the reviewer *is* needed — a capture nobody
judged is still unjudged, and a run that did not look at it changes nothing
about it. Silencing the case would lose the work rather than show it.

**Declaring that a run must cover the whole book**, and refusing a partial
manifest. Rejected: intake accepts holes rather than discarding the evidence a
run did produce (ADR 0016), and the same reasoning applies a level up. It
would also break every client with a targeted suite for a rule the product
never needed.

**Leaving it, and fixing the count instead** — the queue skipping cases whose
evidence it cannot show. Rejected: the count would be honest about a book that
is not. The evidence exists, content-addressed and never overwritten
(ADR 0004); refusing to show it is the defect.

## Cross-references

- [ADR 0024](0024-the-pin-follows-the-lock.md) — the pin follows the lock;
  amended here
- `docs/design/product.md` §7 — intake policy, and what a free case reads
- [ADR 0004](0004-content-addressed-capture-storage.md) — nothing is ever
  overwritten, which is what makes the older edition still readable
