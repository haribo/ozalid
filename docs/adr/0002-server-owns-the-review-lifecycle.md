# ADR 0002 — The server owns the review lifecycle

## Status

Accepted — 2026-08-19

## Context

A case moves through a cycle: never judged, judged with defects, taken over by
issues, clean. The question is where that state lives and who decides it.

The predecessor **derived** it: nothing was stored, the state was recomputed
from the current facts every time the report was generated. That model has one
strong property — the state can never contradict the facts, because it *is* the
facts — and three real costs, which drove this decision:

1. **The past is unrecoverable.** Yesterday's facts were overwritten by today's,
   so yesterday's state could not be reconstructed. No transition was dated, no
   actor recorded. "When did this case become tracked, and because of what?"
   had no answer.
2. **A rule change silently rewrites history.** Adjust the computation and every
   case in the catalogue may change meaning at once, including cases nobody
   touched. There is no way to detect it, because there is nothing to compare
   against.
3. **Every read is a computation.** Answering "which cases are reviewed?"
   means walking the whole set. At 240 cases this is immeasurable and was
   argued as such; as an argument about *simplicity of querying* rather than
   speed, it stands.

The opposite extreme — a client that sets the state — was never acceptable: it
allows a case marked `reviewed` while carrying open defects, with no way to
know when the two diverged.

## Decision

**State is stored, and only the server writes it.**

1. Every case carries its current cycle state as data, readable and filterable
   without recomputation.
2. Every transition is appended to a journal: `{case, from, to, at, actor,
   cause, inputs}`. Nothing is edited in place; the current state is the last
   entry.
3. The server **computes** each transition from the facts it just recorded. The
   API exposes no endpoint that accepts a state as an argument. A client
   reports a fact — a review was saved, an issue was attached, a problem was
   discarded — and the server decides what that means.
4. `inputs` records a fingerprint of the facts the computation consumed: open
   problems, their external references, capture references.

Point 4 is a condition of the decision, not a detail. A stored state is only a
regression oracle if a replay can be attributed: when the rule changes and the
replayed result differs, `inputs` is what distinguishes a code regression from
data that legitimately moved. Without it, the oracle produces false alarms and
gets ignored — and the main argument for storing state evaporates.

## Alternatives rejected

**Derive on read, store nothing** (the predecessor's model, and the position
initially defended). Rejected on the history and regression-oracle arguments
above, which derivation cannot answer at any price.

**Store the state and let clients set it.** Rejected: it is the only design in
which state and facts can disagree, and the disagreement is undetectable.

**Derive on read but keep an append-only journal of observed changes.** A
middle road that was not formally put: it answers the history argument but not
the querying one, and still recomputes on every read. Storing is simpler once
the journal exists anyway.

## Consequences

- Adding a feature means adding a **fact** and teaching the computation what it
  implies — never adding a transition for each caller to remember. This is the
  guard against the state machine sprawling as the product grows.
- The computation lives in exactly **one** place. The predecessor's five copies
  of a single rule are the failure mode this exists to prevent.
- Each recorded fact must state its actor. Machine actions are distinguishable
  from human ones by construction.
- The journal is the audit trail. The predecessor relied on the git history of
  a committed file for this; a hosted service has no such fallback, so the
  journal must be a first-class, queryable, exportable feature — not a log
  file.

## Cross-references

- `docs/design/product.md` §3 — the state axes and the facts that trigger
  transitions
- [ADR 0005](0005-exclusive-case-locking.md) — why occupancy is a separate axis
- [ADR 0006](0006-problems-are-durable-entities.md) — the facts themselves are
  durable too
