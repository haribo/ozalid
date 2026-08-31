# Commit conventions

See also: [git-workflow.md](git-workflow.md) for branching and pull-request
rules.

## Format

```
<type>(<scope>): <description>
<type>(<scope>)!: <description>   ← breaking change
```

## Types

`feat` | `fix` | `docs` | `style` | `refactor` | `perf` | `test` | `chore` |
`ci` | `build`

## Scope

Recommended on every commit. It names the area touched — an application, a
domain, a tool.

Examples: `intake`, `lifecycle`, `problems`, `lock`, `api`, `db`, `cli`,
`review`, `ci`, `justfile`

May be omitted for a generic `style` or `chore` spanning the whole repository.

## Breaking changes

Append `!` after the scope:

```
feat(api)!: drop the legacy manifest shape
```

## Squash merge commits

Squash-merging a feature pull request into `develop` makes GitHub append the
pull request number:

```
type(scope): description (#PR)
```

The pull request title follows `type(scope): description` without the suffix —
GitHub adds it.

## Rules

1. Single line — no body, no footer
2. Maximum 72 characters, excluding the appended `(#PR)`
3. Imperative present tense: "add", not "added"
4. No leading capital, no trailing period
5. No AI references, no promotional content
