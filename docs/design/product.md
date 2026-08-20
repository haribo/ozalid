# ozalid — product specification

This document is the source of truth for **what** ozalid does. The **why**
behind each structural choice — including the alternatives that were rejected —
lives in [`docs/adr/`](../adr/). Where the two disagree, the ADR wins and this
document is the bug.

Status: design phase, nothing implemented.

## 1. The problem

A team ships UI changes continuously. Someone has to look at them. Doing it by
hand does not scale: the reviewer replays the same flows, in the same variants,
mostly to confirm that nothing moved.

An e2e suite already walks every flow. It knows the steps, it can capture what
the screen looked like at each one, in every variant. What it cannot do is
**judge**. ozalid is the missing half: it collects the evidence, tracks who
judged what, and — critically — makes sure a reviewer is only ever asked to
look at what actually changed since their last verdict.

Two things follow, and they define the product:

- ozalid is **not** a visual-regression tool. It never fails a build, never
  asserts that pixels are identical. A byte difference is a *prompt to look*,
  not a defect.
- ozalid **owns the review lifecycle**. The state of a case is not a UI concern
  and not a file convention; it is server-owned data with an audit trail
  ([ADR 0002](../adr/0002-server-owns-the-review-lifecycle.md)).

## 2. Vocabulary

| Term | Meaning |
| --- | --- |
| **Project** | A product under review. Owns its own configuration, users, cases and storage. First-class from day one ([ADR 0001](../adr/0001-standalone-multi-project-service.md)). |
| **Case** | One reviewable user flow. Carries a short server-generated id; title and description are mutable and carry no identity ([ADR 0014](../adr/0014-server-generated-case-identity-and-catalogue-tree.md)). |
| **Category** | A node in the project's catalogue tree, of unrestricted depth. A case belongs to exactly one. |
| **Step** | A named business moment inside a case ("submits the form"). Ordered. |
| **Axis** | A rendering dimension the project declares — `theme`, `viewport`, `locale`, or anything else. ozalid ships no built-in list. |
| **Variant** | A combination of axis values. An axis the client does not supply is simply absent from the combination. |
| **Capture** | One image: a given step, in a given variant, at a given edition. Comparable, hashed, referenced. |
| **Recording** | The flow video. Optional, viewable, **never** compared byte-wise and never a source of state ([ADR 0013](../adr/0013-a-recording-is-not-a-capture.md)). |
| **Edition** | One accepted intake of a run. Immutable once accepted. |
| **Comment** | A reviewer's report against a step and a set of variants. A durable entity with its own lifecycle ([§6](#6-comments)). Formerly called a *problem*. |
| **Verdict** | The stored status of one capture: `to-review`, `to-fix`, `validated`. Computed by the server from the comments covering it. |
| **Reference** | The capture bytes a given capture was last approved against. What "has it changed?" is measured from. |

Grammar convention, inherited and kept: **"to + verb" means pending, a past
participle means done**. `to review` is work waiting; `reviewed` is work
finished.

## 3. Case state

A case carries **three independent axes**. Collapsing them loses information —
this is a hard rule, not a preference.

### 3.1 Cycle state (who holds the ball)

Stored, never received as a parameter — the server computes every transition
from the facts it just recorded ([ADR 0002](../adr/0002-server-owns-the-review-lifecycle.md)).

The state answers **one** question and carries no detail: the detail lives on
the comments ([ADR 0012](../adr/0012-case-carries-the-ball-comment-carries-the-detail.md)).

| State | Condition | Ball |
| --- | --- | --- |
| `not-instrumented` | No capture and no verdict. Outside the funnel. | nobody |
| `to-review` | Something is waiting for the reviewer's judgment. | reviewer |
| `to-fix` | Nothing awaits the reviewer, and at least one comment awaits the dev. | dev |
| `reviewed` | No open comment. The only clean state. | nobody |

`to-review` outranks `to-fix` when both apply: a verdict can cancel work in
progress, so it comes first.

Each capture also carries a **stored status** — `to-review`, `to-fix`,
`validated` — recomputed by the server whenever a comment covering it changes.
Recording a comment and recomputing the captures it covers happen in one
operation; nothing else writes that status.

### 3.2 Occupancy (is someone working on it right now)

Independent of the cycle state, and that separation is the point: when a
reviewer opens a case that sits at `to-fix`, it must still read `to-fix`
afterwards.

- `free`, or `held by <user> since <timestamp>`.
- A held case is read-only for everyone else ([ADR 0005](../adr/0005-exclusive-case-locking.md)).
- Locks expire: the session sends a heartbeat, and a lock whose heartbeat has
  gone silent for longer than the configured window is released automatically.

### 3.3 Freshness (is the evidence still the evidence that was judged)

Computed at intake, per capture, against that capture's reference:

