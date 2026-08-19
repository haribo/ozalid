# ADR 0003 — The server knows neither test runners nor issue trackers

## Status

Accepted — 2026-08-19

## Context

Two integrations sit at the edges of the product, and both are tempting to
absorb.

**The test runner.** Captures come from an e2e suite. The predecessor's
generator read Playwright's JSON reporter output directly and understood its
internal structure. Convenient for one project; fatal for a product meant to
serve several, since every runner would have to be taught to the server.

**The issue tracker.** The cycle depends on defects being taken over by issues.
The predecessor called the GitHub CLI at generation time to read issue states,
with a five-minute cache and an offline fallback — machinery that exists purely
because a third party might be slow or unreachable.

## Decision

**The server's vocabulary is cases, steps, variants, captures, problems.
Nothing else.**

**Runner-agnostic.** Intake accepts a neutral manifest. Translating a runner's
output into it is the client's job; the adapter ships with the CLI. Adding
support for a new runner never touches the server.

**Tracker-agnostic.** A problem may carry an opaque external reference — an
identifier and a URL. ozalid never creates, reads, closes or comments anything
on a tracker, and holds no tracker credential. The flow runs **outward**: a
reviewer accepts a fix in ozalid, and something outside closes the issue as a
consequence.

## Alternatives rejected

**Server-side runner parsing.** Rejected: marries the product to one tool and
puts the fastest-moving knowledge in the slowest-moving component.

**Neutral format plus official server-side adapters.** Not rejected on merit —
deferred. It can be added later without changing the contract, and it is not
needed while there is one client.

**Full tracker integration** (create, comment, poll state). Rejected on the
decision-maker's own observation that a closed issue "lives outside": the
review verdict is the reviewer's, not the tracker's. Once the flow runs
outward, reading the tracker has no purpose — which removes stored credentials,
a state cache, an offline degraded mode, and a per-tracker implementation.

**Optional enrichment connector.** Same reasoning; nothing in the cycle
consumes it.

## Consequences

- The product cannot display "this issue is closed". It displays what the
  reviewer decided, which is the information the cycle actually needs.
- Whoever creates the issue must report the link back to ozalid. That client —
  CLI or automation — is where the tracker knowledge lives.
- These boundaries are the mitigation for the genericity risk accepted in
  [ADR 0001](0001-standalone-multi-project-service.md): a small server
  vocabulary means a wrong abstraction is cheap to replace.

## Cross-references

- `docs/design/product.md` §5, §9
