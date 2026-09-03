# ADR 0005 (frontend) — The carousel has an address

## Status

Accepted — 2026-09-03

## Context

The carousel is where a capture is judged, and it was the one screen that could
not be pointed at. It rendered below the grid, inside the page's column, at a
width written in advance — `w-[240px]` portrait, `w-[560px]` landscape — which
showed a 1280 px capture at 44 % on the screen whose whole job is pixels.
Production use called it out as unusable, and "look at step 2 in dark" could
not be sent to anybody: which capture was open lived in a `ref`, not in the
URL, on an instance where everything else worth showing has had an address
since the URIs were reworked (#71).

One constraint made the obvious fixes wrong. The case page holds the verdict
withheld when a session expires mid-judgment (#70, `useReview.ts`): whatever
opens the carousel must not unmount that page, or the verdict dies with it.

## Decision

**The open capture is named by the route**:

```
/projects/:slug/cases/:caseId/steps/:stepId/variants/:variantId
```

Both this route and the case's own resolve to the **same component**. Vue
reuses the instance when the component type does not change, so opening and
closing a capture navigates without unmounting — the review state, including a
held verdict, survives. The carousel renders over the page (`fixed inset-0`),
which then keeps everything it was holding underneath.

The capture takes the window and is **never stretched**: at most its natural
size, since blown-up pixels are falsified pixels. Arrow keys walk by
`router.replace`, so the back button means the grid rather than a retrace of
every square looked at; `Esc` closes by pushing the case URL, because a link
opened straight onto a capture has no history behind it.

## Alternatives rejected

**A modal.** No address, so no way to send somebody to a square; no back
button; and an overlay that duplicates what routes already do.

**A separate page.** The honest-looking version of the route — and the one
that unmounts the case page and silently drops a held verdict. Rejected for
exactly the reason the child arrangement exists.

## Consequences

- The instance-reuse property is load-bearing and guarded: forcing a remount
  (`:key="$route.fullPath"` on the RouterView) fails three end-to-end tests,
  including the resumed-verdict walk.
- The carousel declares `role="dialog"`, and tests find it by that rather than
  by a style utility — the previous hook (`.rounded-lg`) died in this change.
- A direct load of a carousel URL must land signed-in on that exact square;
  the end-to-end suite asserts it.

## Cross-references

- [ADR 0003](0003-the-grid-answers-one-question.md) — the grid shows, the
  carousel judges
- `docs/adr/0002` (#70) — the withheld verdict this layout exists to keep
- #71 — everything worth showing has an address; this closes the exception
