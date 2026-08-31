# Issue

Create a GitHub issue following project conventions.

Takes an optional argument: the issue title. Ask for it if absent.

## Instructions

### 1. Read the conventions

Read `docs/git-issues.md` and apply it strictly.

### 2. Collect the details

- Use the title if given as an argument; otherwise ask for a description of the
  problem or the feature, then craft the title
- Ask which `type:` label applies (`bug`, `feature`, `chore`, `docs`) when it is
  not obvious
- Ask about `priority:` when relevant

### 3. Validate the title

Imperative present tense, lowercase, no period, 72 characters maximum, no type
prefix — the type is a label. Fix a violating title and show the correction.

### 4. Draft the body

Write the context: problem statement, expected behaviour, or feature
description. Apply the self-contained rule of `docs/git-issues.md` when the
issue may be implemented without the originating conversation. Present the whole
issue — title, labels, body — for approval.

### 5. Create it

```
gh issue create --title "<title>" --label "<type label>" --body "<body>"
```

Add the priority label when specified. Return the issue URL.
