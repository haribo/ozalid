# ADR 0001 (frontend) — Vue 3, TypeScript, Vite, Tailwind

## Status

Accepted — 2026-08-19

## Context

[ADR 0011](../../adr/0011-technology-stack.md) settled Vue 3 and TypeScript for
the web client. The surrounding tooling — bundler, styling, unit runner,
end-to-end runner, lint pipeline — is chosen here, as one set rather than five
independent picks.

What the review book actually asks of a frontend:

- A **dense grid** of captures, steps across variants, with per-cell verdicts
  accumulated in memory and saved as one review session.
- **Image comparison** — a capture against its reference, at a zoom level the
  reviewer controls.
- **Long-lived connections** — a lock heartbeat, and a server-sent event stream
  telling the client that a case changed under it.
- **Two themes.** The product's own subject matter is light and dark rendering
  ([`product.md` § 2](../../design/product.md)); a review tool that renders
  correctly in only one of them is not credible.
- **Keyboard-first review.** A reviewer passing a case moves through cells
  without reaching for the mouse.

## Decision

**Vue 3 (script setup) + TypeScript + Vite + Tailwind CSS + Pinia + Vitest +
Playwright, with oxlint and eslint.**

| Component | Why |
| --- | --- |
| Vue 3, script setup | Composition API for the stateful review surface; the maintainer's fluency is the deciding factor ([ADR 0011](../../adr/0011-technology-stack.md)) |
| TypeScript | Closes the contract loop: the client is generated from the OpenAPI document ([backend ADR 0002](../../backend/adr/0002-spec-first-openapi.md)), so a server-side rename becomes a type error here |
| Vite | esbuild-backed dev server and HMR; the same transform pipeline serves Vitest |
| Tailwind | Utility-first, no naming debates, and a dark variant on every colour utility — which this product cannot treat as optional |
| Pinia | The review session (verdicts pending save, lock state, event-stream updates) is shared across components and outlives any one of them |
| Vitest | Vite-native, no parallel transform configuration to maintain |
| Playwright | Headless by default, multi-browser, the same tool ozalid's own clients will instrument |
| oxlint + eslint | oxlint runs the fast correctness pass; eslint carries the plugin ecosystem (Vue, Vitest, Playwright) and any project rule |

## Alternatives considered

**React.** Rejected in [ADR 0011](../../adr/0011-technology-stack.md): no
decisive advantage at this scale, lower fluency.

**Nuxt.** Rejected: server-side rendering and file-based routing solve problems
this application does not have. The review book is behind authentication, has no
indexable surface, and its routes are few.

**Vuex.** Rejected: superseded by Pinia in the Vue ecosystem, more ceremony for
the same job.

**No state library, provide/inject and composables only.** Considered
seriously — the state is not vast. Rejected because the review session is
written from several places at once (cell verdicts, the problem editor, the
event stream, the lock heartbeat) and an unowned shared ref across four writers
is where bugs of the "my verdict disappeared" kind live. That failure mode is
the one the product exists to prevent.

**Cypress.** Rejected: Playwright parallelises better and covers more browsers.
It also matters that ozalid's first clients instrument their suites with
Playwright — using it here keeps one mental model across the product and its
input.

**Plain CSS or a component library (Vuetify, PrimeVue).** Rejected: a component
library imposes its own visual identity on a tool whose entire job is to display
someone else's UI faithfully. Chrome must stay quiet and must never be confused
with the captures it frames.

## Consequences

- Dark and light are both first-class from the first component. A colour
  utility without its `dark:` counterpart is a defect, not an omission.
- The generated OpenAPI client is the only way the application talks to the
  server; a hand-written `fetch` against an endpoint bypasses the contract.
- Unit tests are colocated with their source (`Component.spec.ts` next to
  `Component.vue`).
- End-to-end tests run headless; `--headed` is a local debugging flag only.
- Lint order: oxlint (correctness), then eslint (Vue and project rules), then
  prettier (formatting).

## Cross-references

- [ADR 0011](../../adr/0011-technology-stack.md) — the stack this details
- [frontend ADR 0002](0002-fsd-architecture.md) — how the code is organised
