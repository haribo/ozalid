-- +goose Up
-- ADR 0021: a case and its captures say the same word for the same fact.
-- to-fix meant "at least one open comment", and an open comment always comes
-- from a refusal; reviewed said looked-at without the conclusion.
ALTER TABLE cases DROP CONSTRAINT cases_state_check;
UPDATE cases SET state = 'refused'  WHERE state = 'to-fix';
UPDATE cases SET state = 'accepted' WHERE state = 'reviewed';
ALTER TABLE cases ADD CONSTRAINT cases_state_check
    CHECK (state IN ('not-instrumented', 'to-review', 'refused', 'accepted'));

-- The journal keeps what was written when it was written: transitions keep
-- their historical words, exactly as ADR 0015's rename did.

-- +goose Down
ALTER TABLE cases DROP CONSTRAINT cases_state_check;
UPDATE cases SET state = 'to-fix'   WHERE state = 'refused';
UPDATE cases SET state = 'reviewed' WHERE state = 'accepted';
ALTER TABLE cases ADD CONSTRAINT cases_state_check
    CHECK (state IN ('not-instrumented', 'to-review', 'to-fix', 'reviewed'));
