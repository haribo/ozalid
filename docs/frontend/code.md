# Web client code conventions

Rules that a linter cannot enforce. Everything a linter can enforce **is** a
linter rule, per the lint-first principle of
[ADR 0009](../adr/0009-documentation-strategy.md).

Comments: see [`docs/code-comments.md`](../code-comments.md).

Layers, slices and public APIs: see
[frontend ADR 0002](adr/0002-fsd-architecture.md).

## Themes

Every colour utility carries its `dark:` counterpart. A component that renders
correctly in one theme only is a defect, not an omission — this product exists
to display light and dark renderings side by side
([`product.md` § 2](../design/product.md)), and a book that cannot render both
has no standing to judge them.

Theme-blind reasoning about a diff is not a substitute for looking: check both.

## The language of the interface

The interface is written **in English**, in the component that shows the words.
There is one locale and no i18n layer.

That is a decision, not an omission. An indirection through locale files buys
nothing while there is a single locale to read from, and it costs a lookup on
every string somebody wants to change. It is introduced the day a second locale
is needed, and not before.

Two consequences worth knowing:

- Dates go through `Intl.DateTimeFormat('en-GB', …)`
  (`apps/web/src/shared/lib/format.ts`). British rather than American so the
  clock stays on 24 hours: evidence carries timestamps, and `09:20 PM` is one
  reading too many.
- End-to-end selectors quote these words. Changing one breaks the suite, which
  is the point — a string nothing looks for is a string nothing checks.

## Chrome versus content

The interface frames someone else's screenshots. A reviewer must never wonder
whether a border, a shadow or a background belongs to the capture or to ozalid.

- A capture sits in a container that is visually inert, on a neutral surface.
- No decorative effect over or around a capture — no gradient, no blur, no
  rounded corner cutting into the image.
- Chrome uses the interface's own palette, kept distinct from anything a
  captured product is likely to use.

## Talking to the server

The generated OpenAPI client is the only path to the API
([backend ADR 0002](../backend/adr/0002-spec-first-openapi.md)). A hand-written
`fetch` against an endpoint bypasses the contract and is a defect.

## Testing

Unit tests are colocated:

```
CaseGrid.vue
CaseGrid.spec.ts
```

A new source file ships with its `.spec.ts` — barrels and pure type files
excepted. End-to-end tests run headless; `--headed` is for local debugging only.
