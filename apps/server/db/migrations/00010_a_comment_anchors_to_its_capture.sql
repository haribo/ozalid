-- +goose Up

-- A comment anchors to the capture it was written about (#132). The step it
-- displays under is read through that capture; a step's name is a label, and
-- the position it sits at is layout. Production proved the old model wrong:
-- one step inserted mid-flow, and a comment about the sent-state screen hung
-- under the typed-address screen.

ALTER TABLE comment_variants
    ADD COLUMN capture_id text REFERENCES captures(id);

-- Backfill: the capture of the newest edition at or before the comment was
-- written, for the comment's step and this variant. A comment written on a
-- square that had no capture keeps a null anchor, which is what it had to say.
UPDATE comment_variants cv
SET capture_id = (
    SELECT cap.id
    FROM captures cap
    JOIN editions e ON e.id = cap.edition_id
    JOIN comments c ON c.id = cv.comment_id
    WHERE cap.step_id = c.step_id
      AND cap.variant_id = cv.variant_id
      AND e.created_at <= c.created_at
    ORDER BY e.created_at DESC, e.id DESC
    LIMIT 1
);

-- A step is matched by name at intake from now on, and never renamed: a new
-- name at an old position is a new row, and the old row keeps its id, its
-- captures and its verdicts. Position stops being an identity, so it stops
-- being unique — display orders by (position, name).
ALTER TABLE steps DROP CONSTRAINT steps_case_id_position_key;
ALTER TABLE steps ADD CONSTRAINT steps_case_id_name_key UNIQUE (case_id, name);

-- +goose Down
ALTER TABLE steps DROP CONSTRAINT steps_case_id_name_key;
ALTER TABLE steps ADD CONSTRAINT steps_case_id_position_key UNIQUE (case_id, position);
ALTER TABLE comment_variants DROP COLUMN capture_id;
