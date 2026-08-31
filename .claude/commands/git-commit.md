# Commit

Commit all pending changes following project conventions. On a feature branch,
rebuild the branch history so commits stay clean and logical.

## Instructions

1. Read `docs/git-commits.md` and apply it strictly
2. Run `git branch --show-current`. On `main` or `develop`, propose a branch name
   following `docs/git-workflow.md` and **wait** for the user to switch
3. Run `git status` and `git diff` to understand every pending change
4. Run `git diff develop...HEAD --name-only` for files already committed on the
   branch
5. Run `git log --oneline develop..HEAD` for the existing branch commits
6. Analyse all changed files together, pending and committed:
   - Group by logical step — tightly coupled changes belong in one commit
   - Each commit is a finalized step, never work in progress
   - A signature change and its call sites are one commit
7. If pending changes affect files already committed, or if the grouping no
   longer reflects logical steps:
   - `git reset --soft develop` to collapse the branch
   - Re-commit everything in logical groups
8. Otherwise commit the pending changes, grouped logically
9. Present the plan — files per commit, with messages — and wait for approval

## Rules

- Never push
- Nothing stays uncommitted after execution
- Every message follows `docs/git-commits.md` strictly
