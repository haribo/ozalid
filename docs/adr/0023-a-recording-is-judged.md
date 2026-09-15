# ADR 0023 — A recording is judged

## Status

Accepted — 2026-09-09

Supersedes the "never a source of state" clause of
[ADR 0013](0013-a-recording-is-not-a-capture.md); its "never compared" clause
stands untouched.

## Context

ADR 0013 removed `to-re-watch`: a freshness the server could not compute from
what it holds. It went one step further and barred recordings from state
entirely — viewable, and nothing else.

Reviewing vilajo showed the gap. The reviewer watches the flow video as
evidence, and has no way to say what they concluded: a verdict pair everywhere
except on the thing just watched. An explicit human judgment on identified
bytes is not the unprovable freshness 0013 removed — it is exactly a fact the
server can verify (ADR 0002): who judged which recording, and when.

## Decision

**A recording is judged, and the judgment lands on the recording on screen**
(ADR 0022): one recording — one edition, one variant — carries its own verdict.

**Facts stored, statuses derived** (ADR 0021): `to-review` (nobody judged
these bytes), `accepted`, `refused`. A refusal carries its mandatory remark
(ADR 0020), and both verdicts can be taken back, symmetrically — the journal
keeps every move.

**Every push resets the verdict.** Encoding is not deterministic, so a new
edition brings new bytes and a new `to-review` recording. What was accepted is
exactly what was watched, never more.

**The recording counts in the case derivation like a capture**: an unjudged
recording keeps the case `to-review`, a refused one hands it to the dev. A
case without a recording stays a normal case — optional by construction.

**Never compared stands.** No reference, no `moved`, no freshness, no
byte-wise reading of two videos — that half of ADR 0013 is untouched.

## Consequences

- The price is explicit: a case with recordings cannot settle without the
  reviewer judging the video of each variant **at every edition**. The video
  is evidence, and evidence is judged each time it changes.
- The refusal's remark is what the dev reads; it lives with the judgment and
  leaves the read model when the refusal is answered or taken back (#212).

## Rejected alternatives

- **A verdict that does not count in the case state.** A pair that gates
  nothing is decoration.
- **A verdict riding the variant across editions.** It would claim approval of
  bytes nobody watched.
