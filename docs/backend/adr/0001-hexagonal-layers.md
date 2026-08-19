# ADR 0001 (backend) — Hexagonal layers, enforced by linter

## Status

Accepted — 2026-08-19

## Context

[ADR 0002](../../adr/0002-server-owns-the-review-lifecycle.md) puts one rule at
the centre of the product: the server computes every cycle transition from the
facts it recorded, and that computation lives in **exactly one place**. The
predecessor's failure was five copies of a single rule spread across a server, a
generator and a repair script.

A rule stays in one place only if there is one place it can be. If the
transition computation can reach a SQL driver, an HTTP request or the system
clock, it will eventually be reimplemented next to each of them, because calling
it from there is inconvenient.

## Decision

The server is layered, and the layering is **enforced by the linter, not by
documentation**.

```
domain    entities, value objects, the state machine — pure, no I/O
app       use cases; defines the interfaces it needs for outbound work
adapters  postgres, filesystem blob store, identity provider
ports     HTTP handlers, middleware, SSE
cmd       assembly
```

1. `internal/app/*` imports neither `internal/adapters/*` nor
   `internal/ports/*`. Outbound needs are interfaces declared in
   `app/<domain>/ports.go`; `cmd/` wires the implementations.
2. `internal/domain/*` imports no other project package. It knows nothing of
   app, adapters or ports.
3. `internal/app/*` does not call `time.Now`, `os.Getenv`, `http.DefaultClient`
   or `sql.Open`. They are injected — a `Clock`, a config struct, a client
   interface, a repository.

Enforcement: `depguard` for rules 1 and 2, `forbidigo` for rule 3 — `depguard`
matches imports, not calls. Test files and `cmd/` are exempt from rule 3.

**Wiring is manual, in `cmd/server/main.go`.** No dependency-injection
framework. A missing or mistyped dependency is a compile error, and reading the
file is the assembly documentation.

## Rationale

- The transition computation is the product's core rule and must be unit
  testable with no database, no network and no wall clock. Rule 3 is what makes
  a deterministic test of a time-dependent transition — a lock expiring
  ([ADR 0005](../../adr/0005-exclusive-case-locking.md)) — possible at all.
- `inputs` fingerprints are meant to serve as a regression oracle
  ([ADR 0002](../../adr/0002-server-owns-the-review-lifecycle.md)): replay
  recorded facts, recompute, compare. Replay is only feasible if the computation
  is a pure function of its inputs, which is precisely what a pure `domain` and
  an I/O-free `app` guarantee.
- Documentation without enforcement drifts, and the drift is invisible until
  someone audits. A linter turns the violation into a failed build.

## Alternatives rejected

**Documentation-only layering.** Rejected on evidence: this is how the
predecessor's rules eroded. A convention nobody can break by accident is worth
more than a convention everyone agrees with.

**Flat package structure, service functions calling the database directly.**
Rejected: it is the shortest path to the five-copies failure, since there is no
place a rule *has* to live.

**A dependency-injection framework (Wire, Fx).** Rejected: a generator adds a
build step and a generated file, a runtime container moves wiring errors from
compile time to boot time. The graph is small enough that straight-line
assembly reads better. Reconsider only with concrete pain, in a new ADR.

## Consequences

- `.golangci.yml` carries the `depguard` and `forbidigo` configuration; it is
  part of the architecture, not of the formatting.
- Adding an outbound concern means adding an interface to `ports.go` and an
  implementation under `adapters/`, never an import shortcut.
- `cmd/server/main.go` grows with the product. That is accepted: it is
  straight-line code, and reading it tells the whole assembly story.
- The state machine of `product.md` § 3 lives in `domain`, has no dependency,
  and is testable as a table of facts in and transitions out.

## Cross-references

- [ADR 0002](../../adr/0002-server-owns-the-review-lifecycle.md) — the one-place
  rule this layering protects
- [ADR 0011](../../adr/0011-technology-stack.md) — the stack this layering sits in
