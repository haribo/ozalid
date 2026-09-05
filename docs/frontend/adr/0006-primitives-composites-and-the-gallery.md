# ADR 0006 (frontend) — Primitives, composites, and the gallery

## Status

Accepted — 2026-09-05. Transposes tribnest's frontend ADR-0008 and ADR-0013,
at ozalid's scale, after the user asked to take over those practices (#155).

## Context

Every button, input and empty state was inline Tailwind, copied and drifting.
One review sitting on production surfaced a missing cursor, off-centre
buttons, wrong disabled interplay and label states — each one N call sites to
fix, because nothing said there was one way to make a button.

## Decision

**A primitive is named by its contract and lives in `shared/ui/`.**
`AppButton`, `EmptyState`, `TextInput`: reusable elements rendering a visual
affordance with no feature knowledge. Variants are contracts — `primary /
secondary / destructive / ghost` for actions — never content (`OnlineBadge`,
`PhoneField` are composites' business). The rule of thumb: a name that reads
as content does not belong in `shared/ui`.

**A composite composes; it never reimplements.** No inline `<button class=`,
no ad-hoc dashed box, no arbitrary text size. Each such rule ships with the
lint ban that enforces it — `vue/no-restricted-html-elements` for `button`,
`vue/no-restricted-class` for `text-[` and `border-dashed` — because a rule
without its lock rots one pull request at a time (#145 proved the pattern).
Scoped, justified exemptions are declared where they live: the toggle chips,
the missing mark's dashes (ADR 0016).

**The gallery is the living catalogue.** `/dev/design-system`, dev builds
only: one primitive = one section — title, the line naming its contract, and
a canvas fed the data that breaks it. A test asserts every `shared/ui` export
has its section, so a primitive without a home fails the suite the day it
lands.

**The loop**: mockup → user validation → contract (doc or ADR) → gallery →
revalidation on the real render before merge. It is the same loop every UI
change already follows (`CLAUDE.md`); primitives simply never skip the
gallery step.

## Alternatives rejected

- **Primitives per feature** — the drift returns in six months, one local
  button at a time; the reviewer has nothing to link to.
- **A gallery shipped in production** — a surface with no user; the dev build
  is where the people who need it live.
- **Convention without lint** — tried implicitly for three months; the
  carousel's five-defect review sitting is what it produced.

## Cross-references

- [ADR 0002](0002-fsd-architecture.md) — where `shared/ui` sits in the layers
- [ADR 0004](0004-a-status-is-a-glyph-on-a-disc.md) — the status glyphs the
  gallery displays
- #145, #155, #158, #161, #162 — the locks and slices this anchors
