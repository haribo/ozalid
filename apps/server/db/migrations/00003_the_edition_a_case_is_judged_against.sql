-- +goose Up
-- +goose StatementBegin

-- The edition a case is read and judged against.
--
-- `product.md` §7 promised this pointer with the `per-case` policy and the
-- schema never carried it, so the grid fell back to the project's most recent
-- edition. A run landing mid-review then moved the ground under the reviewer:
-- they judged one set of bytes and the server showed another.
--
-- The pointer holds while the case is `to-review` -- somebody is looking -- and
-- advances once the review ends.
ALTER TABLE cases
    ADD COLUMN current_edition_id text REFERENCES editions(id) ON DELETE SET NULL;

CREATE INDEX cases_current_edition_idx ON cases (current_edition_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX cases_current_edition_idx;
ALTER TABLE cases DROP COLUMN current_edition_id;

-- +goose StatementEnd