- `current` — the bytes the reviewer approved are still the bytes on display.
- `to-re-review` — the capture moved. The changed cells are marked
  individually; the reviewer re-passes those, not the whole case.

A capture counts as moved when its hash differs **and** a bounded pixel
comparison exceeds a per-project threshold — below it, the difference is
rasterisation noise. The number of differing pixels is recorded so the
threshold can be judged rather than guessed.

Recordings are never compared: encoding is not deterministic, so a video can
never prove anything about its own freshness
([ADR 0013](../adr/0013-a-recording-is-not-a-capture.md)).

Freshness is an **overlay**, not a state. A `reviewed` case whose captures move
is still `reviewed` until the reviewer says otherwise.

### 3.4 Transitions

Every transition is journalled: `{case, from, to, at, actor, cause, inputs}`.

`inputs` is not decoration. It is a fingerprint of the facts the server used to
compute the transition — open comments, their issue links, capture references.
Without it, a stored state cannot serve as a regression oracle: when the
computation rule later changes and a replay produces a different result, there
is no way to tell a code regression from data that legitimately moved.

Facts that trigger a recomputation:

| Fact recorded | Possible effect |
| --- | --- |
| A review is saved (capture verdicts + comments) | any cycle transition |
| A comment is linked to an issue | comment → `tracked` |
| A comment is discarded with a reason | may clear the last blocker → `reviewed` |
| The dev asks for a judgment on a delivered comment | comment → `to-review`, case → `to-review` |
| A delivery is accepted or refused | comment → `validated` or `refused` |
| An edition is accepted | freshness only — never the cycle state |

Returning to `to-review` is **requested by the dev**, never inferred from
captures moving: images also move for a refactor or a dependency bump, and
summoning the reviewer for that is noise. The dev may ask without having
implemented everything — one issue can depend on the verdict given on another.

## 4. Capture storage

Content-addressed ([ADR 0004](../adr/0004-content-addressed-capture-storage.md)):
a capture is stored under the hash of its bytes, so an image that does not
change between editions is stored **once**. Consequences:

- Full visual history is affordable. Any capture, at any past edition, stays
  retrievable — including diffing a screen against itself three months ago.
- Nothing is ever overwritten. A run arriving mid-review cannot destroy what a
  reviewer is looking at; the case simply keeps pointing at its edition.
- References are pointers, not copies.

**Capture provenance is recorded with every capture**: operating system,
browser and browser version, and a client-declared environment id. Byte
comparison is only meaningful within one environment. ozalid compares captures
from different environments **never silently** — it either refuses the
comparison or flags the whole delta as environment-induced. This is what makes
the "a browser update invalidates everything" failure mode survivable.

## 5. Boundaries

Two hard boundaries keep the product small ([ADR 0003](../adr/0003-runner-and-tracker-agnostic-boundaries.md)):

**ozalid knows nothing about test runners.** The API speaks cases, steps,
variants, captures. Translating a Playwright, Cypress or anything-else report
into that vocabulary is the client's job. Supporting a new runner never touches
the server.

**ozalid knows nothing about issue trackers.** A comment may carry an opaque
external reference — an id, a URL and a title, all three **supplied by the
client**. ozalid never fetches them, never refreshes them, and will never know
the issue was closed. ozalid never creates, reads, closes or
comments an issue, and never holds a tracker credential. The flow runs
**outward**: the reviewer accepts a fix in ozalid, and something else closes
the issue as a consequence. Nothing about the review depends on a third party
being reachable.

## 6. Comments

A comment is a durable entity, not a scratch note
([ADR 0006](../adr/0006-problems-are-durable-entities.md)). It was called a
*problem* until 2026-08-20.

- Fields: kind (`defect` | `improvement`), text, the step it anchors to, the
  variants it appears on, its state, and its history.
- One real defect spanning four variants is **one** comment with four variants
  checked — never four comments.
- The **kind stays on the comment**. It is what the issue is written from, and
  it never colours the case's state
  ([ADR 0012](../adr/0012-case-carries-the-ball-comment-carries-the-detail.md)).

### 6.1 Lifecycle

| State | Meaning | Terminal |
| --- | --- | --- |
| `to-track` | Reported, no issue attached | no |
| `tracked` | Carries an external issue reference | no |
| `to-review` | The dev delivered and asked for a judgment | no |
| `refused` | Refused, with a mandatory remark | no — returns to `to-review` on the next delivery |
| `validated` | Accepted | yes |
| `discarded` | Set aside, with a mandatory reason | yes |

A refusal is **not** a way to die. The dev reworks, delivers again, and the
comment returns to `to-review` — as many rounds as needed. Every refusal is
kept: three round trips on one comment is information.

Nothing is deleted. A discarded comment stays visible on its case, with its
reason and its author. "I reported this three months ago, who removed it?" must
always have an answer.

## 7. Run intake

