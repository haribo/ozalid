-- The id is generated here and returned; the client never invents it
-- (ADR 0014).
-- name: CreateCase :one
INSERT INTO cases (project_id, category_id, title, description)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetCase :one
SELECT * FROM cases WHERE id = $1;

-- Filters on the stored state, no scan: that is why the state is stored at all
-- (ADR 0002).
-- name: ListCases :many
SELECT * FROM cases
WHERE project_id = $1
  AND archived_at IS NULL
  AND (sqlc.narg('state')::text IS NULL OR state = sqlc.narg('state')::text)
  AND (sqlc.narg('category_id')::text IS NULL OR category_id = sqlc.narg('category_id')::text)
ORDER BY title;

-- name: UpdateCaseDetails :one
UPDATE cases
SET title = $2, description = $3, category_id = $4, updated_at = now()
WHERE id = $1
RETURNING *;

-- Archived, never deleted (ADR 0014).
-- name: ArchiveCase :execrows
UPDATE cases SET archived_at = now(), updated_at = now()
WHERE id = $1 AND archived_at IS NULL;

-- name: CreateStep :one
INSERT INTO steps (case_id, name, position)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListSteps :many
SELECT * FROM steps WHERE case_id = $1 ORDER BY position;
