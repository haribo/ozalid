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

-- Granting again changes the rights rather than failing: an administrator who
-- meant to demote somebody said so, and refusing would make them revoke first.
-- name: GrantMembership :execrows
INSERT INTO project_members (project_id, user_id, rights)
SELECT p.id, u.id, @rights
FROM projects p, users u
WHERE p.slug = @slug AND u.id = @user_id
ON CONFLICT (project_id, user_id) WHERE user_id IS NOT NULL
DO UPDATE SET rights = excluded.rights;

-- name: RevokeMembership :execrows
DELETE FROM project_members m
USING projects p
WHERE m.project_id = p.id AND p.slug = @slug AND m.user_id = @user_id;

-- Both kinds of holder in one list: a project's members are the people and the
-- programs that reach it, and hiding either would make the list a lie.
-- name: ListProjectMembers :many
SELECT
    coalesce(u.id, sa.id)     AS account_id,
    coalesce(u.name, sa.name) AS name,
    u.email,
    (m.user_id IS NOT NULL)::boolean AS is_person,
    m.rights,
    m.added_at
FROM project_members m
JOIN projects p ON p.id = m.project_id
LEFT JOIN users u ON u.id = m.user_id
LEFT JOIN service_accounts sa ON sa.id = m.service_account_id
WHERE p.slug = @slug
ORDER BY is_person DESC, lower(coalesce(u.name, sa.name));

-- What a person may do on one project: whether they administer the instance,
-- and what their membership carries. A deactivated account resolves to nothing.
-- The same two questions, asked by the slug a caller names rather than by an id
-- it has no way to know. A slug nobody has resolves to nothing, which the rule
-- then refuses — the same answer as an unknown project by a shorter road.
-- name: StandingOfUserOnSlug :one
SELECT u.is_admin, coalesce(m.rights, '')::text AS rights
FROM users u
LEFT JOIN projects p ON p.slug = sqlc.narg('slug')
LEFT JOIN project_members m ON m.user_id = u.id AND m.project_id = p.id
WHERE u.id = $1 AND u.deactivated_at IS NULL;

-- name: StandingOfServiceAccountOnSlug :one
SELECT false AS is_admin, coalesce(m.rights, '')::text AS rights
FROM service_accounts s
LEFT JOIN projects p ON p.slug = sqlc.narg('slug')
LEFT JOIN project_members m ON m.service_account_id = s.id AND m.project_id = p.id
WHERE s.id = $1;

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
-- Idempotent: `deactivated_at` is set once and never moved, so deactivating
-- twice is not two different days.
-- name: DeactivateUser :execrows
UPDATE users SET deactivated_at = coalesce(deactivated_at, now()) WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY lower(name), id;

-- name: CreateServiceToken :one
INSERT INTO service_tokens (service_account_id, label, token_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- The account a token opens, looked up by the hash kept for it.
--
-- Nothing here compares the token itself: the hash is the key, so a lookup
-- never carries a secret into a query plan or a slow-query log.
-- A deactivated service account resolves to nothing, exactly as a deactivated
-- person does: the token still exists, and stops opening anything.
-- name: ServiceAccountByTokenHash :one
SELECT s.id, s.name, t.id AS token_id
FROM service_tokens t
JOIN service_accounts s ON s.id = t.service_account_id
WHERE t.token_hash = $1 AND s.deactivated_at IS NULL;

-- Read on every accepted call, so an operator can tell which tokens are still
-- in use and which can go.
-- name: TouchServiceToken :exec
UPDATE service_tokens SET last_used_at = now() WHERE id = $1;

-- name: UserByEmail :one
SELECT * FROM users WHERE lower(email) = lower($1) AND deactivated_at IS NULL;

-- The expiry is computed by the database, against the clock that will later
-- decide whether the link is still good. Setting it from the application would
-- compare one clock against another, and clock skew would decide.
-- name: CreateSignInLink :exec
INSERT INTO sign_in_links (user_id, link_hash, expires_at)
VALUES (@user_id, @link_hash, now() + make_interval(secs => @lifetime_seconds::int));

-- A link is found by its hash, and only if it is still good. Expired and
-- already used are both refused here rather than reported to the caller: what
-- reaches the browser is one answer, "this link no longer works".
-- name: ClaimSignInLink :one
UPDATE sign_in_links
SET used_at = now()
WHERE link_hash = $1 AND used_at IS NULL AND expires_at > now()
RETURNING user_id;

-- name: CreateSession :exec
INSERT INTO sessions (user_id, token_hash, expires_at)
VALUES (@user_id, @token_hash, now() + make_interval(secs => @lifetime_seconds::int));

-- The account a session belongs to, provided both are still good. A deactivated
-- account resolves to nothing, so shutting an account shuts its sessions in the
-- same instant.
-- name: UserBySessionToken :one
SELECT u.id, s.id AS session_id
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1 AND s.expires_at > now() AND u.deactivated_at IS NULL;

-- name: TouchSession :exec
UPDATE sessions SET last_seen_at = now() WHERE id = $1;

-- name: EndSession :exec
DELETE FROM sessions WHERE token_hash = $1;

-- name: UserByID :one
SELECT * FROM users WHERE id = $1 AND deactivated_at IS NULL;

-- name: CreateServiceAccountInProject :one
INSERT INTO service_accounts (name, owner_id)
SELECT @name, @owner_id
WHERE EXISTS (SELECT 1 FROM projects WHERE slug = @slug)
RETURNING *;

-- name: ServiceAccountInProject :one
SELECT sa.* FROM service_accounts sa
JOIN project_members m ON m.service_account_id = sa.id
JOIN projects p ON p.id = m.project_id
WHERE sa.id = @id AND p.slug = @slug AND sa.deactivated_at IS NULL;

-- Idempotent, like deactivating a person: the day it happened does not move.
-- name: DeactivateServiceAccount :execrows
UPDATE service_accounts sa
SET deactivated_at = coalesce(sa.deactivated_at, now())
FROM project_members m, projects p
WHERE sa.id = @id AND m.service_account_id = sa.id
  AND p.id = m.project_id AND p.slug = @slug;

-- What a token is for and whether anything still uses it. Never the token: only
-- its hash was ever stored, and a hash is not a credential anyone can present.
-- name: ListServiceTokens :many
SELECT t.id, t.label, t.created_at, t.last_used_at
FROM service_tokens t
JOIN service_accounts sa ON sa.id = t.service_account_id
JOIN project_members m ON m.service_account_id = sa.id
JOIN projects p ON p.id = m.project_id
WHERE t.service_account_id = @service_account_id AND p.slug = @slug
ORDER BY t.created_at DESC;

-- name: DeleteServiceToken :execrows
DELETE FROM service_tokens t
USING service_accounts sa, project_members m, projects p
WHERE t.id = @id AND t.service_account_id = @service_account_id
  AND sa.id = t.service_account_id
  AND m.service_account_id = sa.id AND p.id = m.project_id AND p.slug = @slug;
