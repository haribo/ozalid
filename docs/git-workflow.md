# Git workflow

See also: [git-commits.md](git-commits.md) for commit conventions,
[git-issues.md](git-issues.md) for issue conventions.

## Branches

| Branch | Environment | Deploy |
| --- | --- | --- |
| `main` | Production | Auto on push |
| `develop` | Staging | Auto on push |

Both branches are permanent. Never push to either directly — always through a
pull request.

## Issue-first

Every change starts with a GitHub issue, except trivial ones (typo, formatting,
dependency bump) where the pull request alone suffices.

- The issue describes the **what and why**; the pull request describes the
  **how**.
- The branch name carries the issue number.
- The pull request body carries `Closes #N` so the issue closes on merge.

```
issue #12 → branch feat/12-lock-heartbeat → PR "Closes #12" → squash merge
```

## Feature workflow

```bash
# 1. Create the issue
/gh-issue

# 2. Branch from develop, with the issue number
git checkout -b feat/12-lock-heartbeat develop

# 3. Work and commit
/git-commit

# 4. Rebase on develop before opening the pull request
git fetch origin && git rebase origin/develop

# 5. Open the pull request — targets develop, references the issue
/gh-pr-create

# 6. Wait for CI
gh pr checks

# 7. Squash merge
/gh-merge-develop
```

## Release workflow

```bash
# 1. Open develop → main, once staging is validated
gh pr create --base main --head develop

# 2. Wait for CI
gh pr checks

# 3. Merge commit — never squash a release. Set the subject explicitly:
#    left to GitHub, it writes "Merge pull request #N from <owner>/develop",
#    the one non-conventional subject in an otherwise clean history
gh pr merge <n> --merge --subject "chore(release): vX.Y.Z (#<n>)" --body ""

# 4. Tag
git tag vX.Y.Z && git push origin vX.Y.Z
```

## Merge strategy

| Target | Strategy | Command |
| --- | --- | --- |
| Feature → `develop` | **Squash** | `/gh-merge-develop` |
| `develop` → `main` | **Merge commit** | `gh pr merge <n> --merge --subject "chore(release): vX.Y.Z (#<n>)" --body ""` |

**Never merge a feature pull request with `--merge`** — a squash keeps
`develop` readable, one line per delivered change.
**Never leave a merge subject to GitHub** — the release merge lands on `main`,
gets tagged, and a release binary points at that tag. It is immutable in
practice, so the subject is chosen at merge time or never.
**Never target `main` with a feature pull request** — always `develop`.

## CI gating

Workflows are gated by path filters: a pull request that touches none of a
workflow's paths never schedules it. A doc-only change showing no server or web
check is intended, not a missing step.

| Workflow | Triggering paths |
| --- | --- |
| Server CI | `apps/server/**`, `apps/cli/**`, the workflow file |
| Web CI | `apps/web/**`, `apps/server/api/openapi.yaml`, the workflow file |
| PR validation | every pull request |
| Security (secret scan) | every push to a permanent branch, every pull request |

The API document appears in the web filter deliberately: the typed client is
generated from it, so a contract change must re-check the client that consumes
it ([backend ADR 0002](backend/adr/0002-spec-first-openapi.md)).

## Rules

- Never push directly to `main` or `develop`
- One logical change per pull request — split unrelated work
- Keep feature branches short-lived: days, not weeks
- Rebase on `develop` before opening the pull request
- A pull request introducing user-observable behaviour updates `docs/design/` in
  the same diff ([ADR 0009](adr/0009-documentation-strategy.md)). Bug fixes and
  refactors are exempt
- A pull request that changes a structural decision carries its ADR in the same
  diff — never merged first and documented later

## Hooks

The rule above is also enforced mechanically: `.githooks/pre-commit` refuses a
commit made directly on `main` or `develop`. A written rule is read by whoever
takes the time; the hook catches the moment nobody does.

Hooks live in the repository but are **not active in a fresh clone** — git only
runs what `core.hooksPath` points at. Enable them once, per clone:

```bash
git config core.hooksPath .githooks
```

A hook is opt-in per clone and per machine, so it is a convenience, never the
guarantee. What cannot be skipped runs in CI — secret scanning included, which
is why `gitleaks` is a workflow and not only a hook.

## Branch naming

```
feat/12-short-description
fix/34-short-description
refactor/56-short-description
docs/78-short-description
chore/short-description
```

The prefix matches the commit type. The issue number follows the slash.
Kebab-case. The number may be omitted for trivial `chore` and `style` work with
no issue.
