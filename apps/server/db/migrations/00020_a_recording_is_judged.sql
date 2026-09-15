-- +goose Up
-- A recording is judged (ADR 0023): the verdict lands on the recording on
-- screen — one edition, one variant — and every judgment is kept, take-backs
-- included. Statuses are derived from the last row, never stored.
CREATE TABLE recording_judgments (
    id           text PRIMARY KEY DEFAULT short_id(),
    recording_id text NOT NULL REFERENCES recordings(id) ON DELETE CASCADE,
    verdict      text NOT NULL,
    remark       text,
    actor_id     text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX recording_judgments_recording_idx ON recording_judgments (recording_id, created_at);

-- +goose Down
DROP TABLE recording_judgments;
