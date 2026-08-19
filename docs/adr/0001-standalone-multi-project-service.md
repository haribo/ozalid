# ADR 0001 — A standalone, multi-project, hosted service

## Status

Accepted — 2026-08-19

## Context

ozalid begins as the extraction of an internal tool built inside a product
monorepo (July–August 2026). That tool works: 240 catalogued cases, per-step
captures in four variants, videos, a byte-comparison staleness model, a review
UI, and a review cycle documented in enough detail to be followed. It proved
the concept.

It also accumulated structural faults that no amount of incremental work fixes:

- **Multiple writers on the same truth.** Verdicts live in a JSON file written
  by a small review server, by one-shot repair scripts, and — for at least one
  step of the cycle — by a human in a text editor. Attaching an issue number to
  a reported problem has no endpoint at all; it is typed into the file. That
  single manual step is the transition that unblocks the whole pipeline.
- **The same rule implemented five times.** "An open unvalidated defect means
  the case is to-fix" appears three times in the review server, again in the
  generator, and is copied into a repair script. Five copies of one rule, in a
  tool of a few thousand lines, before any new feature was added.
- **State derived on read, nowhere recorded.** The cycle state was recomputed
  from the current facts at generation time. Yesterday's state was not
  retrievable, no transition was dated, and a change to the computation rule
  silently rewrote the past.
- **One copy of each capture.** Only the latest bytes were kept, so a new run
  physically overwrote what a reviewer was looking at. This produced real
  damage and a dedicated repair script to undo it.
- **Single machine, single reviewer.** Approved bytes lived in a local,
  gitignored directory. Nothing was shareable, nothing survived a second
  machine.

The owner's ambition for the tool is explicitly larger: a solid review
instrument for a growing product, reusable across projects, open source.

## Decision

ozalid is a **standalone product** in its own repository, designed for
**multiple projects** from the schema up, deployed as a **hosted service** with
concurrent reviewers.

- `project` is a first-class entity from day one. No client-specific constant
  exists anywhere in the product.
- Everything the predecessor hardcoded becomes per-project configuration: the
  variant axes, the categorisation of cases, the instrumentation conventions.
- The originating project becomes ozalid's first client, not its owner.

## Alternatives rejected

**Stay inside the originating monorepo with a multi-project-ready schema,
extract later.** This was the recommendation put to the decision-maker: keep
genericity as a design constraint (no hardcoded constants, configuration in
data) while building for exactly one real user, and pay for authentication,
deployment and hosting only when a second project actually exists. Rejected:
the owner wants a product, and wants it hosted and open source rather than a
monorepo tool that might one day be extracted.

**Stay specific to the originating project.** Rejected for the same reason.

## Consequences

- Authentication, deployment, backups, restore drills, API versioning and
  schema migrations all enter scope immediately. They are not incidental; each
  is treated as its own decision.
- The predecessor's most valuable assumptions are also its most
  client-specific ones — four fixed variants, a particular e2e folder layout,
  a helper wrapping business steps, identities derived from file path and test
  title, and a determinism recipe involving mail and cache resets.
  Generalising all of it **without a second user to validate the abstractions**
  is the principal risk this ADR knowingly accepts. The mitigation is
  [ADR 0003](0003-runner-and-tracker-agnostic-boundaries.md): keep the server's
  vocabulary small enough that a wrong abstraction is cheap to replace.
- The originating project keeps its existing tool, frozen, until ozalid can
  replace it ([ADR 0008](0008-no-migration-frozen-predecessor.md)).

## Cross-references

- [ADR 0002](0002-server-owns-the-review-lifecycle.md) — the lifecycle model
- [ADR 0008](0008-no-migration-frozen-predecessor.md) — the predecessor's fate
- `docs/design/product.md` — the specification this ADR authorises
