# ADR 0008 — No migration; the predecessor is frozen

## Status

Accepted — 2026-08-19

## Context

The predecessor holds, as of 2026-08-19: 240 known cases, of which **13** have
actually been reviewed, over four days (14 to 18 August); about 100 MB of
captures and approved references; and a verdict history spanning a handful of
commits.

Its case identities are computed from the test file path and the test title, so
renaming a test invalidates every stored identity — a known weakness with a
dedicated repair script.

Meanwhile the originating project keeps shipping UI changes, and its own rules
require visual validation before merge. Building ozalid will take months.

## Decision

**No migration.** ozalid starts with an empty book. The first client pushes a
run and reviews it from scratch.

**The predecessor is frozen.** It stays exactly as it is, receiving only fixes
that keep it running, until ozalid replaces it. No feature work, no
improvements.

ozalid assigns each case a **server-side identity** independent of file path and
title, which become ordinary mutable attributes.

## Alternatives rejected

**Full migration including history.** Rejected: replaying commits to
reconstruct transitions that were never recorded as such is archaeology, for
four days of data.

**Import current state only** — the 13 reviews and their approved references,
with the journal starting fresh. This was the recommendation, on the grounds
that discarding approved references sends the entire book back to "to review",
including cases that are perfectly in order. Rejected: 13 reviews do not
justify writing and validating a converter.

**Keep improving the predecessor in parallel.** Rejected: it funds two tools to
keep one.

**Retire the predecessor before ozalid is ready.** Rejected: it would leave
the originating project without a visual validation channel for months.

## Consequences

- The one manual step identified in the predecessor — typing an issue number
  into the verdicts file to move a case to `tracked` — **remains** until the
  switch. It is accepted as a known cost of the freeze; a targeted exception
  for that single step is possible but not planned.
- The first review session on ozalid re-reviews the whole catalogue. With 13
  cases actually reviewed today, this is a small loss.
- ozalid never needs to read the predecessor's file formats. No compatibility
  code, ever.
- A note recording the freeze belongs in the originating project's own
  documentation, so a future session there does not start improving a frozen
  tool.

## Cross-references

- [ADR 0001](0001-standalone-multi-project-service.md) — why extraction rather
  than repair
- `docs/design/product.md` §10