A client pushes an edition: the manifest (cases, steps, variants) plus the
capture bytes that ozalid does not already hold. Each case is named by the id
ozalid generated for it; a manifest naming the same case twice is refused
([ADR 0014](../adr/0014-server-generated-case-identity-and-catalogue-tree.md)).

An archived case is not part of intake and never blocks it.

Intake is governed by a **per-project policy** ([ADR 0007](../adr/0007-run-intake-policy.md)):

- `strict` — intake is refused outright while any case sits outside
  `{reviewed, to-fix, not-instrumented}` — that is, while any case is
  `to-review`. The refusal lists the blocking cases. This keeps pressure on
  finishing reviews, at the cost of blocking the whole project on one
  unfinished review.
- `per-case` — intake is always accepted and stored; each case keeps pointing
  at the edition its reviewer is judging, and advances when that review ends.

Running the test suite is never gated. Only intake is.

A documented override exists for `strict`, and using it is journalled like any
other fact — an untraceable escape hatch is how a policy quietly dies.

## 8. Identity and access

- **Humans** authenticate through a delegated provider. No passwords stored, no
  reset flow to build, and reviewer identity lines up with the identity they
  already use on their tracker.
- **Machines** — the intake client, automation agents — use service tokens.
  They never borrow a human identity, so the journal can always answer whether
  an action came from a person or a program.

## 9. API shape

Indicative, not a contract yet. The principle is what matters: **every fact
enters through the API, no path writes state behind it.**

```
POST   /projects/:p/editions                  intake a run (manifest + blobs)
GET    /projects/:p/cases?state=…&freshness=…  filter on stored state, no scan
GET    /projects/:p/cases/:id                 case detail, captures, comments
POST   /projects/:p/cases                     create a case, returns its id
PATCH  /projects/:p/cases/:id                 title, description, category
POST   /projects/:p/cases/:id/archive         leave the catalogue, keep everything
GET    /projects/:p/categories                the catalogue tree
POST   /projects/:p/cases/:id/lock            claim / heartbeat / release
POST   /projects/:p/cases/:id/reviews         save a review session
POST   /projects/:p/comments/:id/reference    attach an external issue
POST   /projects/:p/comments/:id/discard      discard with a reason
POST   /projects/:p/comments/:id/delivery     the dev asks for a judgment
POST   /projects/:p/comments/:id/judgment     accept or refuse a delivery
GET    /projects/:p/events                    server-sent event stream
GET    /projects/:p/cases/:id/history         the transition journal
```

There is deliberately **no** endpoint that sets a case's state. State is
written by the server as a consequence of a recorded fact, never as an argument
([ADR 0002](../adr/0002-server-owns-the-review-lifecycle.md)).

The event stream exists so a consumer — a dashboard, or an automation agent —
learns that a case changed without polling and without a human relaying the
information. Who consumes it, and whether that consumer runs permanently, is a
deployment choice the product does not prescribe.

## 10. Out of scope for v1

Named so they are not smuggled in:

- Automatic pass/fail on visual difference. ozalid prompts a human; it never
  judges.
- Hosting or executing test suites. Clients run their own.
- Tracker integration of any kind (see [§5](#5-boundaries)).
- Cross-environment capture comparison. Recorded, flagged, not reconciled.
- Deleting a non-empty category. Only an empty one can be removed
  ([ADR 0014](../adr/0014-server-generated-case-identity-and-catalogue-tree.md)).
- Comparing recordings. Encoding is not deterministic
  ([ADR 0013](../adr/0013-a-recording-is-not-a-capture.md)).
- Migration from the predecessor tool. Deliberately none — the book starts empty
  ([ADR 0008](../adr/0008-no-migration-frozen-predecessor.md)).

## 11. Open questions

To settle before implementation starts. The technology stack, listed here until
2026-08-19, is settled by [ADR 0011](../adr/0011-technology-stack.md).

1. **Hosting and operations** — where it runs, backups, restore drill. Now
   covers two stores, not one: the database and the capture blobs, whose restore
   points must stay consistent with each other
   ([backend ADR 0004](../backend/adr/0004-filesystem-blob-store.md)).
2. **Instrumentation contract** — how a project declares which steps are
   capture-worthy, and how much determinism ozalid demands of a suite before
   byte comparison is meaningful. The predecessor accumulated hard-won recipes
   here (frozen verification codes, purged mailboxes, reset rate limiters);
   they are project-side concerns, but the product should say so explicitly.
3. **Non-fingerprintable steps** — some screens have no deterministic frame
   (animations, timers). The predecessor kept a per-project exemption list;
   confirm that model.
4. **Lock expiry window** — a concrete duration for [§3.2](#32-occupancy-is-someone-working-on-it-right-now).
5. **Retention ceiling** — content addressing makes history cheap, not free.
   Decide whether editions are pruned beyond some horizon, and on what rule.
