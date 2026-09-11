# ADR 0024 — The pin follows the lock

## Status

Accepted — 2026-09-11

Completes [ADR 0005](0005-exclusive-case-locking.md): freezing the bytes
follows occupancy, not the cycle state.

## Context

The pin exists so a run landing mid-review does not change the bytes under a
reviewer (product.md §7, #142). It was keyed on `to-review` — which says *the
ball is with the reviewer* (ADR 0021), not *someone is looking right now*. A
case could sit frozen for weeks on a recording-less edition, hiding evidence
that arrived after it, to protect a reviewer who did not exist (observed on
`dd9c192d95e0`, #248).

Occupancy has had its own answer since ADR 0005 shipped (#247): the lock.

## Decision

**The displayed edition is derived, never stored.** Asked which edition a case
shows, the server answers from what is true at that moment:

- a **live lock** on the case → the edition stamped when it was claimed;
- otherwise → the project's latest edition.

`case_locks` gains `edition_id`, stamped at claim, kept through heartbeats,
re-stamped on a fresh claim — including after expiry. `cases.current_edition_id`
is removed with its two maintenance mechanisms (`AdvanceCurrentEdition`,
`ReleaseToLatestEdition`); intake stops skipping `to-review` cases.

**A delivery still advances the case at once** (#142): `deliver` re-stamps the
live lock onto the latest edition — with no live lock there is nothing to do,
the case already reads at the latest.

**When a review settles, the state re-derives against the latest edition,
ignoring the saver's own lock** — otherwise a case could stamp `accepted`
blind to an edition already waiting, and nothing later flips it: the intake
rule that reopens a case for its videos (ADR 0023) only fires on future
pushes.

## Rejected alternatives

- **An `in-review` cycle status.** Forbidden by ADR 0005: occupancy is a
  separate axis and never folds into the state field, or it destroys the state
  it replaces.
- **Keeping the stored pin and adding the missing catch-ups** — skip held
  cases at intake, release on lock release, repair at read for locks that died
  in silence. A state that repairs itself on read drifts again the first time
  somebody adds a read path; #248 *is* the catch-up nobody wrote.
- **Showing a later edition's recording beside the pinned captures.** A hybrid
  view where the video shows a flow the captures do not: the reviewer would
  judge an inconsistency the product invented.
