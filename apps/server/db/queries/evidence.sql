-- The edition a case is read against defaults to the project's most recent.
-- name: LatestEdition :one
SELECT * FROM editions
WHERE project_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: EditionByID :one
SELECT * FROM editions WHERE id = $1 AND project_id = $2;

-- Every capture of one case at one edition, joined with its step and variant.
-- One query rather than a walk over steps: a case with thirty steps in eight
-- variants would otherwise be two hundred and forty round trips.
-- name: CaseEvidence :many
SELECT
    s.id       AS step_id,
    s.name     AS step_name,
    s.position AS step_position,
    v.id       AS variant_id,
    v.label    AS variant_label,
    v.values   AS variant_values,
    c.blob_hash,
    c.provenance,
    -- A square with no verdict row has not been judged yet: the reviewer holds
    -- the ball on it (ADR 0012).
    coalesce(cv.status, 'to-review') AS status
FROM steps s
LEFT JOIN captures c ON c.step_id = s.id AND c.edition_id = $2
LEFT JOIN variants v ON v.id = c.variant_id
LEFT JOIN capture_verdicts cv
       ON cv.case_id = s.case_id AND cv.step_id = s.id AND cv.variant_id = c.variant_id
WHERE s.case_id = $1
ORDER BY s.position, v.label;

-- name: CaseRecordings :many
SELECT r.blob_hash, v.id AS variant_id, v.label AS variant_label, v.values AS variant_values
FROM recordings r
JOIN variants v ON v.id = r.variant_id
WHERE r.case_id = $1 AND r.edition_id = $2
ORDER BY v.label;
