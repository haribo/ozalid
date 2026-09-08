# 0020 — A remark is a verdict

Date: 2026-09-06
Status: Accepted — settling clause superseded by [ADR 0022](0022-a-judgment-lands-on-the-capture-on-screen.md)

## Context

The first review rounds on ozalid itself (issues #170, #171) exposed three
frictions. The reviewer had two unrelated grammars — validate/unvalidate on a
bare capture, accept/refuse on a delivered fix — plus a third act, commenting,
that blocked the case without saying so. The word `validated` described a
capture while the act on an issue was called *accept*. And the reviewer had to
classify every comment as `defect` or `improvement`, a qualification that was
then redone by whoever wrote the issues.

## Decision

Reviewing a capture is giving one of two verdicts: **accept** or **refuse** —
on a bare capture exactly as on a delivered fix.

A remark exists only inside a refusal. Commenting without refusing does not
exist, and every remark blocks the case until it is settled.

The comment's kind (`defect` / `improvement`) is removed. Qualifying a remark
into a fix or a feature belongs to whoever writes the issues, never to the
reviewer.

**Rejected alternative — the non-blocking comment.** It made the reviewer
classify work they do not own, and it produced "commented but validated"
captures that nobody was accountable for.

## Consequences

- `validated` leaves the product. Captures and refs read `accepted` /
  `refused`; the grid, its counters, the stored states and the API enums say
  the same word as the act.
- The verdict pair is one segmented control, always visible; the filled half
  is the state. Both verdicts can be reconsidered, and switching verdicts
  never passes through "not judged".
- A draft remark — one no issue is attached to yet — is the reviewer's own:
  it can be edited, and taking the draft refusal back withdraws it. This is
  the explicit exception to ADR 0006's "nothing is deleted", scoped to the
  reviewer's own drafts; anything an issue was written from is kept forever.
- The carousel never marks the capture: the verdict lives in the bar, the
  grid keeps its marks.
