# Server code conventions

Rules that a linter cannot enforce. Everything a linter can enforce **is** a
linter rule, per the lint-first principle of
[ADR 0009](../adr/0009-documentation-strategy.md) — configure the linter and
leave this file alone. The rationale of a strong rule belongs in an ADR; this
file is neither the enforcement nor the explanation, only what is left over.

Comments: see [`docs/code-comments.md`](../code-comments.md).

Layering, dependency injection and the linters that enforce them: see
[backend ADR 0001](adr/0001-hexagonal-layers.md).

## Error handling

**Never log and return.** Handling an error twice reports it twice and hides
where it came from. Return it wrapped; the boundary logs, once.

```go
// Wrong — reported twice
if err != nil {
    slog.Error("recording the verdict failed", "error", err)
    return err
}

// Right — the caller logs at the boundary
if err != nil {
    return fmt.Errorf("recording verdict: %w", err)
}
```

Wrapping with `%w` is enforced by `errorlint`; the no-log-and-return rule has no
linter, which is why it is written here.

## Discarding an error

`_ = fn()` throws away an error. Two cases are acceptable, each requiring a
comment at the call site so a reviewer can judge the intent:

- `// best-effort: <reason>` — a non-load-bearing side effect whose failure must
  not change the primary outcome.
- `// fail-closed: <reason>` — an authorisation or integrity path where a failed
  check is treated as the negative answer **and** logged. Silent discarding on
  those paths is forbidden.

Anything else returns the error.

## Transactions

**A fact and the transition it causes commit together, or not at all.** This is
the guarantee [ADR 0002](../adr/0002-server-owns-the-review-lifecycle.md) rests
on: a stored state that disagrees with the journal is the one failure the whole
model exists to make impossible.

Any path writing two or more statements whose all-or-nothing outcome is part of
the contract runs in one transaction. The transaction is opened **in the
repository**, never in a use case — `app` cannot import an adapter
([backend ADR 0001](adr/0001-hexagonal-layers.md)), so a write spanning several
tables becomes one repository method.

**Read-after-write** uses `RETURNING` inside the write rather than a follow-up
read on another pooled connection.

## Context propagation

Inherit the caller's `context.Context`. `context.Background()` is allowed only
where no caller exists: entry points under `cmd/`, `init()` functions, and the
top-level loop of a background worker. Tests are exempt.

Anywhere else, a fresh `Background()` cuts cancellation and deadlines off from
the request that started the work.

## Goroutine lifecycle

A long-running goroutine — the lock-expiry sweeper, the event fan-out — follows
`Run(ctx)` / `Stop()` with a `sync.WaitGroup`, so shutdown is deterministic. The
loop's `select` watches both `ctx.Done()` and the stop channel.

**Fire-and-forget goroutines from a request handler are forbidden.** A handler
that spawns `go doSomething()` to answer faster has lost the goroutine: it
cannot survive shutdown predictably, its errors are unobservable, and its
context dies the moment the response is written. Route the work through a
worker.

## Hashing and intake

The capture hash is a **contract**, shared with the CLI
([ADR 0004](../adr/0004-content-addressed-capture-storage.md)): the same bytes
must produce the same address on both sides, forever. The algorithm and its
encoding live in one shared package and change only through a new ADR — changing
them orphans every blob already stored.

Intake streams to disk while hashing, and never buffers a capture in memory: an
edition is hundreds of files, and the server has no reason to hold any of them
whole.
