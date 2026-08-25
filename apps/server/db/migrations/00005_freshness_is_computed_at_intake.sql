-- +goose Up
-- +goose StatementBegin

-- How many differing pixels a project is willing to call noise.
--
-- The per-channel tolerance -- how far two pixels may differ before they count
-- as different at all -- is fixed in the domain: it describes the same colour
-- rounded differently, which is a property of rendering, not of a project. This
-- is the only dial, and it is the one a project actually needs: how much noise
-- its own suite produces (product.md §3.3).
--
-- Zero means "any real difference counts", which is the honest default for a
-- suite nobody has measured yet.
ALTER TABLE projects
    ADD COLUMN pixel_threshold integer NOT NULL DEFAULT 0
        CHECK (pixel_threshold >= 0);

-- What the comparison found, computed once when the capture arrived.
--
-- NULL is not "unknown": it is "nothing to compare against". Either the case
-- had no reference in this capture's environment, or nobody has approved this
-- square yet. Freshness stays silent rather than guessing (ADR 0017).
ALTER TABLE captures
    ADD COLUMN freshness text
        CHECK (freshness IN ('current', 'to-re-review')),
    -- Recorded so the threshold can be judged rather than guessed. NULL when no
    -- comparison ran: identical hashes need none, and mismatched dimensions
    -- admit none.
    ADD COLUMN moved_pixels integer
        CHECK (moved_pixels >= 0);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE captures DROP COLUMN moved_pixels;
ALTER TABLE captures DROP COLUMN freshness;
ALTER TABLE projects DROP COLUMN pixel_threshold;

-- +goose StatementEnd
