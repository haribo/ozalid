package postgres

import (
	"context"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
)

// factsOf reads everything the computation is allowed to see, and nothing
// else — facts only, never a stored status: the derivation must not eat its
// own output (ADR 0021).
//
// Captures are read at the edition the case is judged against — its pin —
// falling back to the project's latest when the case was never pinned.
func factsOf(ctx context.Context, q *sqlcgen.Queries, kase sqlcgen.Case) (review.Facts, error) {
	var facts review.Facts

	threshold, err := q.PixelThresholdByProject(ctx, kase.ProjectID)
	if err != nil {
		return facts, translate("reading the pixel threshold", err)
	}
	facts.PixelThreshold = int(threshold)

	editionID := kase.CurrentEditionID
	if editionID == nil {
		edition, err := q.LatestEdition(ctx, kase.ProjectID)
		if err != nil {
			if isNoRows(err) {
				// A book can start empty (ADR 0008): no edition, no captures.
				editionID = nil
			} else {
				return facts, translate("reading the edition", err)
			}
		} else {
			editionID = &edition.ID
		}
	}

	if editionID != nil {
		rows, err := q.CaseCaptureFacts(ctx, sqlcgen.CaseCaptureFactsParams{
			CaseID: kase.ID, EditionID: *editionID,
		})
		if err != nil {
			return facts, translate("reading the captures", err)
		}
		for _, row := range rows {
			fact := review.CaptureFact{
				Capture: review.Capture{StepID: row.StepID, VariantID: row.VariantID},
				Hash:    row.BlobHash,
			}
			// Coalesced in SQL: the empty string is "no reference" — a
			// content address is never empty.
			if row.ReferenceHash != "" {
				reference := row.ReferenceHash
				fact.Reference = &reference
			}
			if row.MovedPixels != nil {
				pixels := int(*row.MovedPixels)
				fact.MovedPixels = &pixels
			}
			facts.Captures = append(facts.Captures, fact)
		}
	}

	accepted, err := q.CaseAcceptedCaptures(ctx, kase.ID)
	if err != nil {
		return facts, translate("reading the acceptances", err)
	}
	for _, row := range accepted {
		facts.Accepted = append(facts.Accepted, review.Capture{StepID: row.StepID, VariantID: row.VariantID})
	}

	comments, err := q.CaseComments(ctx, sqlcgen.CaseCommentsParams{CaseID: kase.ID, ProjectID: kase.ProjectID})
	if err != nil {
		return facts, translate("reading the comments", err)
	}
	anchors, err := q.CaseCommentAnchors(ctx, kase.ID)
	if err != nil {
		return facts, translate("reading the anchors", err)
	}
	anchorsByComment := map[string][]review.CommentAnchor{}
	for _, row := range anchors {
		anchorsByComment[row.CommentID] = append(anchorsByComment[row.CommentID], review.CommentAnchor{
			Capture:    review.Capture{StepID: row.StepID, VariantID: row.VariantID},
			AnchorHash: row.AnchorHash,
		})
	}
	for _, c := range comments {
		facts.Comments = append(facts.Comments, review.Comment{
			State:    review.CommentState(c.State),
			Captures: anchorsByComment[c.ID],
		})
	}

	return facts, nil
}
