-- +goose Up
-- +goose StatementBegin

-- What a service account presents to prove itself.
--
-- The hash, never the token. The server can check that a token it is shown is
-- the right one, and cannot show it back: whoever reads a stolen dump of this
-- table gets nothing usable. That is also why a token is displayed once, at
-- creation — a lost one is replaced, never recovered.
CREATE TABLE service_tokens (
    id                 text PRIMARY KEY DEFAULT short_id(),
    service_account_id text NOT NULL REFERENCES service_accounts(id) ON DELETE CASCADE,
    -- What the token was for, in a human's words. A token nobody can name is a
    -- token nobody dares revoke.
    label      text NOT NULL,
    token_hash text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    -- Read on every accepted call. It is what tells an operator which tokens
    -- are still in use and which can go.
    last_used_at timestamptz
);

CREATE INDEX service_tokens_account_idx ON service_tokens (service_account_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE service_tokens;

-- +goose StatementEnd
