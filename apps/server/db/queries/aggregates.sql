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
