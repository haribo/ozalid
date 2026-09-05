-- +goose Up
-- Taking a validation back on a capture that turned validated by an accepted
-- reference takes the judgment back (#167). The take-back joins the judgment
-- history: who un-accepted, and when, is information exactly like the
-- acceptance was (ADR 0012).
ALTER TABLE comment_judgments DROP CONSTRAINT comment_judgments_verdict_check;
ALTER TABLE comment_judgments ADD CONSTRAINT comment_judgments_verdict_check
    CHECK (verdict IN ('accepted', 'refused', 'taken-back'));

-- +goose Down
DELETE FROM comment_judgments WHERE verdict = 'taken-back';
ALTER TABLE comment_judgments DROP CONSTRAINT comment_judgments_verdict_check;
ALTER TABLE comment_judgments ADD CONSTRAINT comment_judgments_verdict_check
    CHECK (verdict IN ('accepted', 'refused'));
