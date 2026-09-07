# 0021 — A status says where the next action happens

Date: 2026-09-07
Status: Accepted

## Context

Four defects in the same area, and one cause (#192): statuses were stored
where they should be computed, and named after two different grammars.

- A capture answered one question with two fields: `capture_verdicts.status`
  said what it waits for, `captures.freshness` said whether the image still
  shows what was approved — and the interface drew the second on top of the
  first. Five marks from two sources that never meet.
- `freshness` was never updated: no `UPDATE` touches it anywhere. Anything
  built on `to-re-review` never empties.
- A case and its captures said the same fact with different words: a refused
  capture put its case at `to-fix`, and `reviewed` said looked-at without
  saying what was concluded.
- The computation fed on its own output: `gatherFacts` read acceptance out of
  the very table `Compute()` writes — the root of #167's take-back loop and
  of #154's stale verdicts.

## Decision

**A status says where the next action happens.** `to-` means ozalid is
waiting for something done inside it; a past participle means it is done
inside, and whatever comes next happens outside. This sharpens the convention
product.md §2 already states ("to + verb means pending, a past participle
means done") on the axis that was missing.

Consequently:

- `to-fix` fails the rule — ozalid has nothing to do, the dev does, outside —
  and becomes **`refused`**. `reviewed` becomes **`accepted`**. A case and
  its captures now share one vocabulary: `not-instrumented · to-review ·
  refused · accepted` for the case, `to-review · refused · accepted · moved`
  for the capture.
- **`moved` stops being an overlay and becomes a capture status.** It applies
  to a capture that is otherwise `accepted` and whose image changed. An open
  comment outranks it: the reason that capture waits is already known.
- **Statuses are derived at read time; only facts are stored** — who accepted
  a capture and when, the remarks, the references, and `moved_pixels` as a
  measurement. The measurement is a fact and survives; the conclusion depends
  on a threshold that can change, and does not.
- **A `refused` capture receiving a new image returns to `to-review`.** The
  refusal's comment stays open and keeps its own cycle: §7's "returning to
  to-review is requested by the dev" keeps governing the **comment**, whose
  judgment waits for a delivery. The capture's status merely says these
  pixels are ones nobody has judged.

## Rejected alternative

Keeping `to-fix` because it names the ball-holder. It does — but so does
`refused` under the rule, and it made a case and its capture describe one
fact with two words.

## Consequences

ADR 0012 stands: the case still carries the ball and the comment the detail —
nothing here contradicts it, so this ADR supersedes nothing. `freshness`
leaves the schema; `moved_pixels` stays. The adapter stops persisting
computed verdicts (#194); the case states rename through the contract (#195);
the interface draws its five marks from one derived status (#196).
