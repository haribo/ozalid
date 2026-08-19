# Claude guidelines

AI directives only — permissions, guardrails, pointers to documentation.
Project conventions belong in `docs/`.
Rules must be concise. One rule per line where possible.

This file takes precedence over auto-memory. If a memory contradicts a rule
here, follow this file and fix the memory.

## General

- The project uses `just` (justfile), never `make`
- File names are lowercase kebab-case. No underscores, no mixedCase
- Check the existing documentation before creating a file
- Never name a file path in a recommendation without verifying it exists in the
  current session — mark it "to confirm" otherwise
- Every written artifact — documentation, code, commit message, issue, pull
  request — is in English. User-facing strings live in i18n locale files
- Responses stay under 15 lines by default. Tables only when tabular beats
  prose. Background and rationale only when asked
- When asked for an opinion, be severe and honest. The goal is code that meets a
  professional standard, not agreement. No flattery, no hedging, no false
  balance. Verdict in one line, then three bullets of substance at most
- Say plainly when something is wrong, and say so when it is right — an unearned
  validation is a defect
- Push back on over-engineering, incoherence and unjustified additions,
  including when the user proposes them
- Cite an established standard (RFC, WCAG, a language idiom) rather than a
  personal preference, and propose the correction — never mere opposition, never
  a strawman of the user's position
- The user decides. Challenge until the decision, then execute it in full. If a
  debate cycles three times on the same axis without converging, propose to
  decide
- Never assume a produced artifact matches its request. Inspect what was
  actually produced, state the invariant it must satisfy, and where the
  invariant is checkable, write the check. Claiming "it now matches" without
  verifying is a defect
- Flag security, performance and design issues the moment they are noticed

## Documentation

- All documentation lives under `docs/`, never inside `apps/` or a code
  directory. The root `README.md` and tool configuration files are not
  documentation
- Documentation strategy — audience, sources of truth, document types,
  lifecycle, the lint-first principle, the four-point test: see
  `docs/adr/0009-documentation-strategy.md`
- `docs/design/` is the source of truth for what the product does
- `docs/backend/` and `docs/frontend/` describe how it is implemented; they
  reference the design, never restate it
- The code is never the source of truth. A disagreement between code and design
  means the code is the bug, or the design needs an explicit amendment — never
  both silently
- If the design is silent on a behaviour that is needed, write the design first
- Anchor a confirmed non-obvious decision — especially one where an alternative
  was rejected — in the design or an ADR before building on it
- An ADR is never deleted. A reversal is a new ADR, and both sides carry the
  link. An ADR with no replacement whose decision no longer applies is marked
  `Deprecated`
- Adding an ADR updates the index `README.md` of its directory in the same diff

## Design and ADRs

- Modifying, deleting or adding a document under `docs/design/` or any `adr/`
  directory: FORBIDDEN without explicit consent — propose first, wait
- Trivial fixes (typo, broken link, markdown formatting) need no consent

## Git

- Commit conventions: `docs/git-commits.md`, strictly
- Workflow: `docs/git-workflow.md`, strictly
- Issue conventions: `docs/git-issues.md`, strictly
- For a commit, a pull request, a merge into `develop` or an issue, invoke the
  matching slash command yourself (`/git-commit`, `/gh-pr-create`,
  `/gh-merge-develop`, `/gh-issue`). Never run `git commit`, `gh pr create`,
  `gh pr merge` or `gh issue create` directly, and never ask the user to type
  the command. Respect each playbook's approval gates
- Before making a change, check the branch. On `main` or `develop`, propose a
  branch name and wait
- **Issue verification gate**: before implementing any issue, audit it against
  the current code and design — never trust the issue text. Cite `file:line` for
  every claim confirmed, and state what could not be confirmed. A claim that
  proves false is written into the issue as a correction comment, never dropped
- Once verified and before implementing: explain the problem simply in the
  conversation and wait for the user to validate that explanation. Trivial
  changes are exempt
- An unrelated bug found while working becomes an issue, never a fix in the
  current branch
- No AI references in commits, code, issues or pull requests

## Server and CLI (`apps/server/`, `apps/cli/`)

- Follow `docs/backend/code.md` and `docs/code-comments.md` strictly
- Layering is enforced by linter — see `docs/backend/adr/0001-hexagonal-layers.md`.
  A violation is fixed by adding an interface, never by an import shortcut
- The OpenAPI document is the source of truth. Edit `apps/server/api/src/`, then
  regenerate. Never edit the bundled `apps/server/api/openapi.yaml`
- Adding an endpoint: allowed. Modifying or deleting an existing endpoint:
  FORBIDDEN without explicit consent
- No endpoint accepts a case state as an argument — state is written by the
  server as a consequence of a recorded fact
  (`docs/adr/0002-server-owns-the-review-lifecycle.md`)
- The capture hash is a cross-application contract. Changing the algorithm or
  its encoding requires a new ADR — it orphans every blob already stored

## Web client (`apps/web/`)

- Follow `docs/frontend/code.md` and `docs/code-comments.md` strictly
- Layers and slice public APIs: `docs/frontend/adr/0002-fsd-architecture.md`
- Never modify server code from client work — propose a server fix instead
- The generated OpenAPI client is the only path to the API
- Run end-to-end tests headless, never `--headed`
- After a code change, propose the tests that cover it

## UI changes

- A pull request **modifies the UI** when it changes what an end user sees. A
  refactor producing identical pixels is exempt and says so explicitly in the
  body
- **Mockup first**: before implementing any UI-modifying change, produce a
  mockup (Artifact: static HTML of the touched surfaces and states, light and
  dark, realistic data including edge cases — long texts, crowded grids) and
  obtain explicit validation before writing code. When variants are debated, the
  artifact shows them side by side
- Exemptions: a provably pixel-identical refactor, and a fix restoring an
  existing rendering with no new surface
- Do not open a pull request for a UI-modifying change without explicit visual
  validation first
- Capture both themes for every touched surface, no exemption. Theme-blind
  reasoning on a diff has proven unreliable
- State what to check: one line per screenshot naming the elements and the
  expected outcome

## Testing

- Never modify an existing test without explicit approval
- A test failing after a change is reported, never silently fixed
- Adding a test is always allowed
- Propose the covering test in the same response as the code change
- A bug fix starts with a failing test that reproduces the bug: write it, watch
  it fail, fix, watch it pass
- That test stays as the regression test and references the issue number, so a
  later reader does not delete it as noise
- A bug fix reproduces the failure from observed evidence — logs, a network
  capture, repro steps. Never from a hypothesis
