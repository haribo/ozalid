-- name: CreateUser :one
INSERT INTO users (name, email, is_admin)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateServiceAccount :one
INSERT INTO service_accounts (name, owner_id)
VALUES ($1, $2)
RETURNING *;

-- A service account's project is written here and nowhere else, so the two can
-- never disagree; the partial unique index keeps it to one (ADR 0018).
-- name: AddProjectMember :exec
INSERT INTO project_members (project_id, user_id, service_account_id, rights)
VALUES (@project_id, @user_id, @service_account_id, @rights);

-- What a person may do on one project: whether they administer the instance,
-- and what their membership carries. A deactivated account resolves to nothing.
-- name: StandingOfUser :one
SELECT u.is_admin, coalesce(m.rights, '')::text AS rights
FROM users u
LEFT JOIN project_members m
       ON m.user_id = u.id AND m.project_id = sqlc.narg('project_id')
WHERE u.id = $1 AND u.deactivated_at IS NULL;

-- What a program may do on one project.
--
-- A service account never administers: administration reaches accounts, and a
-- program that could make accounts could make itself another (product.md §8.2).
-- name: StandingOfServiceAccount :one
SELECT false AS is_admin, coalesce(m.rights, '')::text AS rights
FROM service_accounts s
LEFT JOIN project_members m
       ON m.service_account_id = s.id AND m.project_id = sqlc.narg('project_id')
WHERE s.id = $1;

-- An account is deactivated, never deleted: what it reviewed has to stay
-- readable, and the journal names it (ADR 0018).
-- name: DeactivateUser :exec
UPDATE users SET deactivated_at = now() WHERE id = $1;
