-- Axes are declared by the project, and created on the fly the first time a
-- capture mentions one: ozalid ships no built-in list (ADR 0001).
-- name: UpsertAxis :one
INSERT INTO axes (project_id, name, position)
VALUES ($1, $2, $3)
ON CONFLICT (project_id, name) DO UPDATE SET name = EXCLUDED.name
RETURNING *;

-- A variant is a combination of axis values, held canonically so an axis the
-- client did not supply is absent rather than null.
-- name: UpsertVariant :one
INSERT INTO variants (project_id, values, label)
VALUES ($1, $2, $3)
ON CONFLICT (project_id, values) DO UPDATE SET label = EXCLUDED.label
RETURNING *;

-- name: CreateEdition :one
INSERT INTO editions (project_id, revision)
VALUES ($1, $2)
RETURNING *;

-- Steps are reconciled per case: the manifest gives the order, and re-pushing
-- the same step keeps its identity so the comments anchored to it survive.
-- name: UpsertStep :one
INSERT INTO steps (case_id, name, position)
VALUES ($1, $2, $3)
ON CONFLICT (case_id, position) DO UPDATE SET name = EXCLUDED.name
RETURNING *;

-- name: DeleteStepsBeyond :exec
DELETE FROM steps WHERE case_id = $1 AND position >= $2;

-- name: UpsertBlob :exec
INSERT INTO blobs (hash, size_bytes)
VALUES ($1, $2)
ON CONFLICT (hash) DO NOTHING;

-- name: BlobExists :one
SELECT EXISTS (SELECT 1 FROM blobs WHERE hash = $1);

-- name: CreateCapture :one
INSERT INTO captures (edition_id, step_id, variant_id, blob_hash, provenance)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CreateRecording :one
INSERT INTO recordings (edition_id, case_id, variant_id, blob_hash)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CasesByIDs :many
SELECT * FROM cases WHERE project_id = $1 AND id = ANY($2::text[]);

-- name: CountCasesToReview :one
SELECT count(*) FROM cases
WHERE project_id = $1 AND archived_at IS NULL AND state = 'to-review';

-- A case that has just received its first captures leaves the funnel's edge:
-- it is no longer uninstrumented, and nobody has judged it yet (ADR 0012).
--
-- Only that transition is made here. Every other one is driven by a comment,
-- and comments are not part of intake.
-- name: EnterReviewOnFirstCaptures :many
UPDATE cases k
SET state = 'to-review', updated_at = now()
WHERE k.id = ANY($1::text[])
  AND k.state = 'not-instrumented'
  AND EXISTS (
      SELECT 1 FROM steps s
      JOIN captures c ON c.step_id = s.id
      WHERE s.case_id = k.id
  )
RETURNING k.id;

-- Every transition is journalled: without the trace, a stored state cannot
-- serve as a regression oracle (ADR 0002).
-- name: RecordTransition :exec
INSERT INTO journal (project_id, case_id, from_state, to_state, cause, actor_id, actor_kind, inputs, rule_version)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);
