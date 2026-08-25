-- Every category with the case counts of its whole descendance.
--
-- A recursive CTE rather than a walk: a branch in trouble has to be visible
-- from the root, and drawing one bar must not cost one query per node.
-- name: CategoryTreeWithCounts :many
WITH RECURSIVE descendants AS (
    -- Each category is its own descendant, so a node's own cases count too.
    SELECT id AS root_id, id AS node_id
    FROM categories
    WHERE project_id = $1
  UNION ALL
    SELECT d.root_id, c.id
    FROM categories c
    JOIN descendants d ON c.parent_id = d.node_id
)
SELECT
    cat.id,
    cat.parent_id,
    cat.name,
    cat.position,
    count(k.id)                                                    AS cases,
    count(k.id) FILTER (WHERE k.state = 'not-instrumented')        AS not_instrumented,
    count(k.id) FILTER (WHERE k.state = 'to-review')               AS to_review,
    count(k.id) FILTER (WHERE k.state = 'to-fix')                  AS to_fix,
    count(k.id) FILTER (WHERE k.state = 'reviewed')                AS reviewed,
    max(k.updated_at)::timestamptz                                 AS last_activity
FROM categories cat
LEFT JOIN descendants d ON d.root_id = cat.id
LEFT JOIN cases k ON k.category_id = d.node_id AND k.archived_at IS NULL
WHERE cat.project_id = $1
GROUP BY cat.id, cat.parent_id, cat.name, cat.position
ORDER BY cat.parent_id NULLS FIRST, cat.position, cat.name;

-- Every case with the state of its captures at the edition it points at.
--
-- The verdict rows only exist once a reviewer has written something, so a case
-- that has captures but no verdict reports them as still to judge.
-- name: CasesWithCaptureCounts :many
WITH latest AS (
    SELECT id FROM editions
    WHERE project_id = $1
    ORDER BY created_at DESC, id DESC
    LIMIT 1
)
SELECT
    k.*,
    count(c.id)                                                        AS captures,
    count(*) FILTER (WHERE v.status = 'validated')                     AS validated,
    count(*) FILTER (WHERE v.status = 'to-fix')                        AS commented,
    count(c.id) FILTER (WHERE v.status IS NULL OR v.status = 'to-review') AS to_judge,
    max(e.created_at)::timestamptz                                     AS last_edition
FROM cases k
LEFT JOIN steps s ON s.case_id = k.id
LEFT JOIN captures c ON c.step_id = s.id AND c.edition_id = (SELECT id FROM latest)
LEFT JOIN editions e ON e.id = c.edition_id
LEFT JOIN capture_verdicts v
       ON v.case_id = k.id AND v.step_id = s.id AND v.variant_id = c.variant_id
WHERE k.project_id = $1
  AND k.archived_at IS NULL
  AND (sqlc.narg('category_id')::text IS NULL OR k.category_id = sqlc.narg('category_id')::text)
GROUP BY k.id
ORDER BY k.title;

-- Everything the state computation reads, for one case at one edition.
-- name: CaseCaptureCells :many
SELECT s.id AS step_id, c.variant_id
FROM steps s
JOIN captures c ON c.step_id = s.id AND c.edition_id = $2
WHERE s.case_id = $1;

-- name: CaseValidatedCells :many
SELECT step_id, variant_id FROM capture_verdicts
WHERE case_id = $1 AND status = 'validated';

-- name: CaseComments :many
SELECT c.id, c.step_id, c.kind, c.body, c.state, c.issue_ref, c.issue_url,
       c.issue_title, c.discard_reason, c.author_id, c.created_at, c.updated_at,
       array_remove(array_agg(cv.variant_id), NULL)::text[] AS variant_ids
FROM comments c
LEFT JOIN comment_variants cv ON cv.comment_id = c.id
WHERE c.case_id = $1
GROUP BY c.id
ORDER BY c.created_at;

-- name: CreateComment :one
INSERT INTO comments (case_id, step_id, kind, body, author_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: AttachCommentVariant :exec
INSERT INTO comment_variants (comment_id, variant_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- The verdict of a cell is recomputed, never set by a caller: recording a
-- comment and recomputing what it covers happen together (ADR 0012).
-- name: UpsertCaptureVerdict :exec
INSERT INTO capture_verdicts (case_id, step_id, variant_id, status)
VALUES ($1, $2, $3, $4)
ON CONFLICT (case_id, step_id, variant_id)
DO UPDATE SET status = EXCLUDED.status, updated_at = now();

-- name: SetCaseState :exec
UPDATE cases SET state = $2, updated_at = now() WHERE id = $1;

-- name: GetComment :one
SELECT * FROM comments WHERE id = $1;

-- The state is written by the server as a consequence of a recorded move,
-- never received as an argument (ADR 0002).
-- name: SetCommentState :exec
UPDATE comments SET state = $2, updated_at = now() WHERE id = $1;

-- name: AttachIssue :exec
UPDATE comments
SET state = $2, issue_ref = $3, issue_url = $4, issue_title = $5, updated_at = now()
WHERE id = $1;

-- name: DiscardComment :exec
UPDATE comments SET state = $2, discard_reason = $3, updated_at = now() WHERE id = $1;

-- Every judgment is kept: three round trips on one comment is information
-- (ADR 0012).
-- name: RecordJudgment :exec
INSERT INTO comment_judgments (comment_id, verdict, remark, actor_id)
VALUES ($1, $2, $3, $4);

-- name: CommentJudgments :many
SELECT * FROM comment_judgments WHERE comment_id = $1 ORDER BY created_at;

-- A review that ends releases the case onto the project's most recent edition:
-- it was only held back so the reviewer judged one fixed set of bytes.
-- name: ReleaseToLatestEdition :exec
UPDATE cases k
SET current_edition_id = (
        SELECT e.id FROM editions e
        WHERE e.project_id = k.project_id
        ORDER BY e.created_at DESC, e.id DESC
        LIMIT 1
    ),
    updated_at = now()
WHERE k.id = @case_id;
