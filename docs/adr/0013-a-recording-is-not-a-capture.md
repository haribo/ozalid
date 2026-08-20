# ADR 0013 — A recording is not a capture

## Status

Accepted — 2026-08-20

Amends `docs/design/product.md` § 2, which defined the flow recording as a
capture, and § 3.3, which carried a `to-re-watch` freshness value.

## Context

`product.md` § 2 declared the video "reviewed per variant like any other
capture". That single sentence is the origin of a defect that ran through the
whole freshness model.

A capture has one property everything else rests on: it can be compared. Two
runs producing the same screen produce the same bytes, so the server can prove
that the evidence a reviewer approved is still the evidence on display.

A video has no such property. Encoding is not deterministic — two identical
runs produce different bytes. Treating it as a capture therefore required a
freshness value the server could not compute from what it holds:
`to-re-watch`, defined as "the captures are byte-identical but the code moved".

"The code moved" is not something ozalid can observe. It has to be declared by
the client, and the predecessor declared it as the repository's commit hash —
so any commit at all, on any part of the product, flipped every unchanged case
in the catalogue to "go watch this again". An alert that is always on is an
alert nobody reads.

The deeper rule being broken is the one [ADR 0002](0002-server-owns-the-review-lifecycle.md)
depends on: a state must be computed from facts the server can verify.
Otherwise `inputs` cannot serve as a regression oracle, because a replay cannot
distinguish a code change from a client that changed its declaration.

## Decision

**A capture is comparable evidence.** Hashed, referenced, the source of
freshness.

**A recording is a supporting exhibit.** Attached to an edition, viewable,
**optional**, never hashed for comparison, never referenced, never a source of
state. A case without a recording is a normal case, not an incomplete one.

**Freshness has two values, both provable from bytes**: `current` and
`to-re-review`. `to-re-watch` is removed.

A capture is deemed changed when its hash differs **and** a bounded pixel
comparison exceeds a per-project threshold — below it, the difference is
rasterisation noise. The number of differing pixels is recorded, so the
threshold can be judged rather than guessed.

## Alternatives rejected

**Keeping `to-re-watch` with a client-declared revision**, leaving the client
free to choose the granularity. Rejected: the product would still compute a
state from an unverifiable declaration, and a client that gets the granularity
wrong silently poisons its whole catalogue.

**Making the video comparable**, by hashing decoded frames rather than the
encoded file. Not rejected on merit — recording timing is not deterministic
either, so it needs a perceptual signature and a real experiment. It is not
excluded later; it is excluded now.

**Dropping the video entirely.** Rejected: it shows what happens *between* the
captured moments, which no still image carries.

## Consequences

- Nothing in the cycle depends on information the server cannot check.
- A change to an animation, invisible on the captured frames, no longer raises
  anything by itself. Accepted: the alternative was an alert that never turned
  off. The recording is watched again whenever an image moves, since the case
  returns to review anyway.
- The dedup economics of [ADR 0004](0004-content-addressed-capture-storage.md)
  apply to images only. A recording is stored per edition, and retention has to
  account for it separately.
- An edition may still carry a client-declared revision. It is **displayed**,
  never computed on — an informational value may be imprecise, a state may not.

## Cross-references

- [ADR 0002](0002-server-owns-the-review-lifecycle.md) — a state is computed
  from verifiable facts
- [ADR 0004](0004-content-addressed-capture-storage.md) — what content
  addressing does and does not buy here
- `docs/design/product.md` § 2, § 3.3
