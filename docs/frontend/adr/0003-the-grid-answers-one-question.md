# ADR 0003 (frontend) — The grid answers one question

## Status

Accepted — 2026-08-26

Edited in place the same day, before any code was written against it: the mark
was specified as sitting at the cell's corner, which is a placement the decision
had no business fixing, and the count of readings fell from seven to six once
*validated and moved* and *commented and moved* proved to be one cell.

## Context

A case's grid puts steps down the page and variants across it, one capture per
cell. Each cell carries a verdict — `to-review`, `to-fix`, `validated` — and,
since freshness landed, whether that capture still shows the bytes a reviewer
approved.

Two facts, one cell. The first attempt showed both: the word `validée` next to
the word `a bougé`, the green ring kept, the verdict's badge dropped. It was
defensible on paper — both facts are true, and freshness is an overlay rather
than a replacement (`product.md` §3.3) — and it read badly.

It read badly because the grid is not a report on a capture. It answers one
question, asked by a reviewer with a list to get through: **what have I
validated, and what have I not?** A capture that has moved has not been
validated — not the bytes on display. Whatever verdict it used to carry does not
change that answer, and printing it next to `a bougé` made the reader join two
marks the interface should have joined for them.

Worse, the two disagreed. A green ring says *settled* in colour while the words
next to it say *moved*. The reader believes the colour.

## Decision

**The grid answers "what have I validated, and what have I not?" and nothing
else.**

**A capture that has moved renders as a capture to judge** — neutral ring, full
opacity — carrying the mark that says why it came back in place of the verdict
badge it used to wear. It is the truth: for the only question the grid asks, it
*is* to judge.

It follows that *validated and moved* and *commented and moved* are one cell,
not two: nothing visible separated them, and the verdict they used to carry is
precisely what the grid no longer reports. A commented capture that has moved
also loses its bubble, and that is not a loss — the comment is about an image
that is gone, so pointing at it would be pointing at something stale.

**No state word under a thumbnail.** The ring and the icon carry the reading. An
icon is a shape, not a colour, and it carries its own accessible name, so
dropping the word costs nothing to whoever cannot tell the ring's hue.

**A capture that has been judged and has not moved steps back** — dimmed, with
its verdict stamped on it. Full intensity is reserved for what still needs eyes,
which is what makes the answer readable at a glance rather than by counting.

Six readings exist, and no others: to judge, validated, commented, moved,
missing, and a recording — which carries no verdict at all because nothing
compares it ([ADR 0013](../../adr/0013-a-recording-is-not-a-capture.md)).

## Alternatives rejected

**Showing the old verdict beside `a bougé`.** Rejected: two marks of unequal
weight make the reader do the joining. The information it seemed to preserve —
that a comment may now be about an image that is gone — is not carried by the
word `commentée` anyway; it is carried by the comment, which lives in the recap
and in the carousel.

**Keeping the verdict's ring colour and dropping only the word.** Rejected: the
colour would say *settled* while the mark says *moved*, and colour wins that
argument every time.

**Keeping the word and letting the verdict recede to grey.** Built, looked at,
rejected: it fixes the shouting but not the joining. The cell still reports two
things where the grid asks one.

## Consequences

- A commented capture that has moved is indistinguishable, in the grid, from a
  validated one that has moved, and from one never judged. Accepted: "where were
  my comments?" is answered by the recap below the grid and by the carousel in
  front of the image. The grid is not where that question belongs.
- The merging stops there. *Validated* and *commented* stay apart on a capture
  that has **not** moved: the bubble points at a comment that is still true and
  sits just below. On a capture that has moved it would point at nothing, which
  is the difference — not the verdict, but whether the mark still designates
  something exact.
- The grid needs a legend, since the words are gone. One place to learn the
  language, rather than a translation repeated under every cell.
- `to-review` combined with *moved* cannot occur: a reference is only stamped
  when a reviewer validates, so a capture never validated has nothing to compare
  against ([ADR 0017](../../adr/0017-a-reference-belongs-to-an-environment.md)).
- The carousel is unaffected. It is where a capture is judged rather than
  counted, so it says everything: the verdict, the comment, whether an issue was
  delivered, and how many pixels moved.

## Cross-references

- [ADR 0012](../../adr/0012-case-carries-the-ball-comment-carries-the-detail.md)
  — the same instinct one level up: a state answers one question, the detail
  lives elsewhere
- [ADR 0017](../../adr/0017-a-reference-belongs-to-an-environment.md) — why a
  capture may have nothing to compare against
- `docs/design/product.md` § 3.3
