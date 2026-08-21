-- +goose Up
-- Deleting a project must take its whole book with it. The initial schema
-- cascaded from projects down to cases, editions and variants, but the rows
-- pointing *at* a variant did not cascade — so the delete stopped halfway with
-- a foreign key violation and left the project undeletable.
--
-- A variant is never deleted on its own; it only disappears when its project
-- does. So anything referencing one goes with it.

ALTER TABLE captures
    DROP CONSTRAINT captures_variant_id_fkey,
    ADD CONSTRAINT captures_variant_id_fkey
        FOREIGN KEY (variant_id) REFERENCES variants(id) ON DELETE CASCADE;

ALTER TABLE recordings
    DROP CONSTRAINT recordings_variant_id_fkey,
    ADD CONSTRAINT recordings_variant_id_fkey
        FOREIGN KEY (variant_id) REFERENCES variants(id) ON DELETE CASCADE;

ALTER TABLE comment_variants
    DROP CONSTRAINT comment_variants_variant_id_fkey,
    ADD CONSTRAINT comment_variants_variant_id_fkey
        FOREIGN KEY (variant_id) REFERENCES variants(id) ON DELETE CASCADE;

ALTER TABLE capture_references
    DROP CONSTRAINT capture_references_variant_id_fkey,
    ADD CONSTRAINT capture_references_variant_id_fkey
        FOREIGN KEY (variant_id) REFERENCES variants(id) ON DELETE CASCADE;

ALTER TABLE capture_verdicts
    DROP CONSTRAINT capture_verdicts_variant_id_fkey,
    ADD CONSTRAINT capture_verdicts_variant_id_fkey
        FOREIGN KEY (variant_id) REFERENCES variants(id) ON DELETE CASCADE;

-- A blob is shared by every edition that produced the same bytes, so it is
-- deliberately NOT cascaded: it outlives the captures pointing at it, and
-- reclaiming one is a retention decision, not a side effect of a delete.

-- +goose Down
ALTER TABLE capture_verdicts
    DROP CONSTRAINT capture_verdicts_variant_id_fkey,
    ADD CONSTRAINT capture_verdicts_variant_id_fkey
        FOREIGN KEY (variant_id) REFERENCES variants(id);

ALTER TABLE capture_references
    DROP CONSTRAINT capture_references_variant_id_fkey,
    ADD CONSTRAINT capture_references_variant_id_fkey
        FOREIGN KEY (variant_id) REFERENCES variants(id);

ALTER TABLE comment_variants
    DROP CONSTRAINT comment_variants_variant_id_fkey,
    ADD CONSTRAINT comment_variants_variant_id_fkey
        FOREIGN KEY (variant_id) REFERENCES variants(id);

ALTER TABLE recordings
    DROP CONSTRAINT recordings_variant_id_fkey,
    ADD CONSTRAINT recordings_variant_id_fkey
        FOREIGN KEY (variant_id) REFERENCES variants(id);

ALTER TABLE captures
    DROP CONSTRAINT captures_variant_id_fkey,
    ADD CONSTRAINT captures_variant_id_fkey
        FOREIGN KEY (variant_id) REFERENCES variants(id);
