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
SELECT c.id, c.step_id, c.kind, c.body, c.state,
       c.discard_reason, c.author_id, c.created_at, c.updated_at,
       array_remove(array_agg(cv.variant_id), NULL)::text[] AS variant_ids
FROM comments c
LEFT JOIN comment_variants cv ON cv.comment_id = c.id
-- Scoped by the project, not merely filtered by the case: a case id from
-- another project returns nothing rather than someone else's remarks (#71).
JOIN cases k ON k.id = c.case_id
WHERE c.case_id = $1 AND k.project_id = $2
GROUP BY c.id
ORDER BY c.created_at;

-- name: CreateComment :one
INSERT INTO comments (case_id, step_id, kind, body, author_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- The anchor is the capture the reviewer was looking at: the one of the
-- edition the case is judged against, for this step and variant. It is what
-- the comment shows for as long as it lives — a step's name is a label, and
-- positions shift (#132). Null when the square had no capture, which is what
-- there was to see.
-- name: AttachCommentVariant :exec
INSERT INTO comment_variants (comment_id, variant_id, capture_id)
SELECT @comment_id, @variant_id, (
    SELECT cap.id FROM captures cap
    JOIN comments c ON c.id = @comment_id
    JOIN cases k ON k.id = c.case_id
    WHERE cap.step_id = c.step_id
      AND cap.variant_id = @variant_id
      AND cap.edition_id = coalesce(
            k.current_edition_id,
            (SELECT e.id FROM editions e WHERE e.project_id = k.project_id
             ORDER BY e.created_at DESC, e.id DESC LIMIT 1))
)
ON CONFLICT DO NOTHING;

-- The verdict of a cell is recomputed, never set by a caller: recording a
-- comment and recomputing what it covers happen together (ADR 0012).
-- name: UpsertCaptureVerdict :exec
INSERT INTO capture_verdicts (case_id, step_id, variant_id, status)
VALUES ($1, $2, $3, $4)
ON CONFLICT (case_id, step_id, variant_id)
DO UPDATE SET status = EXCLUDED.status, updated_at = now();

-- Taking a validation back deletes the row rather than writing a state: the
-- recompute below re-derives the cell from what remains, and the journal is
-- what remembers both moves (#156).
-- name: DeleteCaptureVerdict :exec
DELETE FROM capture_verdicts
WHERE case_id = $1 AND step_id = $2 AND variant_id = $3 AND status = 'validated';

-- name: SetCaseState :exec
UPDATE cases SET state = $2, updated_at = now() WHERE id = $1;

-- A comment, inside the project the caller named. Reached through its case,
-- which is what carries the project: the comment table names no project of its
-- own (#71).
-- name: GetComment :one
SELECT c.* FROM comments c
JOIN cases k ON k.id = c.case_id
JOIN projects p ON p.id = k.project_id
WHERE c.id = $1 AND p.slug = $2;

-- The state is written by the server as a consequence of a recorded move,
-- never received as an argument (ADR 0002).
-- name: SetCommentState :exec
UPDATE comments SET state = $2, updated_at = now() WHERE id = $1;

-- One row per attached issue; attaching twice adds a second (#138).
-- name: CreateCommentIssue :one
INSERT INTO comment_issues (comment_id, issue_id, url, title)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CommentIssueStates :many
SELECT state FROM comment_issues WHERE comment_id = $1;

-- Scoped through the comment and its project, like the comment itself (#71).
-- name: GetCommentIssue :many
SELECT ci.* FROM comment_issues ci
JOIN comments c ON c.id = ci.comment_id
JOIN cases k ON k.id = c.case_id
JOIN projects p ON p.id = k.project_id
WHERE ci.comment_id = $1 AND p.slug = $2
  AND ($3::text = '' OR ci.id = $3)
ORDER BY ci.created_at
LIMIT 2;

-- name: SetCommentIssueState :exec
UPDATE comment_issues SET state = $2 WHERE id = $1;

-- The refs of every comment of one case, with each ref's last refusal remark:
-- what the dev has to read is the remark, and the table shows it under the
-- title (#138).
-- name: CaseCommentIssues :many
SELECT ci.*, (
    SELECT j.remark FROM comment_judgments j
    WHERE j.comment_issue_id = ci.id AND j.verdict = 'refused'
    ORDER BY j.created_at DESC LIMIT 1
) AS last_refusal
FROM comment_issues ci
JOIN comments c ON c.id = ci.comment_id
WHERE c.case_id = $1
ORDER BY ci.comment_id, ci.created_at;

-- name: DiscardComment :exec
UPDATE comments SET state = $2, discard_reason = $3, updated_at = now() WHERE id = $1;

-- Every judgment is kept: three round trips on one comment is information
-- (ADR 0012).
-- name: RecordJudgment :exec
INSERT INTO comment_judgments (comment_id, comment_issue_id, verdict, remark, actor_id)
VALUES ($1, $2, $3, $4, $5);

-- name: CommentJudgments :many
SELECT * FROM comment_judgments WHERE comment_id = $1 ORDER BY created_at;

-- The bytes a reviewer approved, taken from the edition they were judging.
--
-- Nothing is stamped when that edition holds no capture for the square: a
-- validated hole approves nothing. The environment comes from the capture's own
-- provenance, so a reference never crosses environments (ADR 0004, ADR 0017).
-- name: StampCaptureReference :exec
INSERT INTO capture_references (case_id, step_id, variant_id, environment_id, blob_hash, approved_by)
SELECT
    @case_id, c.step_id, c.variant_id,
    coalesce(c.provenance->>'environmentId', ''),
    c.blob_hash, @approved_by
FROM captures c
WHERE c.edition_id = @edition_id
  AND c.step_id = @step_id
  AND c.variant_id = @variant_id
ON CONFLICT (case_id, step_id, variant_id, environment_id) DO UPDATE
SET blob_hash   = EXCLUDED.blob_hash,
    approved_by = EXCLUDED.approved_by,
    approved_at = now();

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

-- What a case is judged against right now, and who wrote the reference.
-- name: CaseReferences :many
SELECT step_id, variant_id, environment_id, blob_hash, approved_by, approved_at
FROM capture_references
WHERE case_id = $1
ORDER BY step_id, variant_id, environment_id;
