-- +goose Up
-- +goose StatementBegin

-- A link sent to an address, good once.
--
-- The hash, never the link. A stolen dump of this table yields nothing that can
-- be followed — the same reason a service token is stored as a hash.
CREATE TABLE sign_in_links (
    id         text PRIMARY KEY DEFAULT short_id(),
    user_id    text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    link_hash  text NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    -- Stamped when followed. A link that has been used is refused rather than
    -- deleted: "this link was already used" is a better answer than "no such
    -- link", and someone who finds an old mail deserves to know which.
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX sign_in_links_user_idx ON sign_in_links (user_id);

-- What the browser carries afterwards.
--
-- A row rather than a signed, stateless token: deactivating an account has to
-- shut its sessions at once, and a signed token stays valid until it expires no
-- matter what the database says.
CREATE TABLE sessions (
    id           text PRIMARY KEY DEFAULT short_id(),
    user_id      text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash   text NOT NULL UNIQUE,
    expires_at   timestamptz NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz
);

CREATE INDEX sessions_user_idx ON sessions (user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE sessions;
DROP TABLE sign_in_links;

-- +goose StatementEnd
