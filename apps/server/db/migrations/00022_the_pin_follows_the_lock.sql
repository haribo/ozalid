-- +goose Up
-- The displayed edition is derived, never stored (ADR 0024): a live lock
-- pins the bytes stamped at claim, a free case reads at the latest. The
-- stored pin and its maintenance go.
ALTER TABLE case_locks
    ADD COLUMN edition_id text REFERENCES editions(id) ON DELETE SET NULL;
DROP INDEX cases_current_edition_idx;
ALTER TABLE cases DROP COLUMN current_edition_id;

-- +goose Down
ALTER TABLE cases
    ADD COLUMN current_edition_id text REFERENCES editions(id) ON DELETE SET NULL;
CREATE INDEX cases_current_edition_idx ON cases (current_edition_id);
ALTER TABLE case_locks DROP COLUMN edition_id;
