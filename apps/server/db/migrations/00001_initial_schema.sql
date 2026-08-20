-- +goose Up
-- The whole model decided in phase 0, written once on an empty database.
-- ADR 0012 (states), ADR 0013 (capture vs recording), ADR 0014 (identity and
-- catalogue).

-- A short, non-sequential public id. Sequential integers would expose the
-- catalogue's size and let it be walked (ADR 0014).
-- +goose StatementBegin
CREATE FUNCTION short_id() RETURNS text AS $$
  -- 12 hex characters off a v4 uuid: 48 bits of randomness, which the primary
  -- key would catch a collision on anyway. gen_random_uuid is built in, so
  -- this needs no extension.
  SELECT substr(replace(gen_random_uuid()::text, '-', ''), 1, 12);
$$ LANGUAGE sql VOLATILE;
-- +goose StatementEnd

CREATE TABLE projects (
    id          text PRIMARY KEY DEFAULT short_id(),
    slug        text NOT NULL UNIQUE,
    name        text NOT NULL,
    -- 'strict' refuses intake while any case is to-review; 'per-case' never
    -- refuses (ADR 0007).
    intake_policy text NOT NULL DEFAULT 'per-case'
        CHECK (intake_policy IN ('strict', 'per-case')),
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- The catalogue is a tree of unrestricted depth; a case belongs to exactly one
-- node (ADR 0014).
CREATE TABLE categories (
    id         text PRIMARY KEY DEFAULT short_id(),
    project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    parent_id  text REFERENCES categories(id),
    name       text NOT NULL,
    position   integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    -- Siblings cannot share a name. NULLS NOT DISTINCT so the rule also holds
    -- at the root, where parent_id is null.
    UNIQUE NULLS NOT DISTINCT (project_id, parent_id, name)
);
CREATE INDEX categories_parent_idx ON categories (project_id, parent_id, position);

-- A rendering dimension the project declares. ozalid ships no built-in list
-- (ADR 0001).
CREATE TABLE axes (
    id         text PRIMARY KEY DEFAULT short_id(),
    project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name       text NOT NULL,
    position   integer NOT NULL DEFAULT 0,
    UNIQUE (project_id, name)
);

-- A variant is a combination of axis values, held as a canonical object so an
-- axis the client did not supply is simply absent -- no null to compare.
CREATE TABLE variants (
    id         text PRIMARY KEY DEFAULT short_id(),
    project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    values     jsonb NOT NULL,
    label      text NOT NULL,
    UNIQUE (project_id, values)
);

CREATE TABLE cases (
    id          text PRIMARY KEY DEFAULT short_id(),
    project_id  text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    category_id text REFERENCES categories(id),
    title       text NOT NULL,
    description text,
    -- Computed by the server from the case's comments, never received from a
    -- client (ADR 0002, ADR 0012).
    state       text NOT NULL DEFAULT 'not-instrumented'
        CHECK (state IN ('not-instrumented', 'to-review', 'to-fix', 'reviewed')),
    -- A case is archived, never deleted: deleting takes its captures, its
    -- comments and its journal with it (ADR 0014).
    archived_at timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX cases_catalogue_idx ON cases (project_id, category_id) WHERE archived_at IS NULL;
CREATE INDEX cases_state_idx ON cases (project_id, state) WHERE archived_at IS NULL;

CREATE TABLE steps (
    id         text PRIMARY KEY DEFAULT short_id(),
    case_id    text NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    name       text NOT NULL,
    position   integer NOT NULL,
    UNIQUE (case_id, position)
);

-- One accepted intake, immutable once written.
CREATE TABLE editions (
    id         text PRIMARY KEY DEFAULT short_id(),
    project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    -- Opaque to the server: displayed, never computed on (ADR 0013).
    revision   text,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX editions_project_idx ON editions (project_id, created_at DESC);

-- Bytes live on disk under their hash; this table holds what the database
-- needs to know about them (backend ADR 0004).
CREATE TABLE blobs (
    hash       text PRIMARY KEY,
    size_bytes bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Comparable evidence: hashed, referenced, the source of freshness (ADR 0013).
CREATE TABLE captures (
    id         text PRIMARY KEY DEFAULT short_id(),
    edition_id text NOT NULL REFERENCES editions(id) ON DELETE CASCADE,
    step_id    text NOT NULL REFERENCES steps(id) ON DELETE CASCADE,
    variant_id text NOT NULL REFERENCES variants(id),
    blob_hash  text NOT NULL REFERENCES blobs(hash),
    -- Operating system, browser and version, environment id. Byte comparison
    -- only means something within one environment (ADR 0004).
    provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
    UNIQUE (edition_id, step_id, variant_id)
);
CREATE INDEX captures_step_idx ON captures (step_id, variant_id);

-- A supporting exhibit: optional, never byte-compared, never a source of state
-- (ADR 0013).
CREATE TABLE recordings (
    id         text PRIMARY KEY DEFAULT short_id(),
    edition_id text NOT NULL REFERENCES editions(id) ON DELETE CASCADE,
    case_id    text NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    variant_id text NOT NULL REFERENCES variants(id),
    blob_hash  text NOT NULL REFERENCES blobs(hash),
    UNIQUE (edition_id, case_id, variant_id)
);

-- A reviewer's report. Durable: nothing here is ever deleted (ADR 0006).
CREATE TABLE comments (
    id       text PRIMARY KEY DEFAULT short_id(),
    case_id  text NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    step_id  text NOT NULL REFERENCES steps(id) ON DELETE CASCADE,
    kind     text NOT NULL CHECK (kind IN ('defect', 'improvement')),
    body     text NOT NULL,
    state    text NOT NULL DEFAULT 'to-track'
        CHECK (state IN ('to-track', 'tracked', 'to-review', 'refused', 'validated', 'discarded')),
    -- Supplied by the client, never fetched: ozalid holds no tracker
    -- credential and will never know the issue was closed (ADR 0003).
    issue_ref   text,
    issue_url   text,
    issue_title text,
    -- Mandatory on discard, and kept forever with its author.
    discard_reason text,
    author_id  text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (state <> 'discarded' OR discard_reason IS NOT NULL),
    CHECK (state NOT IN ('tracked', 'to-review', 'refused', 'validated') OR issue_ref IS NOT NULL)
);
CREATE INDEX comments_case_idx ON comments (case_id, state);

-- One defect spanning four variants is one comment with four variants checked,
-- never four comments (ADR 0006).
CREATE TABLE comment_variants (
    comment_id text NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    variant_id text NOT NULL REFERENCES variants(id),
    PRIMARY KEY (comment_id, variant_id)
);

-- Every refusal is kept: three round trips on one comment is information
-- (ADR 0012).
CREATE TABLE comment_judgments (
    id         text PRIMARY KEY DEFAULT short_id(),
    comment_id text NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    verdict    text NOT NULL CHECK (verdict IN ('accepted', 'refused')),
    remark     text,
    actor_id   text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (verdict <> 'refused' OR remark IS NOT NULL)
);
CREATE INDEX comment_judgments_comment_idx ON comment_judgments (comment_id, created_at DESC);

-- The bytes a capture was last approved against, per capture cell -- never per
-- case, or a changed cell could not be marked on its own.
CREATE TABLE capture_references (
    case_id    text NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    step_id    text NOT NULL REFERENCES steps(id) ON DELETE CASCADE,
    variant_id text NOT NULL REFERENCES variants(id),
    blob_hash  text NOT NULL REFERENCES blobs(hash),
    -- Stamped only on what the reviewer actually validated in the session, so
    -- "who approved this, and when" stays honest.
    approved_by text NOT NULL,
    approved_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (case_id, step_id, variant_id)
);

-- Recomputed by the server whenever a comment covering the cell changes; no
-- endpoint sets it (ADR 0012).
CREATE TABLE capture_verdicts (
    case_id    text NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    step_id    text NOT NULL REFERENCES steps(id) ON DELETE CASCADE,
    variant_id text NOT NULL REFERENCES variants(id),
    status     text NOT NULL DEFAULT 'to-review'
        CHECK (status IN ('to-review', 'to-fix', 'validated')),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (case_id, step_id, variant_id)
);

-- The audit trail. Append-only, queryable, exportable -- not a log file
-- (ADR 0002).
CREATE TABLE journal (
    id         bigserial PRIMARY KEY,
    project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    case_id    text REFERENCES cases(id) ON DELETE CASCADE,
    from_state text,
    to_state   text,
    cause      text NOT NULL,
    actor_id   text NOT NULL,
    actor_kind text NOT NULL CHECK (actor_kind IN ('human', 'machine')),
    -- A fingerprint of the facts the computation consumed. Without it a stored
    -- state cannot serve as a regression oracle (ADR 0002).
    inputs     jsonb NOT NULL DEFAULT '{}'::jsonb,
    -- Which version of the computation produced this, so a replay knows what
    -- it is comparing against.
    rule_version integer NOT NULL DEFAULT 1,
    at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX journal_case_idx ON journal (case_id, at DESC);
CREATE INDEX journal_project_idx ON journal (project_id, at DESC);

-- +goose Down
DROP TABLE journal;
DROP TABLE capture_verdicts;
DROP TABLE capture_references;
DROP TABLE comment_judgments;
DROP TABLE comment_variants;
DROP TABLE comments;
DROP TABLE recordings;
DROP TABLE captures;
DROP TABLE blobs;
DROP TABLE editions;
DROP TABLE steps;
DROP TABLE cases;
DROP TABLE variants;
DROP TABLE axes;
DROP TABLE categories;
DROP TABLE projects;
DROP FUNCTION short_id();
