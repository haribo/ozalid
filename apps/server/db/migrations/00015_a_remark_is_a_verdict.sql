-- +goose Up
-- ADR 0020: validated leaves the product — captures and comments say the same
-- word as the act. And a comment carries no kind: qualifying into fix or
-- feature belongs to whoever writes the issues, never to the reviewer.
ALTER TABLE capture_verdicts DROP CONSTRAINT capture_verdicts_status_check;
UPDATE capture_verdicts SET status = 'accepted' WHERE status = 'validated';
UPDATE capture_verdicts SET status = 'refused' WHERE status = 'to-fix';
ALTER TABLE capture_verdicts ADD CONSTRAINT capture_verdicts_status_check
    CHECK (status IN ('to-review', 'refused', 'accepted'));

ALTER TABLE comments DROP CONSTRAINT comments_state_check;
UPDATE comments SET state = 'accepted' WHERE state = 'validated';
ALTER TABLE comments ADD CONSTRAINT comments_state_check
    CHECK (state IN ('to-track', 'tracked', 'to-review', 'refused', 'accepted', 'discarded'));

ALTER TABLE comments DROP COLUMN kind;

-- +goose Down
ALTER TABLE comments ADD COLUMN kind text NOT NULL DEFAULT 'defect'
    CHECK (kind IN ('defect', 'improvement'));

ALTER TABLE comments DROP CONSTRAINT comments_state_check;
UPDATE comments SET state = 'validated' WHERE state = 'accepted';
ALTER TABLE comments ADD CONSTRAINT comments_state_check
    CHECK (state IN ('to-track', 'tracked', 'to-review', 'refused', 'validated', 'discarded'));

ALTER TABLE capture_verdicts DROP CONSTRAINT capture_verdicts_status_check;
UPDATE capture_verdicts SET status = 'validated' WHERE status = 'accepted';
UPDATE capture_verdicts SET status = 'to-fix' WHERE status = 'refused';
ALTER TABLE capture_verdicts ADD CONSTRAINT capture_verdicts_status_check
    CHECK (status IN ('to-review', 'to-fix', 'validated'));
