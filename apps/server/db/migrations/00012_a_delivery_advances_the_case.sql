-- +goose Up

-- A delivery advances the case at once (product.md §7, #142): judging a fix
-- means reading the bytes that claim to fix it. Cases already holding a
-- delivered ref were stuck showing the edition from before their fix — this
-- applies the rule to them.
UPDATE cases k
SET current_edition_id = (
    SELECT e.id FROM editions e
    WHERE e.project_id = k.project_id
    ORDER BY e.created_at DESC, e.id DESC
    LIMIT 1
), updated_at = now()
WHERE EXISTS (
    SELECT 1 FROM comment_issues ci
    JOIN comments c ON c.id = ci.comment_id
    WHERE c.case_id = k.id AND ci.state = 'to-review'
);

-- +goose Down
-- Nothing to undo: which edition a case points at is state, not structure.
