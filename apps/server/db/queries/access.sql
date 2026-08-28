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
-- name: DeactivateUser :exec
UPDATE users SET deactivated_at = now() WHERE id = $1;

-- name: CreateServiceToken :one
INSERT INTO service_tokens (service_account_id, label, token_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- The account a token opens, looked up by the hash kept for it.
--
-- Nothing here compares the token itself: the hash is the key, so a lookup
-- never carries a secret into a query plan or a slow-query log.
-- name: ServiceAccountByTokenHash :one
SELECT s.id, s.name, t.id AS token_id
FROM service_tokens t
JOIN service_accounts s ON s.id = t.service_account_id
WHERE t.token_hash = $1;

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
