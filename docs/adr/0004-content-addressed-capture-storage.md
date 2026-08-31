# ADR 0004 — Content-addressed capture storage, with provenance

## Status

Accepted — 2026-08-19

## Context

The predecessor kept **only the latest** bytes of each capture, plus a local
copy of the approved bytes as a reference. Measured on the predecessor: 75 MB of
current captures and 27 MB of references, for 240 cases with partial
instrumentation — about 100 MB for a single edition.

Two consequences followed. There was no visual history: comparing a screen to
its state three months ago was impossible. And a new run **physically
overwrote** the image a reviewer was judging — which happened, and produced a
one-shot repair script to restore approved bytes over corrupted ones.

Keeping every edition naively would multiply that 100 MB by the number of
editions — gigabytes per year, per project — to store overwhelmingly identical
images, since between two runs the vast majority of captures are byte-for-byte
the same.

Separately: byte comparison only means something within one rendering
environment. The predecessor documented this as an accepted limitation — a
browser update caused a mass false alarm and a full re-review. A hosted service
receiving captures from several machines makes that permanent rather than
occasional.

## Decision

**Captures are stored by content hash.** An identical image is stored once,
however many editions reference it. Editions and references are pointers.

**Every capture records its provenance**: operating system, browser and
version, and a client-declared environment identifier. Captures from different
environments are never compared silently — the comparison is refused, or the
delta is flagged as environment-induced.

## Alternatives rejected

**Latest only, as before.** Rejected: no history, and it is the mechanism by
which a run destroys a review in progress.

**Keep every edition in full.** Rejected: gigabytes per project per year to
store duplicates.

## Consequences

- Full visual history for roughly the cost of what actually changes.
- Nothing is ever overwritten, so a reviewer's evidence cannot be destroyed by
  an incoming run. This is what makes the per-case intake policy possible at
  all ([ADR 0007](0007-run-intake-policy.md)), and it retires the repair script
  by construction.
- Approved references become pointers rather than copied files, and they stop
  being machine-local.
- Cheap is not free. A retention rule is still needed and is left open in
  `docs/design/product.md` §11.
- Provenance is a schema concern, not a client convention: the server must be
  able to refuse a meaningless comparison on its own.

## Cross-references

- `docs/design/product.md` §4
