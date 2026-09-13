# 0007 (frontend) — A walk resolves to the page it started from

## Status

Accepted — 2026-09-13

## Context

[ADR 0005](0005-the-carousel-has-an-address.md) gave the open capture an
address and made both the case route and the carousel route resolve to the
**same component**, so Vue reuses the instance: opening and closing a capture
navigates without unmounting, and a verdict withheld through an expired
session (#70) survives.

The review queue (`docs/design/product.md` §3.6, #205) walks captures across
cases in one sitting. Its address cannot sit under a case — the case changes as
the reviewer walks — and it is entered from the catalogue, not from a case.
Applied naively, a route of its own would mount a new page and drop whatever
the catalogue was holding, which is the failure ADR 0005 exists to prevent, one
screen over.

## Decision

**A walk is addressed, and its route resolves to the page it was started
from.**

```
/projects/:slug/queue/cases/:caseId/steps/:stepId/variants/:variantId
/projects/:slug/categories/:categoryId/queue/cases/:caseId/steps/:stepId/variants/:variantId
```

Both resolve to `CataloguePage`, as the carousel's route resolves to
`CasePage`. Two addresses and not one, mirroring the catalogue's own two: the
walk's reach is the depth it was started from, so the scope is in the path
rather than in a ref that a reload would lose.

**The entries are fixed for the length of the walk.** The queue is read when
the catalogue is read, and recomputing it after each verdict would move the
list under the reviewer — the walk they started is the walk they finish. It is
read back on the way out, so the catalogue says what is left rather than what
was there on entry.

**The walk claims and releases as it goes.** Entering a case claims it with a
fresh claim before the first read, so the hold pins what is current
([ADR 0024](../../adr/0024-the-pin-follows-the-lock.md)); leaving it releases.
One lock at a time, held by the case on screen.

## Alternatives rejected

**A page of its own.** Rejected: it unmounts the catalogue, and ADR 0005's
whole point is that the page underneath keeps what it holds. The property is
not about the case page in particular — it is about not destroying the page a
verdict is being written over.

**A second carousel component.** Rejected: it would fork accept, unaccept,
refuse, unrefuse, judge, unjudge, draft editing and the keyboard map. The walk
is a second **source of navigation** handed to the one carousel, not a second
carousel.

**Keeping the walk in a ref, without an address.** Rejected for the reason
ADR 0005 already gave: a capture worth judging is worth pointing at, and a
reload mid-walk would lose the reviewer's place.

## Consequences

- The wiring between the carousel's verdicts and the session that writes them
  is extracted (`ReviewCarousel`), since two pages now open the carousel and a
  second copy of those handlers would drift.
- The queue walk's counter is the queue's position, not the flow's — the
  reviewer is not asking where they are in a case.
- The catalogue holds a review session, which it did not before: it claims,
  heartbeats and releases while a walk is open.

## Cross-references

- [ADR 0005](0005-the-carousel-has-an-address.md) — extended here to a second
  host page
- `docs/design/product.md` §3.6 — what the queue is and what it holds
- [ADR 0024](../../adr/0024-the-pin-follows-the-lock.md) — the edition a hold
  pins
