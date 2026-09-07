-- +goose Up
-- ADR 0021: statuses are derived at read time; only facts are stored. The
-- reviewer's acceptance is the fact — who, and when. capture_verdicts held
-- that fact mixed with computed results the derivation now produces on
-- demand, and captures.freshness was a stored conclusion nobody updated;
-- moved_pixels stays, the measurement is a fact.
CREATE TABLE capture_acceptances (
    case_id     text NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    step_id     text NOT NULL REFERENCES steps(id) ON DELETE CASCADE,
    variant_id  text NOT NULL REFERENCES variants(id) ON DELETE CASCADE,
    -- Null for rows migrated from before the actor was recorded: history did
    -- not note it, and it is not invented.
    accepted_by text,
    accepted_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (case_id, step_id, variant_id)
);

-- The approver of the reference is who accepted, by construction: accepting
-- is what stamps it (session.go). Best-effort — an acceptance whose capture
-- never had a reference keeps a null actor.
INSERT INTO capture_acceptances (case_id, step_id, variant_id, accepted_by, accepted_at)
SELECT v.case_id, v.step_id, v.variant_id,
       (SELECT r.approved_by FROM capture_references r
        WHERE r.case_id = v.case_id AND r.step_id = v.step_id AND r.variant_id = v.variant_id
        ORDER BY r.approved_at DESC LIMIT 1),
       v.updated_at
FROM capture_verdicts v
WHERE v.status = 'accepted';

DROP TABLE capture_verdicts;
ALTER TABLE captures DROP COLUMN freshness;

-- +goose Down
ALTER TABLE captures ADD COLUMN freshness text
    CHECK (freshness IN ('current', 'to-re-review'));

CREATE TABLE capture_verdicts (
    case_id    text NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    step_id    text NOT NULL REFERENCES steps(id) ON DELETE CASCADE,
    variant_id text NOT NULL REFERENCES variants(id),
    status     text NOT NULL DEFAULT 'to-review'
        CHECK (status IN ('to-review', 'refused', 'accepted')),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (case_id, step_id, variant_id)
);
INSERT INTO capture_verdicts (case_id, step_id, variant_id, status, updated_at)
SELECT case_id, step_id, variant_id, 'accepted', accepted_at FROM capture_acceptances;

DROP TABLE capture_acceptances;
