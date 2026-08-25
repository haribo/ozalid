# ADR 0017 — A reference belongs to an environment

## Status

Accepted — 2026-08-25

Refines [ADR 0004](0004-content-addressed-capture-storage.md), which required
that captures from different environments never be compared silently, without
saying where that separation lives.

## Context

Freshness answers one question: are the bytes on display still the bytes the
reviewer approved? It is answered by keeping, per square, the content address
that was approved — the *reference*.

The table has held one row per `(case, step, variant)` since the initial
schema. That key assumes every capture of a square comes from the same place.

It does not. Two machines rendering the same screen produce different bytes:
font rasterisation differs, antialiasing differs, a scrollbar is a different
width. Nothing changed in the product, and the images differ anyway.

With one row per square, a suite run on CI and on a developer's laptop writes
to the same reference. Every alternation between the two reports the whole case
as moved. That is an alert that is always on, which is the exact defect
[ADR 0013](0013-a-recording-is-not-a-capture.md) removed `to-re-watch` for.

## Decision

**A reference is keyed by environment**, taken from the capture's own
provenance: `(case, step, variant, environment)`. A capture is only ever
compared against a reference from its own environment.

**A square with no reference in its environment is unjudged there, not moved.**
It has never been approved on that machine, and the interface says so rather
than raising an alarm about a change nobody can see.

**The empty string is the unnamed environment.** A client that declares no
environment still compares against itself, which is the common single-runner
case and must stay free of ceremony.

## Alternatives rejected

**Keeping one reference per square**, letting the last run win. Rejected: the
catalogue flips on every alternation between two runners. An alert nobody can
turn off is an alert nobody reads.

**Refusing captures from an environment the case has no reference for.** Rejected:
it throws away evidence that was produced, and it turns adding a runner into an
intake failure rather than into more coverage. Intake records what happened;
policing the suite is [ADR 0007](0007-run-intake-policy.md)'s job.

**Normalising the images so environments compare** — render-independent hashing,
a perceptual signature. Not rejected on merit; it is a research project with no
result yet, and freshness cannot wait for it. Nothing here forecloses it: it
would collapse the environments back into one key, which is a migration, not a
redesign.

## Consequences

- The reference's primary key grows a column. The table was empty, so there is
  nothing to backfill.
- A reviewer meeting a case captured by an environment they have never seen
  starts from zero on it. That is correct — they have not seen those images —
  but it has to be shown, or it reads as work lost.
- Retention interacts: pruning an edition must not leave a reference pointing at
  a blob that no longer exists, and must not silently strip an environment.
- The differing-pixel count and its per-project threshold
  (`docs/design/product.md` §3.3) are unaffected: they apply within one
  environment, which is the only place they mean anything.

## Cross-references

- [ADR 0004](0004-content-addressed-capture-storage.md) — comparison is only
  meaningful inside one environment
- [ADR 0013](0013-a-recording-is-not-a-capture.md) — why an always-on alert was
  removed once already
- `docs/design/product.md` § 3.3, § 7
