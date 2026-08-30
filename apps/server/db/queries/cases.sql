-- The id is generated here and returned; the client never invents it
-- (ADR 0014).
-- name: CreateCase :one
INSERT INTO cases (project_id, category_id, title, description)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- A case, inside the project the caller named. The project is not checked
-- beside the query, it *is* the query: a case from another project returns no
-- row, so there is no consistency check to write, and none to forget (#71).
-- name: CaseInProject :one
SELECT c.* FROM cases c
JOIN projects p ON p.id = c.project_id
WHERE c.id = $1 AND p.slug = $2;

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
WHERE cases.id = $1
  AND cases.project_id = (SELECT p.id FROM projects p WHERE p.slug = $5)
RETURNING *;

-- Archived, never deleted (ADR 0014).
-- name: ArchiveCase :execrows
UPDATE cases SET archived_at = now(), updated_at = now()
WHERE cases.id = $1
  AND cases.project_id = (SELECT p.id FROM projects p WHERE p.slug = $2)
  AND cases.archived_at IS NULL;

-- name: CreateStep :one
INSERT INTO steps (case_id, name, position)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListSteps :many
SELECT * FROM steps WHERE case_id = $1 ORDER BY position;
