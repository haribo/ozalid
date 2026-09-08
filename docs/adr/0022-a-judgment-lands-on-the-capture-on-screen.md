# 0022 — A judgment lands on the capture on screen

Date: 2026-09-08
Status: Accepted — supersedes the settling clause of [ADR 0020](0020-a-remark-is-a-verdict.md)

## Context

Judging fix #173 from the dark capture also judged the light one (production
rc.19, #208). ADR 0020's "settling was the judgment" resolves a remark
all-or-nothing: settling it settles every covered capture at once. Designed
and tested — and wrong at the act's granularity, because a fix can work on
one variant and not the other. The all-or-nothing round forces refusing a
half-good fix or accepting a half-bad one.

## Decision

**The verdict pair always judges the capture on screen**, whatever the
context — bare capture or delivered fix. It already held for accept and its
take-back; the fix judgment was the inconsistent exception.

A remark stays **one unit of reporting**: one defect over N variants is one
comment. Its **resolution is per variant** — the coverage is a stored fact,
and it shrinks:

- **Accepting a delivered fix on a variant** removes that variant from the
  remark's coverage, records an explicit capture acceptance, and stamps the
  reference for it — the judge approved those bytes (#206).
- The ref — and the comment — **settle when the coverage empties**: the last
  accepted variant closes the round.
- **Refusing on a variant** opens a **partial round**: the ref returns to the
  dev for the remaining coverage; released variants stay released. The
  refusal's anchor follows the refused variant — the judge refused *these*
  bytes.
- **Take-backs are symmetric per variant**: unjudging one variant restores
  its coverage.
- Every coverage change is journalled: who released or restored what, when.

## Rejected alternative

Keeping all-or-nothing and merely displaying the scope ("covers dark +
light"). Predictable to read, wrong to act: it still forces a verdict on
pixels of another variant.

## Consequences

ADR 0020 stands except its settling clause — a remark is still born from a
refusal and always blocks; only how it dies changes. The zone names the scope
("issue #173 · this variant") and a mixed state — one variant accepted, the
other still delivered — is legitimate and visible. Statuses stay derived
(ADR 0021): the shrinking coverage is exactly the kind of fact the
derivation reads.
