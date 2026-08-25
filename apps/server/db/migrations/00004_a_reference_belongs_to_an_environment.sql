-- +goose Up
-- +goose StatementBegin

-- Two machines rendering the same screen produce different bytes -- font
-- rasterisation, antialiasing, a scrollbar of another width. With one reference
-- row per square, every alternation between CI and a laptop reports the whole
-- case as moved. That is the always-on alert `to-re-watch` was removed for
-- (ADR 0004, ADR 0017).
--
-- The empty string is the unnamed environment: a client that declares none
-- still compares against itself.
ALTER TABLE capture_references
    ADD COLUMN environment_id text NOT NULL DEFAULT '';

ALTER TABLE capture_references
    DROP CONSTRAINT capture_references_pkey,
    ADD CONSTRAINT capture_references_pkey
        PRIMARY KEY (case_id, step_id, variant_id, environment_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE capture_references
    DROP CONSTRAINT capture_references_pkey,
    ADD CONSTRAINT capture_references_pkey
        PRIMARY KEY (case_id, step_id, variant_id);

ALTER TABLE capture_references DROP COLUMN environment_id;

-- +goose StatementEnd
