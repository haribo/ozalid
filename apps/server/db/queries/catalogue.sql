-- name: CreateProject :one
INSERT INTO projects (slug, name, intake_policy)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetProjectBySlug :one
SELECT * FROM projects WHERE slug = $1;

-- name: CreateCategory :one
INSERT INTO categories (project_id, parent_id, name, position)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListCategories :many
SELECT * FROM categories
WHERE project_id = $1
ORDER BY parent_id NULLS FIRST, position, name;

-- Refused unless empty: deleting a filing drawer must not move what is inside
-- it (ADR 0014).
--
-- An archived case still points at its category, so it counts. Otherwise
-- deleting the category would leave it pointing at nothing, and an archived
-- case is meant to stay whole -- captures, comments, journal and where it was
-- filed.
-- name: CountCategoryContents :one
SELECT
    (SELECT count(*) FROM categories WHERE parent_id = $1) AS subcategories,
    (SELECT count(*) FROM cases WHERE category_id = $1) AS cases,
    (SELECT count(*) FROM cases WHERE category_id = $1 AND archived_at IS NOT NULL) AS archived_cases;

-- name: DeleteEmptyCategory :execrows
DELETE FROM categories c
WHERE c.id = $1
  AND c.project_id = (SELECT p.id FROM projects p WHERE p.slug = $2)
  AND NOT EXISTS (SELECT 1 FROM categories s WHERE s.parent_id = c.id)
  AND NOT EXISTS (SELECT 1 FROM cases k WHERE k.category_id = c.id);
