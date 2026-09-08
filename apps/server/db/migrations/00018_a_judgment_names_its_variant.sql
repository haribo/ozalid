-- +goose Up
-- ADR 0022: a judgment lands on the capture on screen, and the journal says
-- which one. Null for history and for ref-level moves (a refusal's take-back).
ALTER TABLE comment_judgments ADD COLUMN variant_id text REFERENCES variants(id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE comment_judgments DROP COLUMN variant_id;
