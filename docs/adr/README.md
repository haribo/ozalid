# Architecture decision records

Cross-cutting and product decisions. Decisions confined to one side live in
[`docs/backend/adr/`](../backend/adr/) and
[`docs/frontend/adr/`](../frontend/adr/).

Each ADR records one structural decision, the alternatives that were rejected,
and why. An ADR is never deleted: a reversal is a **new** ADR, and both sides
carry the link — `Superseded by ADR-NNNN` on the old one, `Supersedes ADR-MMMM`
on the new one. The full lifecycle is set by
[ADR 0009](0009-documentation-strategy.md).

| # | Decision |
| --- | --- |
| [0001](0001-standalone-multi-project-service.md) | A standalone, multi-project, hosted service |
| [0002](0002-server-owns-the-review-lifecycle.md) | The server owns the review lifecycle |
| [0003](0003-runner-and-tracker-agnostic-boundaries.md) | The server knows neither test runners nor issue trackers |
| [0004](0004-content-addressed-capture-storage.md) | Content-addressed capture storage, with provenance |
| [0005](0005-exclusive-case-locking.md) | Exclusive case locking, on its own axis |
| [0006](0006-problems-are-durable-entities.md) | A reported problem is a durable entity |
| [0007](0007-run-intake-policy.md) | Run intake is governed by a per-project policy |
| [0008](0008-no-migration-frozen-predecessor.md) | No migration; the predecessor is frozen |
| [0009](0009-documentation-strategy.md) | Documentation strategy |
| [0010](0010-monorepo-with-server-web-and-cli.md) | One repository holding server, web client and CLI — *partly superseded by 0015* |
| [0011](0011-technology-stack.md) | Technology stack |
| [0012](0012-case-carries-the-ball-comment-carries-the-detail.md) | The case carries the ball, the comment carries the detail |
| [0013](0013-a-recording-is-not-a-capture.md) | A recording is not a capture |
| [0014](0014-server-generated-case-identity-and-catalogue-tree.md) | Server-generated case identity, and a catalogue tree |
| [0015](0015-no-cli-clients-call-the-api.md) | No CLI; clients call the API directly |
| [0016](0016-a-case-is-complete-or-it-says-so.md) | A case is complete, or it says so |
| [0017](0017-a-reference-belongs-to-an-environment.md) | A reference belongs to an environment |
| [0018](0018-an-actor-is-never-invented.md) | An actor is never invented — *partly superseded by 0019* |
| [0019](0019-two-kinds-of-account-one-set-of-rights.md) | Two kinds of account, one set of rights |

0001 to 0008 were decided in a single design conversation on 2026-08-18/19 and
settle what the product is. 0009 to 0011 followed on 2026-08-19 and settle how
it is built. 0012 to 0014, on 2026-08-20, settle the review lifecycle in detail
and correct three points where the original specification was wrong. 0015 to
0017 come from building the thing: each removes something the specification had
assumed without saying so.
