# ADR 0004 (frontend) — A status is a glyph on a disc

## Status

Accepted — 2026-08-26

## Context

ozalid draws its marks on top of somebody else's product. A capture is a
screenshot of an application whose colours, shapes and iconography were chosen
by other people, and the grid lays verdicts over those pixels.

The first version laid bare glyphs on the captures: a check for validated, a
speech bubble for commented, two arrows for moved. It worked until it was looked
at on real evidence. The demonstration screens carried an indigo primary button;
the moved mark is indigo; the mark landed on the button and became
indistinguishable from it. Nothing in the code was wrong. The rule was.

A first fix scoped the remedy: give the mark a plate only where the capture is
at full strength, since a judged capture is dimmed and `opacity` composites
toward the application's own surface, which does guarantee a ground. That
reasoning holds, and it produced two dialects in one page — some marks plated,
some bare, the reader left to work out why.

## Decision

**A status is a glyph on a disc, everywhere a status is shown**: the case grid,
its legend, the state pills, the category gauges, the comment recap.

**An action is not a status and wears no disc.** Validate, comment, accept,
refuse are gestures; putting a disc on them would say that pressing the button
*is* the state rather than the way to reach it. The two are told apart by shape
before they are read.

**An empty disc is a status.** Nothing has been said about this square yet. It
needs no glyph, and inventing one would name something that has not happened.

## Alternatives rejected

**The disc only where it is earned** — on a capture at full strength, since a
dimmed one already has a guaranteed ground. Rejected: it is defensible pixel by
pixel and indefensible as a language. A rule that stops at one widget is not a
rule, and the reader has no way to learn where it stops.

**Keeping bare glyphs and relying on the dim.** Rejected by the evidence: the
dim guarantees a ground, not a contrast, and a capture whose palette is close to
a verdict's hue defeats it. More plainly, it had already failed once on a real
screen.

**A drop shadow instead of a disc.** Not seriously considered: a shadow is a
depth cue that the rest of this interface does not use, and it would separate the
mark from the capture by pretending both are physical.

## Consequences

- The check loses its enclosing circle — the disc carries the enclosure now — so
  the circle-and-check shape leaves the vocabulary entirely. A retired shape may
  not survive as something else, so it was replaced everywhere it still served as
  an action glyph.
- `StateIcon` draws the disc; `ActionIcon` carries the bare gestures. The split
  is enforced by having two components rather than a flag, so choosing wrong
  means naming the wrong thing rather than passing the wrong argument.
- Six surfaces speak one language, and a reader learns it once.
- A disc is heavier than a glyph at small sizes. Accepted: the sizes where it
  costs most are the ones where a bare glyph was least legible anyway.

## Cross-references

- [ADR 0003](0003-the-grid-answers-one-question.md) — what the grid says, of
  which this is how it says it
- [ADR 0004 (product)](../../adr/0004-content-addressed-capture-storage.md) —
  the captures these marks are laid over
