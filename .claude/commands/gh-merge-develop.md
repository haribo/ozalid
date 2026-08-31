# Merge into develop

Merge a feature pull request into `develop` with strict validation.

Takes an optional argument: pull request number or branch name. Detected from
the current branch if absent.

## Instructions

### 1. Resolve the pull request

A number is used directly; a branch name resolves to its open pull request
targeting `develop`; no argument means the current branch. No pull request
found: **refuse**.

### 2. Collect the information

In parallel:

- `gh pr view <n> --json number,title,state,baseRefName,headRefName,mergeable,reviews,statusCheckRollup,commits,body,labels`
- `gh pr checks <n>`
- `gh pr diff <n>`

### 3. Validate — collect every violation, report them together

| # | Check | Rule |
| --- | --- | --- |
| 1 | State | must be `OPEN` |
| 2 | Base | must be `develop` — refuse any other target |
| 3 | Mergeability | must be `MERGEABLE`; on `CONFLICTING`, demand a rebase |
| 4 | CI | every required check green — no pending, no failure |
| 5 | Reviews | zero `CHANGES_REQUESTED` |
| 6 | Diff | read it fully — flag debug prints, stray TODO/FIXME, hardcoded secrets, commented-out code, unrelated changes |

Any failure: report everything and **stop**.

### 4. Craft the squash message

- Read `docs/git-commits.md`
- Derive `type(scope):` from the actual changes, not blindly from the title
- One line, 72 characters maximum excluding the suffix, imperative, no capital,
  no period
- Append ` (#<n>)` — required, since `--subject` overrides GitHub's auto-append
- Use `type(scope)!:` for a breaking change
- Present the message for approval before merging

### 5. Merge

```
gh pr merge <n> --squash --delete-branch --subject "<approved message>" --body ""
```

On failure: report, do not retry.

### 6. Clean up

`git checkout develop`, `git pull origin develop`, delete the local branch if it
survives, confirm with `git log --oneline -1`.

## Output

```
PR #<n> merged into develop
  <hash> <message>
  Branch <branch> deleted (remote + local)
```
