# Code comments

Conventions for inline comments across the server, the CLI and the web client.
Single source of truth; per-language notes at the end.

## Rules

- **Why, not how.** The code already shows what it does. A comment explains
  intent, a constraint, or a trade-off.
- **No paraphrase.** A comment restating the next line is noise.
- **No commented-out code.** Delete it — git history is the archive.
- **No section banners** (`// ===== Helpers =====`). A file needing visual
  separation needs splitting.
- **`TODO` and `FIXME` reference an issue**: `// TODO(#NNN): <context>`. An
  untracked TODO is dead code.
- **English only.**
- **No emojis, no AI references.**

## When a comment earns its place

- A rule that is not derivable from the code: a constraint learned from a real
  incident, a measured performance trade-off.
- A workaround for a known bug in a dependency, with a link upstream.
- A non-obvious invariant the next reader must preserve.

## When it does not

- Restating the code.
- Documenting a language feature or standard-library behaviour.
- Marking the obvious.

## Per language

**Go.** Doc comments on exported symbols follow godoc: one sentence starting
with the symbol name. Keep them short — the signature carries most of the
contract.

**TypeScript and Vue.** JSDoc on an exported symbol where the name does not
convey the contract. One sentence beats a block.

**SQL.** A query whose shape is driven by an index, or by a locking concern,
says so — that is exactly the kind of intent a reader cannot recover from the
statement.
