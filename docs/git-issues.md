# Issue conventions

See also: [git-commits.md](git-commits.md), [git-workflow.md](git-workflow.md).

## Title

A pure imperative description. The type is carried by a label, never by the
title.

1. Imperative present tense: "add", not "added"
2. Lowercase, no trailing period
3. Maximum 72 characters
4. Stands alone without the conversation that produced it

## Labels

### Type — required, exactly one

| Label | Usage |
| --- | --- |
| `type: bug` | defect or malfunction |
| `type: feature` | new capability or improvement |
| `type: chore` | CI, tooling, maintenance |
| `type: docs` | documentation change |

### Priority — optional

| Label | Usage |
| --- | --- |
| `priority: critical` | needs immediate attention |

### State — optional, closing qualifiers

| Label | Usage |
| --- | --- |
| `state: wontfix` | will not be worked on |
| `state: duplicate` | already exists |
| `state: invalid` | not a valid issue |

## Self-contained bodies

If an issue may be implemented by someone without the conversation that produced
it — another session, another contributor, or the author in six months — the
body must be executable without asking a question:

- **Why**, in one to three lines
- **Every decision already taken**, with no `TBD` unless genuinely open
- **Explicit mappings** for a refactor: current state → target state, per
  artifact
- **Validation criteria**: concrete checks, grep patterns, file lists, test names
- **Pull-request strategy** when the work splits
- **Out of scope**, so nothing creeps in at implementation time

Trivial issues are exempt. The test: if the implementer would have to ask "what
did you mean?" or "A or B?", the issue is incomplete.

## Epics

When a change splits into three or more related pieces of one subsystem, write
an epic plus short sub-issues rather than independent issues.

- **The epic carries the shared context once**: why, the decisions locked in
  discussion, the doc-anchoring plan, what is out of scope. It lists the
  sub-issues as a checklist.
- **Sub-issues stay short**: `Part of #<epic>`, a Build section, a Validation
  section. No repetition of the epic.
- **Do not over-epic.** An isolated change stays a plain issue. The threshold is
  design coherence, not size.

## Anchoring decisions

A decision taken in conversation — especially one where an alternative was
rejected — becomes an issue targeting the document that should carry it:
`docs/design/product.md` for product behaviour, an ADR directory for a
structural decision. Trigger when all three hold: the decision is not derivable
from the code, an alternative was explicitly rejected, and it will influence
later decisions.

## Examples

| Title | Labels |
| --- | --- |
| `release an expired lock on heartbeat timeout` | `type: feature` |
| `keep a discarded problem visible on its case` | `type: bug`, `priority: critical` |
| `add commitlint to the ci pipeline` | `type: chore` |
