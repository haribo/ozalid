# Backups

ozalid keeps **two stores**, and they have to be restored to points that agree
with each other.

| | holds | lose it and |
| --- | --- | --- |
| PostgreSQL | cases, comments, verdicts, accounts, and the **addresses** of captures | the book is gone, the images survive as files nobody can name |
| The blob store | the capture **bytes**, under the hash of their content | every image ever reviewed is gone, and the database keeps serving rows that point at nothing |

## The order is the rule

**Dump the database first. Snapshot the blobs second.**

Not a matter of taste, and not obvious enough to leave to whoever is holding the
keyboard at 2am. The blob store is append-only — a capture is written under the
hash of its bytes and nothing is ever overwritten
([ADR 0004](adr/0004-content-addressed-capture-storage.md)) — so a snapshot
taken later contains everything an earlier one did. What consistency needs is
that every blob the database *references* is in the snapshot, which makes the
snapshot the later of the two.

Walked through with a run landing at 10:02, between the two:

| order | database | blobs | that run's capture |
| --- | --- | --- | --- |
| database first | 10:00 | 10:05 | bytes present, no row — an orphan blob nobody references, harmless |
| blobs first | 10:05 | 10:00 | row present, bytes missing — a 404 nobody can repair |

The second is unrecoverable: content addressing means the bytes cannot be
rebuilt from anything the database holds. An operator who reverses the order
will not find out for months, and will find out from a reviewer.

## Restoring

The same order, for the same reason: restore the dump, then unpack the blobs.
Bring the server up last — it applies its migrations at boot, so it must not
race the restore.

A restore that finished without an error is not a restore that worked. The one
question worth answering is whether a capture's bytes come back, and the only
way to answer it is to fetch one.

## The drill

```
just restore-drill
```

It is the procedure above, executed. It seeds an instance, backs it up in both
orders, restores each, and checks:

- the capture reads back **byte for byte**;
- a case created after the dump is absent, and its bytes are a harmless orphan;
- the reverse order answers **404** for a capture whose bytes were not restored,
  **and says so in the log** — `a stored address has no bytes behind it` is what
  an operator sees when the store has lost evidence;
- the capture from before the snapshot still reads, so that 404 is about the
  missing bytes and nothing else.

The drill exists because this document was wrong before it was run: it said
blobs first. A procedure nobody has executed is a procedure nobody has checked.

## Still open

How often, how long kept, and where the artefacts go depend on where this is
hosted, which is not settled
([product.md §11](design/product.md#11-open-questions)).
So does whether the drill runs on a schedule — a restore proven once and never
again is a restore nobody can rely on a year later.
