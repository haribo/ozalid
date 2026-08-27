# ADR 0019 — Two kinds of account, one set of rights

## Status

Accepted — 2026-08-27

Supersedes part of [ADR 0018](0018-an-actor-is-never-invented.md): what a
person's identity is, and how they prove it. The rest of 0018 stands — a token
belongs to one project, and the rows already written as `anonymous` stay.

Rewrites `docs/design/product.md` § 8.

## Context

ADR 0018 settled what ozalid writes down once identity is proven, and assumed
§ 8's delegated provider without examining it. Building the first piece raised
the question it had skipped: **what does a person present, and what does a
program present?**

The tempting answer was one account type — a name, and nothing else, whether it
belongs to a person or to an agent. It is smaller, and it fails on the product's
own premise. ozalid exists because *a person looks at the screens*. A book that
cannot separate "somebody looked" from "a script stamped two hundred captures"
answers a weaker question than the one it was built for.

The distinction was already there and already worthless: `journal.actor_kind`
was **declared**, never verified, and until recently was *inferred from the
action* — a developer marking a fix delivered by hand was recorded as a program.

Every large identity system solves this the same way, and none of them use a
flag: a person and a program are **different kinds of account**, each proving
itself differently, both landing in one set of rights. Microsoft states plainly
that a user account must not be used as a service account.

## Decision

**A person is a `user`**: an id, a name, and a unique email address.

**A program is a `service_account`**: an id, a name, the project it belongs to,
and the user who owns it. A machine account with no owner is one nobody
revokes.

**The kind is derived, never declared.** It is read from how the caller
authenticated: a person came through a sign-in, a program presented a token.
That is what makes the journal's answer worth keeping — a claim nobody checks
is not evidence.

**Rights belong to membership, never to the kind.** A user belongs to N
projects with rights on each; a service account belongs to one. Nothing is
forbidden because the caller is a program. An agent that pushes captures and a
person who pushes captures do the same thing, and only the journal separates
them.

**No password.** A person signs in through a link sent to their email address.
There is nothing to forget, so there is no recovery procedure to write: the
recovery *is* the sign-in, which means one mechanism instead of two.

**The journal holds no foreign key to either table.** It keeps `actor_id` and
`actor_kind` as it already does. Evidence has to survive the account that
produced it; a foreign key would either block deleting an account or take the
history with it.

## Alternatives rejected

**One account type, a name and nothing else.** Rejected: the kind then becomes a
declaration nobody verifies, and in a year, reading `validated by atlas-ci`,
nobody will remember which names were agents. The information that matters most
when re-reading the book would be the one carried worst.

**Keeping a declared `kind` column on a single table.** Same defect with more
ceremony: it looks structural and is not.

**Passwords.** Rejected: a secret to store, an algorithm to keep current, and a
reset flow to build. § 8 refused them from the start, and nothing has changed
except that we now know what to put in their place.

**A delegated provider** — ADR 0018's answer. Deferred rather than rejected on
merit: it is more machinery than this product needs today, and it is the largest
piece of work in the identity epic. The `email` column is what keeps the door
open, since a provider's subject can be matched to it later.

**A foreign key from the journal to the accounts.** Rejected: see above.
Deleting a departed colleague's account must not delete what they reviewed.

## Consequences

- `docs/design/product.md` § 8 is rewritten. The delegated provider leaves it;
  the promise it existed for — no password stored, no reset flow to build — is
  kept by other means.
- **Sending mail becomes a dependency of v1.** Until it exists, an administrator
  hands a personal token over out of band and the email address sits unused. The
  data does not change when mail sending arrives; only the door does.
- `journal.actor_kind` keeps its meaning and gains the trustworthiness it never
  had.
- A service account has an owner, so "who is responsible for this program?" has
  an answer — which is what makes revocation possible rather than theoretical.
- Rights need a home: a project membership carrying what its holder may do. That
  is a separate decision and it is not taken here.

## Cross-references

- [ADR 0018](0018-an-actor-is-never-invented.md) — partly superseded; what
  survives is the project-scoped token and the untouched `anonymous` rows
- [ADR 0001](0001-standalone-multi-project-service.md) — the project as a
  first-class boundary, which is what membership attaches to
- [Google Cloud — service accounts](https://cloud.google.com/iam/docs/service-account-overview),
  [Microsoft Entra — securing service accounts](https://learn.microsoft.com/en-us/entra/architecture/secure-service-accounts),
  [Databricks — service principals](https://docs.databricks.com/gcp/en/admin/users-groups/service-principals)
- `docs/design/product.md` § 8
