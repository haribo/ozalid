-- +goose Up
-- +goose StatementBegin

-- A person.
--
-- No password: a person signs in through a link sent to this address, so there
-- is nothing to forget and no recovery procedure to write (ADR 0019). The
-- address is stored from the first day even before mail can be sent — it is
-- what makes the sign-in possible later without touching the data.
CREATE TABLE users (
    id       text PRIMARY KEY DEFAULT short_id(),
    name     text NOT NULL,
    email    text NOT NULL,
    -- Administration is a single flag, not a hierarchy. It reaches accounts and
    -- the creation of projects, never the content of one (product.md §8.2).
    is_admin boolean NOT NULL DEFAULT false,
    -- Accounts are deactivated, never deleted: what they reviewed has to stay
    -- readable, and the journal names them (ADR 0018).
    deactivated_at timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now()
);

-- Addresses differing only in case are the same address to every mail server,
-- so they must be the same address here.
CREATE UNIQUE INDEX users_email_key ON users (lower(email));

-- A program: the intake client, an automation agent, an assistant pushing
-- captures.
--
-- Owned by a person, and the ownership is required rather than advisory: a
-- machine account nobody owns is a machine account nobody revokes. Deleting the
-- owner is refused rather than cascading, which forces a reassignment instead
-- of an orphan.
CREATE TABLE service_accounts (
    id         text PRIMARY KEY DEFAULT short_id(),
    name       text NOT NULL,
    owner_id   text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Who reaches a project, and what they may do there.
--
-- Rights belong to membership, never to the kind of account holding it: an
-- agent pushing captures and a person pushing captures do the same thing
-- (ADR 0019). One row therefore names exactly one of the two.
--
-- A service account has no project column of its own. Its membership is where
-- its project is written, so the fact lives in one place; the partial index
-- below is what keeps ADR 0018's "a token belongs to one project" true by
-- construction rather than by discipline.
CREATE TABLE project_members (
    project_id         text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id            text REFERENCES users(id) ON DELETE CASCADE,
    service_account_id text REFERENCES service_accounts(id) ON DELETE CASCADE,
    rights             text NOT NULL CHECK (rights IN ('reader', 'member')),
    added_at           timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT project_members_one_holder
        CHECK ((user_id IS NOT NULL) <> (service_account_id IS NOT NULL))
);

CREATE UNIQUE INDEX project_members_user_key
    ON project_members (project_id, user_id) WHERE user_id IS NOT NULL;

-- One project per service account, enforced rather than trusted.
CREATE UNIQUE INDEX project_members_service_account_key
    ON project_members (service_account_id) WHERE service_account_id IS NOT NULL;

CREATE INDEX project_members_project_idx ON project_members (project_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE project_members;
DROP TABLE service_accounts;
DROP TABLE users;

-- +goose StatementEnd
