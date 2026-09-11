-- +goose Up
-- Occupancy is its own axis (ADR 0005, #95): who is reviewing a case right
-- now, never folded into the cycle state. One live row per case; expiry is
-- read, not written — a lock whose heartbeat went silent past the window
-- simply stops counting.
CREATE TABLE case_locks (
    case_id    text PRIMARY KEY REFERENCES cases(id) ON DELETE CASCADE,
    account_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    claimed_at timestamptz NOT NULL DEFAULT now(),
    beaten_at  timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE case_locks;
