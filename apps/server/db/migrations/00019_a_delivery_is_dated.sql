-- +goose Up
-- A standing refusal is one the dev has not answered (#212): telling it apart
-- from an answered one needs the moment of the last delivery on the ref.
ALTER TABLE comment_issues ADD COLUMN delivered_at timestamptz;

-- +goose Down
ALTER TABLE comment_issues DROP COLUMN delivered_at;
