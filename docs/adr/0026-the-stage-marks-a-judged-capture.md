# ADR 0026 — The stage marks a judged capture

## Status

Accepted — 2026-09-14

Supersedes the last consequence of
[ADR 0020](0020-a-remark-is-a-verdict.md): *"The carousel never marks the
capture: the verdict lives in the bar, the grid keeps its marks."*

## Context

ADR 0020 kept every mark off the stage so the capture shows at full strength:
the surface where pixels are judged should not tint, veil or cover the pixels
being judged.

Walking a case with the arrow keys shows what that costs. The stage says
nothing about whether the capture on screen has already been judged — the
verdict sits in the bar at the bottom, outside the eye's path during a fast
walk — so the reviewer re-reads captures they had already settled. The grid
solved the same problem long ago: a settled thumbnail steps back to
`opacity-40` and wears its status glyph, and full intensity is reserved for
what still needs eyes (frontend ADR 0003, ADR 0004).

The stage was the one surface refusing that language, which made the product
answer "has this been judged?" in two different ways depending on where the
reviewer stood.

## Decision

**A settled capture is marked on the stage exactly as it is in the grid**: the
image steps back to `opacity-40` and its status wears a disc, centred
(frontend ADR 0004). One language for one question, on every surface that
shows a capture.

- **The mark reads the capture's status, never the verdict pair.** The pair
  answers a different question — it goes empty while a delivered fix awaits
  judgment — and a mark taken from it would make the stage contradict the grid
  about the same capture.
- **A `moved` capture keeps full strength**, no veil and no disc: it needs eyes
  again, and the badge already on the stage says why it came back.
- **The recording view is untouched.** A veil over a playing video, with a
  glyph where the browser puts its play control, degrades the one act that view
  exists for.

## Rejected alternatives

**The disc alone, without the veil.** It answers the question the reviewer
actually asks — *has this been judged?* — and leaves the pixels intact, which
is what ADR 0020 was written to protect. Rejected because it makes the stage a
third dialect: judged reads as pale-plus-disc in the grid, the recap and the
legend, and one surface saying it differently is the divergence the design
system exists to prevent. A capture that is settled is not a capture being
judged; ranking it below what still needs eyes is the same statement here as
on the grid, made about one image instead of twelve.

**Keeping the stage bare, as ADR 0020 had it.** Rejected: it is the status quo,
and it makes the reviewer carry in their head what the screen could say.

## Consequences

- **Re-reading a judged capture at full strength means taking its verdict
  back.** The veil lifts with the verdict, so looking costs a write. Accepted
  as the price: the capture is one click from being reconsidered, and the pair
  is always on screen.
- `INK`, `TONE`, `LABEL` and `settled` stop being private to the grid and move
  to `shared/lib`, since a widget may not import another widget (frontend
  ADR 0002) and a second copy is how two surfaces drift apart.
- The stage keeps its own `moved` badge, which now reads as the one mark that
  means *look again* rather than *already looked at*.

## Cross-references

- [ADR 0020](0020-a-remark-is-a-verdict.md) — the consequence superseded here
- `docs/frontend/adr/0004-a-status-is-a-glyph-on-a-disc.md` — why the mark is a
  disc
- `docs/frontend/adr/0003-the-grid-answers-one-question.md` — where the
  language comes from
