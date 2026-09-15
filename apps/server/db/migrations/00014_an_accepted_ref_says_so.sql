-- +goose Up
-- The reviewer's yes on a ref is an acceptance, and the state says the act:
-- validated is the vocabulary of captures, and nothing else (#170).
ALTER TABLE comment_issues DROP CONSTRAINT comment_issues_state_check;
UPDATE comment_issues SET state = 'accepted' WHERE state = 'validated';
ALTER TABLE comment_issues ADD CONSTRAINT comment_issues_state_check
    CHECK (state IN ('tracked', 'to-review', 'refused', 'accepted'));

-- +goose Down
ALTER TABLE comment_issues DROP CONSTRAINT comment_issues_state_check;
UPDATE comment_issues SET state = 'validated' WHERE state = 'accepted';
ALTER TABLE comment_issues ADD CONSTRAINT comment_issues_state_check
    CHECK (state IN ('tracked', 'to-review', 'refused', 'validated'));
