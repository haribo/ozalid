# ADR 0004 (backend) — Capture bytes on the filesystem, behind a port

## Status

Accepted — 2026-08-19

## Context

[ADR 0004 (product)](../../adr/0004-content-addressed-capture-storage.md) settled
*how* captures are addressed — by the hash of their bytes, stored once however
many editions reference them. It did not settle *where* the bytes live.

They do not live in PostgreSQL: hundreds of megabytes of images per project is
not what a relational database is for, and it would put the review workload
behind the same connection pool as the lifecycle. The database holds hashes and
pointers; the bytes go somewhere else.

Two candidates. A directory on the machine's disk, or an object store speaking
the S3 protocol. The hosting question is still open (`product.md` § 11), so
the decision is being taken before the constraint that would settle it is known.

## Decision

**v1 stores capture bytes on the local filesystem, behind a port.**

`app` declares the interface and knows nothing else:

```go
type BlobStore interface {
    Put(ctx context.Context, hash string, r io.Reader) error
    Get(ctx context.Context, hash string) (io.ReadCloser, error)
    Exists(ctx context.Context, hash string) (bool, error)
}
```

The v1 adapter writes under a configured root, sharding by hash prefix
(`blobs/a3/f2/a3f2…`) to keep directories to a workable size. Writes are
atomic — temporary file, then rename — so a crash mid-intake leaves no
half-written blob claiming a valid hash.

An S3-compatible adapter is a second implementation of the same interface, added
when hosting requires it, in its own ADR.

## Rationale

- Content addressing makes the filesystem adapter nearly trivial: the path *is*
  the hash, nothing is ever modified, nothing is ever overwritten, and
  `Exists` — the check that lets intake skip bytes the server already holds — is
  a `stat`. The properties an object store would provide are properties the
  addressing scheme already guarantees.
- It removes a service, a set of credentials and a slow test path from v1, at a
  stage where no deployment requires them.
- The cost of deferring is bounded and known: one adapter, plus copying the
  files. No business rule changes, because no business rule mentions storage.

## Alternatives rejected

**S3-compatible object storage from v1.** Rejected for v1 only. It is the right
answer as soon as the service runs on more than one machine, or on a platform
with ephemeral container disks — at which point the port makes it an additive
change. Adopting it now buys nothing and costs a service in every development
environment and every test run.

**Both adapters from v1**, selected by configuration. Rejected: two
implementations to write, test and keep behaviourally identical before either
is needed.

**Bytes in PostgreSQL as bytea or large objects.** Rejected: it bloats backups
with data that is immutable and already deduplicated, puts image serving on the
connection pool, and gains only a transactional guarantee that content
addressing makes unnecessary — an orphaned blob is inert, and a missing one is
detectable by hash.

## Consequences

- Deployment gains a stateful requirement: a durable volume, backed up
  separately from the database. Two restore drills, not one, and they must be
  drilled together — a database restored to a point where it references blobs
  that were pruned is a corrupt book.
- Nothing in `app` or `domain` mentions a path, so the S3 migration is an
  adapter and a copy.
- Serving captures to the browser goes through the server for now. When that
  becomes a bottleneck, the S3 adapter also brings pre-signed URLs — one more
  reason the decision is deferred rather than reversed.
- The retention rule left open in `product.md` § 11 lands here: pruning means
  reference counting against the database, and a blob must never be deleted
  while an edition or a reference still points at it.

## Cross-references

- [ADR 0004 (product)](../../adr/0004-content-addressed-capture-storage.md) —
  the addressing scheme this stores
- [backend ADR 0001](0001-hexagonal-layers.md) — why this is a port
- `docs/design/product.md` § 11 — the hosting and retention questions this
  touches
