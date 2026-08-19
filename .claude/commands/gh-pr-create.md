# Pull request

Run the local gates, push, and open a pull request targeting `develop`.

Takes an optional argument: the pull request title. Generate one from the
commits if absent.

## Instructions

### 1. Validate the branch

`git branch --show-current`. On `main` or `develop`: **refuse** — a feature
branch is required.

### 2. Resolve the issue reference

- Parse the issue number from the branch name (`type/NUMBER-description`)
- No number found and the title does not start with `chore` or `style`:
  **refuse** and ask for the issue number
- Title starting with `chore` or `style`: skip, no reference required

### 3. Local gates — stop at the first violation, do not push on failure

#### 3.1 Lint and typecheck

- Server and CLI: `just be-check`
- Web: `just fe-check`

#### 3.2 Unit tests

- Server and CLI: `just be-test`
- Web: `just fe-test`

#### 3.3 End-to-end

Any change that can affect user-observable behaviour, on either side, runs
`just fe-test-e2e`.

Only exemptions:

- A refactor with provably identical output, the proof referenced in the body
- Documentation only (`docs/**`, `CLAUDE.md`, markdown outside code)
- Configuration that does not alter runtime behaviour

"This component is not covered" is not a valid skip. The gate runs the suite and
lets the suite decide.

#### Pre-existing failures — fail closed

A failure unrelated to this branch still fails the gate. Do not bypass. Open an
issue documenting it and stop.

#### 3.4 Test coverage — hard stop, answer with evidence

Two acceptable answers:

(a) "Existing tests cover this" — quote the spec paths and the grep output of
the assertions.
(b) "Tests added here" — list the spec files from
`git diff develop...HEAD --name-only`.

Unacceptable, treated as failure: "manually verified", "lint passes", "I think
so", "no test required", silence. If neither answer holds: stop, add tests,
re-run § 3.1–3.3.

### 4. Prepare the content

In parallel: `git log --oneline develop..HEAD` and
`git diff develop...HEAD --stat`.

- **Title**: `type(scope): description`, under 70 characters, no `(#N)` suffix —
  it is appended on squash merge
- **Body**: summary bullets, test plan, `Closes #N`

### 5. Push and open

```
git push -u origin <branch>
gh pr create --title "<title>" --body "$(cat <<'BODY'
## Summary
<1-3 bullets>

## Test plan
<checklist>

Closes #<N>
BODY
)"
```

Omit `Closes` for `chore` and `style`.

### 6. Output

Return the pull request URL.
