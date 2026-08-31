package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	appcat "github.com/haribo/ozalid/apps/server/internal/app/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/app/evidence"
	"github.com/haribo/ozalid/internal/contract"
)

// CaseGrid reads one case's evidence at one edition.
func (r *Repository) CaseGrid(ctx context.Context, slug, caseID string, editionID *string) (evidence.Grid, error) {
	// The project is part of the lookup, so a case from elsewhere has no grid
	// here rather than one somebody may not see (#71).
	kase, err := r.q.CaseInProject(ctx, sqlcgen.CaseInProjectParams{ID: caseID, Slug: slug})
	if err != nil {
		return evidence.Grid{}, translate("reading the case", err)
	}

	edition, err := r.resolveEdition(ctx, kase, editionID)
	if errors.Is(err, evidence.ErrNoEdition) {
		// Nothing has been taken in yet. An empty grid is the honest answer.
		return evidence.Grid{CaseID: caseID}, nil
	}
	if err != nil {
		return evidence.Grid{}, err
	}

	grid := evidence.Grid{
		CaseID:    caseID,
		EditionID: edition.ID,
		TakenAt:   edition.CreatedAt.Time,
	}
	if edition.Revision != nil {
		grid.Revision = *edition.Revision
	}

	rows, err := r.q.CaseEvidence(ctx, sqlcgen.CaseEvidenceParams{
		CaseID: caseID, EditionID: edition.ID,
	})
	if err != nil {
		return evidence.Grid{}, translate("reading the evidence", err)
	}

	// One flat result set becomes steps and their cells. The variants are
	// collected as they appear, so the grid only mentions those that exist.
	variants := map[string]evidence.Variant{}
	var steps []evidence.Step
	byStep := map[string]int{}

	for _, row := range rows {
		idx, seen := byStep[row.StepID]
		if !seen {
			steps = append(steps, evidence.Step{
				ID: row.StepID, Name: row.StepName, Position: int(row.StepPosition),
			})
			idx = len(steps) - 1
			byStep[row.StepID] = idx
		}

		// A step with no capture at this edition still exists: the left join
		// gives it a row with no variant.
		if row.VariantID == nil {
			continue
		}

		if _, known := variants[*row.VariantID]; !known {
			values := map[string]string{}
			if err := json.Unmarshal(row.VariantValues, &values); err != nil {
				return evidence.Grid{}, fmt.Errorf("decoding a variant: %w", err)
			}
			variants[*row.VariantID] = evidence.Variant{
				ID: *row.VariantID, Label: *row.VariantLabel, Values: values,
			}
		}

		var provenance contract.Provenance
		if len(row.Provenance) > 0 {
			if err := json.Unmarshal(row.Provenance, &provenance); err != nil {
				return evidence.Grid{}, fmt.Errorf("decoding a provenance: %w", err)
			}
		}

		cell := evidence.Cell{
			ID: *row.CaptureID, VariantID: *row.VariantID, Hash: *row.BlobHash,
			Status: row.Status, Provenance: provenance,
		}
		if row.Freshness != nil {
			cell.Freshness = *row.Freshness
		}
		if row.MovedPixels != nil {
			moved := int(*row.MovedPixels)
			cell.MovedPixels = &moved
		}
		steps[idx].Cells = append(steps[idx].Cells, cell)
	}

	grid.Steps = steps
	grid.Variants = sortedVariants(variants)

	recordings, err := r.q.CaseRecordings(ctx, sqlcgen.CaseRecordingsParams{
		CaseID: caseID, EditionID: edition.ID,
	})
	if err != nil {
		return evidence.Grid{}, translate("reading the recordings", err)
	}
	for _, rec := range recordings {
		grid.Recordings = append(grid.Recordings, evidence.Recording{
			ID: rec.RecordingID, VariantID: rec.VariantID, Hash: rec.BlobHash,
		})
	}

	return grid, nil
}

// resolveEdition picks the edition to read against: the one asked for, then the
// one the case is being judged against, then the project's most recent.
//
// The case's own pointer comes before the latest edition on purpose. A run
// landing mid-review must not change what the reviewer is looking at, or they
// would judge one set of bytes and approve another (product.md §7).
func (r *Repository) resolveEdition(ctx context.Context, kase sqlcgen.Case, editionID *string) (sqlcgen.Edition, error) {
	if editionID == nil {
		editionID = kase.CurrentEditionID
	}
	if editionID != nil {
		edition, err := r.q.EditionByID(ctx, sqlcgen.EditionByIDParams{
			ID: *editionID, ProjectID: kase.ProjectID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlcgen.Edition{}, appcat.ErrNotFound
		}
		if err != nil {
			return sqlcgen.Edition{}, translate("reading the edition", err)
		}
		return edition, nil
	}

	edition, err := r.q.LatestEdition(ctx, kase.ProjectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlcgen.Edition{}, evidence.ErrNoEdition
	}
	if err != nil {
		return sqlcgen.Edition{}, translate("reading the latest edition", err)
	}
	return edition, nil
}

// sortedVariants returns the variants in label order, so the grid's columns are
// stable from one read to the next.
func sortedVariants(m map[string]evidence.Variant) []evidence.Variant {
	out := make([]evidence.Variant, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Label < out[j-1].Label; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// CaptureBlob answers where one capture's bytes are stored, inside the project
// the caller named.
func (r *Repository) CaptureBlob(ctx context.Context, slug, captureID string) (string, error) {
	hash, err := r.q.CaptureBlobInProject(ctx, sqlcgen.CaptureBlobInProjectParams{
		ID: captureID, Slug: slug,
	})
	if err != nil {
		return "", translate("reading the capture", err)
	}
	return hash, nil
}

// RecordingBlob does the same for a recording.
func (r *Repository) RecordingBlob(ctx context.Context, slug, recordingID string) (string, error) {
	hash, err := r.q.RecordingBlobInProject(ctx, sqlcgen.RecordingBlobInProjectParams{
		ID: recordingID, Slug: slug,
	})
	if err != nil {
		return "", translate("reading the recording", err)
	}
	return hash, nil
}
