# ADR 0007 — Run intake is governed by a per-project policy

## Status

Accepted — 2026-08-19

## Context

New captures arriving while a review is unfinished is the central tension of
the product.

The predecessor answered it with a **global gate**: while a single case sat
outside `{reviewed, tracked, not-instrumented}`, merging a run into the report
was refused entirely, with the blocking cases listed and a documented override
for emergencies. Running the suite was never gated; only the report was.

That gate exists because there was one copy of each capture: an incoming run
literally overwrote the image under review. Content-addressed storage
([ADR 0004](0004-content-addressed-capture-storage.md)) removes that hazard —
nothing is overwritten, and a case can keep pointing at the edition its
reviewer is judging.

But the gate has a second effect that is not technical: it makes the catalogue
unable to move forward until reviews are finished. That pressure is deliberate,
and the decision-maker wants to keep it.

## Decision

Intake is governed by a **per-project policy**:

- **`strict`** — intake is refused while any case sits outside
  `{reviewed, tracked, not-instrumented}`. The refusal names the blocking
  cases. The originating project runs in this mode.
- **`per-case`** — intake is always accepted and stored; each case keeps
  pointing at the edition under review and advances when that review closes.

Running the test suite is never gated; only intake is. The `strict` override is
itself a recorded fact — an untraceable escape hatch is how a policy quietly
dies.

## Alternatives rejected

**Per-case freezing only.** This was the recommendation: with nothing
overwritable, blocking becomes unnecessary, and a global gate means one
reviewer's unfinished session blocks intake for the whole project — a real cost
once there are several reviewers. Rejected as the sole model, in favour of
keeping the discipline the gate enforces. Retained as a configurable mode,
because that discipline is a project's choice and must not be inherited by
every project the product serves.

**Accept and overwrite with a warning.** Rejected: it is the predecessor's
accident, promoted to a feature.

## Consequences

- The product implements both modes, and the state model must support a case
  pointing at an edition that is not the newest.
- In `strict` mode, an abandoned review blocks the project. The lock expiry of
  [ADR 0005](0005-exclusive-case-locking.md) limits the damage but does not
  remove it; the refusal message must make the blocking cases immediately
  actionable.
- Overrides are journalled and countable. A policy whose override is used
  weekly is a policy that needs revisiting.

## Cross-references

- `docs/design/product.md` §7
