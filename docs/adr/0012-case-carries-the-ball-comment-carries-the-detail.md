# ADR 0012 — The case carries the ball, the comment carries the detail

## Status

Accepted — 2026-08-20

Supersedes the six-state cycle described in `docs/design/product.md` § 3.1 as
originally written.

## Context

[ADR 0002](0002-server-owns-the-review-lifecycle.md) settled that the server
computes a case's state from the facts it recorded. It did not settle **how
much** that state should say.

The first attempt said too much. Designing the lifecycle in detail produced a
state per situation: one for a defect awaiting triage, another for an
improvement, another for a delivery awaiting judgment, another for a fix that
was refused. Each looked justified alone. Together they broke on a case that
carries several comments at once — which is the normal case, not the edge case.

A case with an untriaged defect *and* a delivered fix awaiting judgment holds
two balls. A single state field holds one. Whichever wins, the other
disappears from the list, and the person it concerned never learns there was
something for them.

The diagnosis is that the state was being asked to carry information that
belongs one level down. A case is an aggregate of comments; the detail of what
is happening lives on each comment, not on their sum.

## Decision

**A case state answers one question: who holds the ball.** Four values, and
nothing else.

| State | Condition |
| --- | --- |
| `not-instrumented` | No capture, no verdict. Outside the funnel. |
| `to-review` | Something is waiting for the reviewer's judgment. |
| `to-fix` | Nothing awaits the reviewer, and at least one comment awaits the dev. |
| `reviewed` | No open comment. |

`to-review` wins when both apply: a verdict can cancel work in progress, so it
comes first.

**A comment carries its own lifecycle**, and that is where the detail lives.

| State | Meaning | Terminal |
| --- | --- | --- |
| `to-track` | Reported, no issue attached | no |
| `tracked` | Carries an external issue reference | no |
| `to-review` | The dev delivered and asked for a judgment | no |
| `refused` | Refused, with a mandatory remark | no — returns to `to-review` on the next delivery |
| `validated` | Accepted | yes |
| `discarded` | Set aside, with a mandatory reason | yes |

The kind of a comment — defect, improvement, anything else — stays **on the
comment**, where it is exact and where it is used to write the issue. It never
colours the case.

**A capture carries a stored status**, recomputed by the server whenever a
comment covering it changes: `to-review`, `to-fix`, `validated`. It is written
on one path only — recording a comment recomputes the captures it covers, in
the same transaction. No endpoint sets it.

**Returning to `to-review` is requested by the dev**, never inferred from
captures moving. Images also move for a refactor or a dependency bump, and
asking the reviewer to look at that is noise. "I finished this issue, look" is
something only the dev knows. They may ask for it without having implemented
everything — one issue can depend on the verdict given on another.

## Alternatives rejected

**A state per situation** — the first design, which reached eight case states
including `to-judge` for a pending delivery and `to-rework` for a refusal.
Rejected: two of them duplicated information already carried by comments, and
the whole set collapsed as soon as a case held several comments in different
moments.

**Keeping `to-fix` and `to-improve` apart at case level.** Rejected: the dev's
action is identical in both — write an issue — and a case holding both had to
crush one. The distinction is real on the comment and useless on the case.

**A multi-valued case state**, so that a case can hold several balls at once.
Rejected: it makes every filter and every list ambiguous, for a gain that a
detail view already provides.

**Inferring the return to review from captures moving.** Rejected on noise: a
refactor that changes nothing observable would summon the reviewer.

## Consequences

- A refusal is no longer visible from a list of cases — the case simply reads
  `to-fix`. A marker on the case and a filter on comments compensate. This is
  weaker than a dedicated state and is accepted knowingly.
- The computation has one entry point: a comment changes, the case and the
  covered captures are recomputed. Adding a situation means teaching the comment
  lifecycle, never adding a case state.
- A stored capture status can drift from the comments if anything writes it
  outside that path. That path is therefore the only one, and it is the rule to
  protect.
- `product.md` § 3.1 is rewritten; `to-improve` disappears from the case
  vocabulary and survives only as a comment kind.

## Cross-references

- [ADR 0002](0002-server-owns-the-review-lifecycle.md) — the server computes,
  the client never sets
- [ADR 0006](0006-problems-are-durable-entities.md) — the comment is durable,
  which is what makes it able to carry this lifecycle
- `docs/design/product.md` § 3, § 6
