# ADR 0005 — Exclusive case locking, on its own axis

## Status

Accepted — 2026-08-19

## Context

Hosting the service with several reviewers ([ADR 0001](0001-standalone-multi-project-service.md))
creates a situation the predecessor never had: two people opening the same case
at once, one validating while the other is writing a defect report.

The predecessor's protection was incidental — a single reviewer, on a single
machine, editing a file he owned.

## Decision

**A case is held exclusively by its reviewer for the duration of the session.**
Others see it read-only.

**Occupancy is a separate axis from the cycle state**, and they are never
merged into one field. A case that reads `to-fix` before being opened must
still read `to-fix` while held. Storing "under review" in the state field would
destroy the state it replaces.

**Locks expire.** The session heartbeats; a lock whose heartbeat has been
silent past the configured window is released automatically. A reviewer who
closes their laptop must not block a case indefinitely.

## Alternatives rejected

**Last write wins.** Rejected: silently destroys a colleague's work and records
nothing about who judged what.

**Signed reviews with optimistic conflict detection** — every review attributed
to its author, and a save refused if the case changed underneath, the way a
push is rejected. This was the recommendation: no locks to reclaim, no
abandoned-state problem. Rejected in favour of exclusivity, which prevents the
collision instead of detecting it. Optimistic detection remains a useful
backstop and may be implemented underneath the lock.

## Consequences

- Lock lifecycle enters scope: claim, heartbeat, release, expiry, and a way to
  see who holds what.
- The expiry window is a real product parameter; too short interrupts a careful
  reviewer, too long blocks the team. Left open in `docs/design/product.md` §11.
- Reviews are attributed to their author regardless, since every fact records
  its actor ([ADR 0002](0002-server-owns-the-review-lifecycle.md)).

## Cross-references

- `docs/design/product.md` §3.2
