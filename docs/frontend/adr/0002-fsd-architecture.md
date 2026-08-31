# ADR 0002 (frontend) — Feature-Sliced Design

## Status

Accepted — 2026-08-19

## Context

A single-page application without an architectural rule drifts the same way a
backend does: components reach into each other's internals, business logic
settles inside presentation components, and the import graph becomes a mesh
where every change ripples.

The backend answers this with enforced layering
([backend ADR 0001](../../backend/adr/0001-hexagonal-layers.md)). The frontend
needs an equivalent, and it needs one before the code exists rather than after —
retrofitting an architecture is the expensive order.

## Decision

The web client follows **Feature-Sliced Design** (https://feature-sliced.design/).

- **Layers**, in dependency order — `app → pages → widgets → features → shared`.
  A layer may depend downward, never upward.
- **No cross-slice imports.** One feature never imports another feature
  directly. Shared code moves to `shared/`; coordination happens in a layer
  above.
- **A public API per slice.** Each slice exposes one `index.ts`; importing a
  slice's internals from outside is forbidden.

The `entities/` layer is deliberately absent until a slice genuinely belongs
there — a pure business model with no UI. Introducing it empty invites
misplacement.

Layer boundaries and the public-API rule are enforced by lint configuration, per
the lint-first principle of
[ADR 0009](../../adr/0009-documentation-strategy.md).

## Rationale

- Layers cap upward dependencies and slices cap horizontal coupling: adding a
  feature does not put unrelated features at risk.
- A public API per slice means a slice can be rewritten freely as long as its
  `index.ts` holds. Without the rule, every internal symbol becomes public by
  accident, and the first refactor discovers it.
- FSD is documented and maintained externally, so the onboarding cost is
  amortised — a newcomer reads the methodology before reading the code.

## Alternatives considered

**Flat feature folders, no layering.** Rejected: it works until features start
sharing, then coupling has nowhere to go and the refactor blast radius is
unbounded.

**Atomic design** (atoms, molecules, organisms). Rejected: a UI-only taxonomy
with no home for stores, side-effecting composables or domain logic. The review
book is state-heavy, not a component gallery.

**Routing-driven structure.** Rejected: it couples the architecture to the URL
surface, and the application has few routes and a lot of state behind each.

**No enforced architecture, rely on review.** Rejected on the same grounds as
[backend ADR 0001](../../backend/adr/0001-hexagonal-layers.md): a boundary nobody
can cross by accident is worth more than one everyone agrees with.

## Re-evaluation triggers

- Cross-slice coupling becomes pervasive despite the rule — a sign the layering
  does not match the product.
- The layer model adds more friction than it absorbs.

Either trigger produces a new ADR superseding this one, with a migration plan.

## Consequences

- `apps/web/src/` follows the layers; a new feature is a slice with a declared
  public API, consuming only from below.
- The generated OpenAPI client lives in `shared/api` — every layer may consume
  it, and nothing else talks to the server.
- Lint configuration enforcing the boundaries ships with the first feature, not
  after several.

## Cross-references

- [frontend ADR 0001](0001-frontend-stack.md) — the stack these layers organise
- [ADR 0009](../../adr/0009-documentation-strategy.md) — the lint-first principle
- https://feature-sliced.design/ — the methodology
