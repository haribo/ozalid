-- +goose Up

-- A comment's text is the reviewer's draft: it is what the issues are written
-- from, and once a ref is attached the book reads the issue's title. A comment
-- carries one or more refs, each delivered and judged on its own; the comment
-- closes when its last ref does (#138).

CREATE TABLE comment_issues (
    id         text PRIMARY KEY DEFAULT short_id(),
    comment_id text NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    issue_id   text NOT NULL,
    url        text,
    title      text,
    state      text NOT NULL DEFAULT 'tracked'
        CHECK (state IN ('tracked', 'to-review', 'refused', 'validated')),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX comment_issues_comment_idx ON comment_issues (comment_id, created_at);

-- The single ref a comment could carry becomes its first row.
INSERT INTO comment_issues (comment_id, issue_id, url, title, state)
SELECT id, issue_ref, issue_url, issue_title,
       CASE WHEN state IN ('to-review', 'refused', 'validated') THEN state
            ELSE 'tracked' END
FROM comments
WHERE issue_ref IS NOT NULL;

ALTER TABLE comments
    DROP COLUMN issue_ref,
    DROP COLUMN issue_url,
    DROP COLUMN issue_title;

-- A judgment lands on one ref from now on. Old rows keep NULL: they were
-- judgments of the whole comment, which then had at most one ref anyway.
ALTER TABLE comment_judgments
    ADD COLUMN comment_issue_id text REFERENCES comment_issues(id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE comment_judgments DROP COLUMN comment_issue_id;
ALTER TABLE comments
    ADD COLUMN issue_ref text,
    ADD COLUMN issue_url text,
    ADD COLUMN issue_title text;
UPDATE comments c
SET issue_ref = ci.issue_id, issue_url = ci.url, issue_title = ci.title
FROM (SELECT DISTINCT ON (comment_id) * FROM comment_issues ORDER BY comment_id, created_at) ci
WHERE ci.comment_id = c.id;
DROP TABLE comment_issues;
