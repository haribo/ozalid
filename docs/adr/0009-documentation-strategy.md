# ADR 0009 — Documentation strategy

## Status

Accepted — 2026-08-19

## Context

ADRs 0001–0008 settled what the product is, before any code existed. Writing
code now adds two more kinds of writing — how the product is built, and the
conventions the build follows — and without a rule for where each kind lives,
every session re-litigates the same questions: is this a design change or an
implementation detail? Does it belong in an ADR or in a technical doc? Does it
belong in a doc at all, or in the code?

The failure mode this prevents is documented in
[ADR 0001](0001-standalone-multi-project-service.md): the predecessor spread one
rule across five places. Documentation duplicates the same way code does, and
rots faster, because nothing compiles it.

## Decision

### Audience

Primary: the solo maintainer and the AI assistants working with them. Secondary:
future contributors, served without extra pedagogy. Out of scope: non-technical
readers — marketing material is not documentation and does not live in `docs/`.

### Two sources of truth, strictly separated

| Source | Scope |
| --- | --- |
| `docs/design/` | The product as a user observes it — what it does, what it forbids, its rules and contracts |
| The code, with the OpenAPI document as part of it | How that is implemented |

The OpenAPI document counts as code for this rule: it is the canonical source of
wire format — field names, parameter names, status codes, exact bounds. The
design must never paraphrase it.

**Discriminant test.** If a value can change without changing what a user
observes, it is *how* → code. If the observable behaviour depends on the value
or on its existence, it is *what* → design. "A cap exists on page size" is
design; "the cap is 100" is code.

**Forbidden**: documentation that paraphrases code. Table schemas, function
signatures, file trees, type definitions — anything derivable by reading the
source is not written down a second time.

### Three document types, and no fourth

| Type | Location | Scope |
| --- | --- | --- |
| Design | `docs/design/` | User-observable rules |
| ADR | `docs/adr/`, `docs/backend/adr/`, `docs/frontend/adr/` | One decision, its rejected alternatives, its consequences |
| Technical doc | `docs/backend/*.md`, `docs/frontend/*.md`, `docs/*.md` | Conventions and operational rules that are neither design nor decision |

`docs/adr/` holds decisions that cross the stack or concern the product itself.
A decision confined to one side lives in that side's `adr/` directory.

### Lint-first

Any convention a linter can enforce with reasonable effort **is** a linter rule,
not a documented one. A technical doc carries only what cannot be automated, or
whose automation cost is plainly excessive. When a documented rule becomes
lintable, the linter is configured and the doc entry is deleted — the rationale
stays in its ADR.

The division of labour: **the linter enforces, the ADR explains, the technical
doc covers the rest.**

### The four-point test

Applies to a new technical doc, a new section in one, and to changing an
existing rule. Skipped for typos and reformulations. All four are required:

1. **Single concern** — the topic fits in one word. `database` passes;
   `security` does not.
2. **Not in the code** — it says something types, signatures and generated
   structure do not already say.
3. **No duplicate** — `grep` confirms the content exists nowhere else.
4. **Spontaneously missed** — remove it, hand the repository to a senior
   developer, and they would eventually write it themselves.

### Lifecycle

**ADRs are append-only.** In-place edits are for corrections of form and for
clarifications that do not change the decision. A reversal is a **new** ADR;
both sides carry the link — `Superseded by ADR NNNN` on the old, `Supersedes ADR
MMMM` on the new. A one-sided link is how the chain rots. An ADR whose decision
no longer applies and has no replacement is marked `Deprecated`. An ADR is never
deleted and never moved.

**Design and technical docs move with the code.** A change that alters
user-observable behaviour updates `docs/design/` in the same diff. If the design
is silent on a behaviour that is needed, the design is written first.

**The code is never the source of truth.** A disagreement between code and
design means the code is the bug, or the design needs an explicit amendment —
never both silently.

## Alternatives rejected

**No strategy, decide case by case.** Rejected: it is the status quo the
predecessor demonstrated, and it produces exactly the duplication
[ADR 0001](0001-standalone-multi-project-service.md) exists to escape.

**One documentation type, everything in `docs/`.** Rejected: design and decision
have different lifecycles. Design is amended as the product changes; a decision
is never amended, only superseded. Merging them means either editing away the
history or freezing the design.

**Document conventions rather than lint them.** Rejected on evidence: a
documented rule with no enforcement drifts, and the drift is invisible until
someone audits. Lint-first exists because documentation is not a control.

## Consequences

- A rule that could be linted but is written down instead is a defect, and the
  fix is an issue to configure the linter.
- Adding a technical doc is a deliberate act that passes the four-point test.
  The default answer to "should this be documented?" is no.
- ADR numbering is per directory and never reused, including for superseded
  records.
- Each ADR directory carries a `README.md` index; adding an ADR updates it in
  the same diff.

## Cross-references

- [ADR 0001](0001-standalone-multi-project-service.md) — the duplication this
  strategy is designed to prevent
- `docs/design/product.md` — the design document this governs
