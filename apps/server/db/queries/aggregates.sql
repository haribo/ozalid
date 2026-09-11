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
    count(k.id) FILTER (WHERE k.state = 'refused')                 AS refused,
    count(k.id) FILTER (WHERE k.state = 'accepted')                AS accepted,
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
    max(e.created_at)::timestamptz                                     AS last_edition
FROM cases k
LEFT JOIN steps s ON s.case_id = k.id
LEFT JOIN captures c ON c.step_id = s.id AND c.edition_id = (SELECT id FROM latest)
LEFT JOIN editions e ON e.id = c.edition_id
WHERE k.project_id = $1
  AND k.archived_at IS NULL
  AND (sqlc.narg('category_id')::text IS NULL OR k.category_id = sqlc.narg('category_id')::text)
GROUP BY k.id
ORDER BY k.title;

-- Everything the state computation reads, for one case at one edition.
-- name: CaseCaptures :many
SELECT s.id AS step_id, c.variant_id
FROM steps s
JOIN captures c ON c.step_id = s.id AND c.edition_id = $2
WHERE s.case_id = $1;

-- name: CaseAcceptedCaptures :many
SELECT step_id, variant_id FROM capture_acceptances
WHERE case_id = $1;

-- name: CaseComments :many
SELECT c.id, c.step_id, c.body, c.state,
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
INSERT INTO comments (case_id, step_id, body, author_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- The anchor is the capture the reviewer was looking at: the one of the
-- edition the case is judged against, for this step and variant. It is what
-- the comment shows for as long as it lives — a step's name is a label, and
-- positions shift (#132). Null when the step and variant had no capture, which is what
-- there was to see.
-- The edition rides in from the caller's resolver (ADR 0024): the pin is
-- derived, never stored, so no query reads it off the case.
-- name: AttachCommentVariant :exec
INSERT INTO comment_variants (comment_id, variant_id, capture_id)
SELECT @comment_id, @variant_id, (
    SELECT cap.id FROM captures cap
    JOIN comments c ON c.id = @comment_id
    WHERE cap.step_id = c.step_id
      AND cap.variant_id = @variant_id
      AND cap.edition_id = @edition_id::text
)
ON CONFLICT DO NOTHING;

-- The acceptance is the reviewer's own fact — who, and when (ADR 0021).
-- Recording it is the write; every status is derived at read time.
-- The old note stands for the case state: recorded, never set by a caller: recording a
-- comment and recomputing what it covers happen together (ADR 0012).
-- name: RecordCaptureAcceptance :exec
INSERT INTO capture_acceptances (case_id, step_id, variant_id, accepted_by)
VALUES ($1, $2, $3, $4)
ON CONFLICT (case_id, step_id, variant_id) DO NOTHING;

-- Taking an acceptance back deletes the row rather than writing a state: the
-- recompute below re-derives the capture from what remains, and the journal is
-- what remembers both moves (#156).
-- name: DeleteCaptureAcceptance :exec
DELETE FROM capture_acceptances
WHERE case_id = $1 AND step_id = $2 AND variant_id = $3;

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

-- The refs of every comment of one case, with each ref's last standing
-- refusal remark: what the dev has to read is the remark (#138). A refusal
-- speaks only while it stands (#212) — taken back or answered by a
-- redelivery, it leaves the read model.
-- name: CaseCommentIssues :many
SELECT ci.*, (
    SELECT j.remark FROM comment_judgments j
    WHERE j.comment_issue_id = ci.id AND j.verdict = 'refused'
      AND ci.state = 'refused'
      AND (ci.delivered_at IS NULL OR j.created_at > ci.delivered_at)
      AND NOT EXISTS (SELECT 1 FROM comment_judgments t
                      WHERE t.comment_issue_id = ci.id
                        AND t.verdict = 'taken-back'
                        AND t.created_at > j.created_at)
    ORDER BY j.created_at DESC LIMIT 1
) AS last_refusal
FROM comment_issues ci
JOIN comments c ON c.id = ci.comment_id
WHERE c.case_id = $1
ORDER BY ci.comment_id, ci.created_at;

-- Every standing refusal of one case's refs, each naming the capture it was
-- given on (#212): the recap anchors the remark in its variant's column.
-- name: CaseStandingRefusals :many
SELECT ci.id AS ref_id, j.variant_id, j.remark
FROM comment_issues ci
JOIN comments c ON c.id = ci.comment_id
JOIN comment_judgments j ON j.comment_issue_id = ci.id
WHERE c.case_id = $1 AND ci.state = 'refused' AND j.verdict = 'refused'
  AND (ci.delivered_at IS NULL OR j.created_at > ci.delivered_at)
  AND NOT EXISTS (SELECT 1 FROM comment_judgments t
                  WHERE t.comment_issue_id = ci.id
                    AND t.verdict = 'taken-back'
                    AND t.created_at > j.created_at)
ORDER BY ci.id, j.created_at;

-- name: StampCommentIssueDelivery :exec
UPDATE comment_issues SET delivered_at = now() WHERE id = $1;

-- name: DiscardComment :exec
UPDATE comments SET state = $2, discard_reason = $3, updated_at = now() WHERE id = $1;

-- Every judgment is kept: three round trips on one comment is information
-- (ADR 0012).
-- name: RecordJudgment :exec
INSERT INTO comment_judgments (comment_id, comment_issue_id, verdict, remark, actor_id, variant_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: CommentJudgments :many
SELECT * FROM comment_judgments WHERE comment_id = $1 ORDER BY created_at;

-- The bytes a reviewer approved, taken from the edition they were judging.
--
-- Nothing is stamped when that edition holds no capture for the step and variant: a
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

-- What a case is judged against right now, and who wrote the reference.
-- name: CaseReferences :many
SELECT step_id, variant_id, environment_id, blob_hash, approved_by, approved_at
FROM capture_references
WHERE case_id = $1
ORDER BY step_id, variant_id, environment_id;

-- The accepted refs whose settling made one capture read accepted: the ones
-- an unvalidate on that capture must take back (#167). A discarded comment
-- keeps its refs untouched — discarding was said with a reason and it stands.
-- Accepting released the variant from the coverage (ADR 0022), so the comment
-- is found through the acceptance that names the variant, not the coverage.
-- name: SettledRefsOnCapture :many
SELECT ci.id, ci.comment_id, ci.state, c.state AS comment_state
FROM comment_issues ci
JOIN comments c ON c.id = ci.comment_id
WHERE c.case_id = @case_id AND c.step_id = @step_id
  AND c.state = 'accepted' AND ci.state = 'accepted'
  AND EXISTS (SELECT 1 FROM comment_judgments cj
              WHERE cj.comment_id = c.id AND cj.variant_id = @variant_id::text
                AND cj.verdict = 'accepted');

-- The reviewer's own drafts on one capture: remarks with no issue attached yet.
-- Withdrawing a draft refusal takes them with it — ADR 0020's explicit
-- exception to "nothing is deleted", scoped to the author's own drafts.
-- name: DraftCommentsOnCapture :many
SELECT DISTINCT c.id FROM comments c
JOIN comment_variants cv ON cv.comment_id = c.id
WHERE c.case_id = $1 AND c.step_id = $2 AND cv.variant_id = $3
  AND c.author_id = $4 AND c.state = 'to-track'
  AND NOT EXISTS (SELECT 1 FROM comment_issues ci WHERE ci.comment_id = c.id);

-- name: DeleteComment :exec
DELETE FROM comments WHERE id = $1;

-- Editing is the author reworking their own draft (ADR 0020): body only —
-- variants are replaced beside it in the same transaction.
-- name: UpdateCommentBody :exec
UPDATE comments SET body = $2, updated_at = now() WHERE id = $1;

-- name: DetachCommentVariants :exec
DELETE FROM comment_variants WHERE comment_id = $1;

-- The settled ref-less remarks whose acceptance made one capture read
-- accepted: unaccepting that capture takes their judgment back too (#167,
-- #175) — the rule is "whatever made it accepted", refs and remarks alike.
-- name: SettledRemarksOnCapture :many
SELECT c.id, c.state FROM comments c
WHERE c.case_id = @case_id AND c.step_id = @step_id
  AND c.state = 'accepted'
  AND NOT EXISTS (SELECT 1 FROM comment_issues ci WHERE ci.comment_id = c.id)
  AND EXISTS (SELECT 1 FROM comment_judgments cj
              WHERE cj.comment_id = c.id AND cj.variant_id = @variant_id::text
                AND cj.verdict = 'accepted');

-- Everything the derivation reads about one case's captures (ADR 0021): the
-- bytes on display, the reference approved in this capture's own environment
-- (ADR 0017), and the measurement intake recorded.
-- name: CaseCaptureFacts :many
SELECT s.id AS step_id, c.variant_id, c.blob_hash, c.moved_pixels,
       coalesce(r.blob_hash, '') AS reference_hash
FROM steps s
JOIN captures c ON c.step_id = s.id AND c.edition_id = $2
LEFT JOIN LATERAL (
    SELECT ref.blob_hash FROM capture_references ref
    WHERE ref.case_id = s.case_id AND ref.step_id = s.id AND ref.variant_id = c.variant_id
      -- A push without provenance lives in the empty environment: intake
      -- keys its comparison on '' and the read must too — a NULL here
      -- matched nothing and left measured captures blind (#230).
      AND ref.environment_id = coalesce(c.provenance->>'environmentId', '')
    ORDER BY ref.approved_at DESC LIMIT 1
) r ON true
WHERE s.case_id = $1;

-- The captures each comment covers, with the bytes the remark was written on
-- (#132): what decides whether a refusal still talks about the image shown.
-- name: CaseCommentAnchors :many
SELECT cv.comment_id, c.step_id, cv.variant_id, a.blob_hash AS anchor_hash
FROM comment_variants cv
JOIN comments c ON c.id = cv.comment_id
LEFT JOIN captures a ON a.id = cv.capture_id
WHERE c.case_id = $1;

-- name: CommentCoveredVariants :many
SELECT variant_id FROM comment_variants WHERE comment_id = $1;

-- Releasing a variant from a remark's coverage (ADR 0022): the acceptance of
-- a fix on one capture takes that variant out of the claim.
-- name: ReleaseCommentVariant :execrows
DELETE FROM comment_variants WHERE comment_id = $1 AND variant_id = $2;

-- Restoring it on a take-back, anchored to the capture on display at the
-- case's pinned edition — the bytes the judgment was about.
-- name: RestoreCommentVariant :exec
INSERT INTO comment_variants (comment_id, variant_id, capture_id)
SELECT @comment_id, @variant_id, (
    SELECT c.id FROM captures c
    JOIN comments k ON k.id = @comment_id
    WHERE c.step_id = k.step_id AND c.variant_id = @variant_id::text
      AND c.edition_id = @edition_id
    LIMIT 1
)
ON CONFLICT DO NOTHING;

-- The refusal's anchor follows the refused variant: the judge refused these
-- bytes (ADR 0022).
-- name: ReanchorCommentVariant :exec
UPDATE comment_variants cv
SET capture_id = (
    SELECT c.id FROM captures c
    JOIN comments k ON k.id = cv.comment_id
    WHERE c.step_id = k.step_id AND c.variant_id = cv.variant_id
      AND c.edition_id = @edition_id
    LIMIT 1
)
WHERE cv.comment_id = @comment_id AND cv.variant_id = @variant_id;

-- Claiming is also renewing (ADR 0005, #95): one atomic statement takes a
-- free or expired lock, or beats the caller's own. Somebody else's live lock
-- makes the upsert a no-op — no row comes back, and the caller reads who
-- holds it instead. The window rides in as seconds so expiry is read, never
-- written.
-- name: ClaimCaseLock :one
INSERT INTO case_locks (case_id, account_id, edition_id)
VALUES (@case_id, @account_id, @edition_id)
ON CONFLICT (case_id) DO UPDATE
SET account_id = EXCLUDED.account_id,
    claimed_at = CASE
        WHEN case_locks.account_id = EXCLUDED.account_id
         AND case_locks.beaten_at > now() - make_interval(secs => @window_seconds::int)
        THEN case_locks.claimed_at
        ELSE now()
    END,
    -- The pin follows the claim (ADR 0024): a heartbeat keeps the bytes, a
    -- fresh claim — opening the page, expiry included — re-stamps onto what
    -- is current now. Reloading is leaving and coming back.
    edition_id = CASE
        WHEN NOT @fresh::boolean
         AND case_locks.account_id = EXCLUDED.account_id
         AND case_locks.beaten_at > now() - make_interval(secs => @window_seconds::int)
        THEN case_locks.edition_id
        ELSE EXCLUDED.edition_id
    END,
    beaten_at = now()
WHERE case_locks.account_id = EXCLUDED.account_id
   OR case_locks.beaten_at < now() - make_interval(secs => @window_seconds::int)
RETURNING case_id, account_id, claimed_at;

-- name: ReadCaseLock :one
SELECT l.account_id, l.claimed_at, l.edition_id, u.name AS holder_name
FROM case_locks l
JOIN users u ON u.id = l.account_id
WHERE l.case_id = @case_id
  AND l.beaten_at > now() - make_interval(secs => @window_seconds::int);

-- A delivery advances the case at once (#142, ADR 0024): the live lock is
-- re-stamped onto the latest edition; with no live lock there is nothing to
-- do — the case already reads at the latest.
-- name: RestampLiveLock :exec
UPDATE case_locks
SET edition_id = @edition_id
WHERE case_id = @case_id
  AND beaten_at > now() - make_interval(secs => @window_seconds::int);

-- Releasing somebody else's lock, or one nobody holds, changes nothing.
-- name: ReleaseCaseLock :exec
DELETE FROM case_locks WHERE case_id = $1 AND account_id = $2;
