# ADR 0018 — An actor is never invented

## Status

Accepted — 2026-08-27. **Partly superseded by
[ADR 0019](0019-two-kinds-of-account-one-set-of-rights.md)** — what a person's
identity is, and how they prove it. What survives: a service token belongs to
one project, and the rows already written as `anonymous` stay.

Settles what `docs/design/product.md` § 8 left open: it says how someone proves
who they are, not what ozalid then writes down.

## Context

Every fact ozalid records carries an actor. The schema has said so since the
first migration — `journal.actor_id`, `journal.actor_kind` constrained to
`human` or `machine`, `comments.author_id`, `capture_references.approved_by`.

All of them currently read `anonymous`. The product's own promise, that
*"I reported this three months ago, who removed it?"* always has an answer
(§ 6), is unmet, and `grep -n "security" apps/server/api/openapi.yaml` returns
nothing: no endpoint is protected at all.

Three questions have to be answered before a single credential is checked,
because each of them becomes unreversible the moment real data exists. They are
answered here rather than in the pull requests that will need them, so that no
pull request settles by accident what should be settled on purpose.

## Decision

**A human's `actor_id` is the provider's subject, stored as it comes.** No user
table, no profile, no mail address serving as a key. A person who changes their
address keeps their identity, and ozalid holds nothing about them it would then
have to protect, migrate or forget.

**A service token belongs to exactly one project.** A token naming project A
cannot push to project B, and the attempt is refused rather than ignored. A
project is a first-class boundary ([ADR 0001](0001-standalone-multi-project-service.md));
a credential that crosses it silently makes the boundary decorative.

**The rows already written as `anonymous` stay.** The journal is append-only.
Rewriting history so it looks answered would produce a book that lies about
itself, which is worse than a book with an honest gap in its first chapters.

## Alternatives rejected

**A `users` table keyed on the mail address.** Rejected: an address is a
contact detail, not an identity. People change them, providers reassign them,
and every fact ever recorded would point at whoever holds it now.

**A `users` table keyed on an ozalid-generated id, with the subject as a
column.** Not wrong, and rejected as premature: it buys a display name and a
preference or two, at the cost of a profile to keep in step with the provider.
Nothing in the product needs it yet. If it ever does, adding the table is a
migration, not a redesign — the subject stays the key.

**Tokens scoped to an installation rather than a project.** Rejected: one leaked
token would reach every project on the host. The narrower scope costs nothing
today and cannot be retrofitted once tokens are in use.

**Backfilling `anonymous` to a real actor.** Rejected on principle. A journal
that can be rewritten is not evidence, and the value of every row after it
depends on that being true.

## Consequences

- Nothing is added to the schema to carry identity. What is missing is what
  proves it: a table of service tokens, and a session for humans.
- `actor_kind` already distinguishes a person from a program, so the journal
  answers "was this a human?" the day the first credential is checked.
- An `anonymous` row is legible forever as "before identity", which is a true
  statement about this period and a useful one.
- Authorisation is **not** settled here. Who may do what is a separate question,
  and § 8 is silent on it — a gap in the design, not an oversight in the code.
  It is tracked by its own piece of the work.

## Cross-references

- [ADR 0001](0001-standalone-multi-project-service.md) — the project as a
  first-class boundary
- [ADR 0002](0002-server-owns-the-review-lifecycle.md) — every recorded fact is
  journalled with its actor
- [ADR 0006](0006-problems-are-durable-entities.md) — nothing is deleted, which
  is why nothing may be rewritten either
- `docs/design/product.md` § 6, § 8
