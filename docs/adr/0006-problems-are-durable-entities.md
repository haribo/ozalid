# ADR 0006 — A reported problem is a durable entity

## Status

Accepted — 2026-08-19

## Context

The predecessor treated a reviewer's report as **volatile**: reviewer input,
not a stateful thing. Each one was triaged — promoted to an issue when
justified, or discarded after discussion — and the trace lived in the
conversation and in the git history of the verdicts file.

That held for a single reviewer reading his own diffs. It does not survive a
shared service, where "I reported this three months ago, who removed it and
why?" is a question that will be asked, and where the git fallback no longer
exists.

## Decision

**A problem is a first-class entity with its own lifecycle.** Nothing is ever
deleted.

- Fields: kind (defect or improvement), text, anchoring step, affected
  variants, state, history.
- One real defect spanning several variants is **one** problem with several
  variants checked — never one per variant.
- States: `open` → `tracked` (an external reference was attached) or
  `discarded`, **with a mandatory reason**. A tracked problem is later
  `accepted` or `rejected`, a rejection carrying a mandatory comment.
- A discarded problem stays visible on its case, with its reason and its
  author.

## Alternatives rejected

**Keep the volatile model.** Rejected: it is the only place in the system where
information would be destroyed, while everything else is journalled.

**Volatile on screen, retained in the journal.** Rejected as a half-measure:
the information would exist but only be reachable by digging, which in practice
means never. The reason a defect was dismissed is a design decision and belongs
on the case.

## Consequences

- Discarding costs a sentence. That friction is intended — a defect dismissed
  without a reason is exactly what comes back in six months.
- The UI must distinguish live problems from settled ones without hiding the
  settled ones.
- Attaching a reference and discarding are both **facts**, so both can move the
  cycle state ([ADR 0002](0002-server-owns-the-review-lifecycle.md)).

## Cross-references

- `docs/design/product.md` §6
