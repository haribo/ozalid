# ADR 0016 — A case is complete, or it says so

## Status

Accepted — 2026-08-25

Amends `docs/design/product.md` § 3, which declared three independent state
axes, and § 7, which was silent on an incomplete run.

## Context

The grid drew a missing capture as a neutral blank labelled "no capture". That
label states a fact and hides a defect.

A case is instrumented to be looked at in every variant its suite declares for
it. If the login screen is captured on `desktop·light`, `desktop·dark` and
`mobile·light` but not `mobile·dark`, nobody decided that — a run failed, a
viewport timed out, a step threw before its screenshot. Drawing that hole the
same way as a deliberate absence tells the reviewer nothing is wrong.

Worse, the hole is invisible at scale. A reviewer approves the eleven captures
that exist, the case reaches `reviewed`, and the catalogue gauge reports a
clean case whose evidence has a gap. The state is not false — no comment is
open — but read alone it certifies more than the evidence supports.

The predecessor had the same blank and the same silence.

## Decision

**A case is complete at an edition when every step carries a capture for every
variant that edition declares for that case.**

The variant set of a case at an edition is the union of the variants across its
steps. A step missing one of those variants has a **hole**. An axis value the
case never exercises — a flow that is desktop-only — produces no column and no
hole: completeness is measured against what the run itself declared, never
against the project's full axis catalogue.

**Completeness is a fourth independent axis**, alongside cycle state, occupancy
and freshness. It is computed from facts the server already holds, displayed
next to the state, and never collapsed into it.

**Intake accepts a run with holes.** It records them and does not refuse.

**A hole is not a capture**: it carries no verdict, cannot be judged, and is
drawn as an anomaly rather than as an absence.

**`reviewed` stays reachable on an incomplete case.** A reviewer who approved
every capture that exists has genuinely finished; the case then reads
`reviewed` *and* incomplete. Two true facts, neither hiding the other — the
same reason § 3 keeps its axes apart.

## Alternatives rejected

**Refusing the edition at intake.** Rejected: one flaky step would block the
whole intake, and the evidence that *was* produced would be thrown away. ozalid
records what happened; policing the suite is [ADR 0007](0007-run-intake-policy.md)'s
job, and its `strict` mode already exists for projects that want pressure.

**Collapsing incompleteness into the cycle state** — a fifth value, or forcing
the case to `to-review`. Rejected: the ball would sit with a reviewer who can do
nothing about it, and the case could never leave that state by any act of
theirs. The cycle state answers who must act on the *review*; a hole is a defect
of the *run*, whose owner ozalid does not model.

**Leaving the blank neutral**, and reporting completeness only in the intake
response. Rejected: the intake response is read once, by a machine. The gap is
discovered months later by whoever trusted the gauge.

## Consequences

- The grid draws a hole as an anomaly and counts it on the case; the catalogue
  reports incomplete cases per category, beside the gauge.
- The server computes completeness per case and edition, and exposes it on the
  grid. No new stored state: it is a property of the edition's contents.
- A case whose suite legitimately drops a variant shows no hole, so tightening
  a matrix costs nothing and loosening it is visible.
- § 3's opening sentence changes: four axes, not three. The rule it protects is
  unchanged — collapsing any of them loses information.
- Retention interacts: pruning an edition must not turn an older complete case
  into an apparently incomplete one. Whatever [§ 11](../design/product.md)'s
  retention rule becomes, it prunes whole editions or nothing.

## Cross-references

- [ADR 0002](0002-server-owns-the-review-lifecycle.md) — a state is computed
  from verifiable facts
- [ADR 0007](0007-run-intake-policy.md) — what intake is allowed to refuse
- [ADR 0012](0012-case-carries-the-ball-comment-carries-the-detail.md) — why the
  cycle state carries no detail
- `docs/design/product.md` § 3, § 7
