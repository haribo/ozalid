# ozalid

A hosted review book for end-to-end test evidence.

Your e2e suite already walks through every user flow. ozalid captures what the
user actually saw at each step — in every variant that matters (desktop/mobile,
light/dark, and whatever else your product ships) — and turns it into a
catalogue a human reviews instead of clicking through the app by hand.

The point is not to assert that pixels did not move. It is to let a reviewer
judge the product from evidence, and then to **only ever re-review what
actually changed**.

> An *ozalid* is the final proof pulled before a print run: the last chance to
> read every page, annotate what is wrong, and sign off.

## Status

Design phase. Nothing is implemented. The specification and the structural
decisions are complete enough to start building.

- [`docs/design/product.md`](docs/design/product.md) — what the product does,
  its vocabulary, its data model, and its review cycle. **Start here.**
- [`docs/adr/`](docs/adr/) — the product and cross-cutting decisions, each with
  the alternatives that were rejected and why.
- [`docs/backend/adr/`](docs/backend/adr/),
  [`docs/frontend/adr/`](docs/frontend/adr/) — the decisions confined to one
  side of the stack.

## Stack

Settled in [ADR 0011](docs/adr/0011-technology-stack.md). One repository holds
both applications; a client pushes evidence by calling the API, and ozalid ships
no client program ([ADR 0015](docs/adr/0015-no-cli-clients-call-the-api.md)).

| | |
| --- | --- |
| `apps/server` | Go — the API, the review lifecycle, the capture store |
| `apps/web` | Vue 3, TypeScript, Vite, Tailwind — the review book |

State lives in PostgreSQL; capture bytes live on the filesystem, addressed by
content. The OpenAPI document is the contract the three share, and it is
authored, not generated.

## Origin

ozalid is the extraction of an internal tool built inside a product monorepo
between July and August 2026. That tool proved the concept over 240 test cases
and is now frozen: its architecture (flat JSON files, derived-on-read state,
direct file edits alongside a partial API) does not survive multiple reviewers
or a hosted deployment. ozalid keeps the concept and rebuilds the foundations.

See [ADR 0001](docs/adr/0001-standalone-multi-project-service.md) for the full
account of what was learned and what it costs.

## License

ozalid is free software under the
[GNU Affero General Public License v3.0](LICENSE).

    Copyright (C) 2026  Nicolas CHAUVIN

Running it — for yourself, for your team, inside a company — carries no
obligation. What section 13 adds over the GPL is this: **offer a modified ozalid
to other people over a network, and you owe them its source.** Self-hosting an
unmodified copy triggers nothing.

That is the whole reason for this licence rather than a permissive one. ozalid
is a service somebody could run for others, not a library, and the AGPL is the
licence written for that shape. It is also the reversible choice: relicensing to
something permissive later is a decision the copyright holder can still make,
while a permission already granted cannot be withdrawn.
