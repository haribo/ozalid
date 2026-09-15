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
func (r *Repository) factsOf(ctx context.Context, q *sqlcgen.Queries, kase sqlcgen.Case) (review.Facts, error) {
	editionID, err := r.displayedEdition(ctx, q, kase)
	if err != nil {
		return review.Facts{}, err
	}
	return factsOfAt(ctx, q, kase, editionID)
}

// factsOfAt reads the facts against one already-resolved edition — the
// settle-time re-derivation resolves to the latest itself (ADR 0024).
func factsOfAt(ctx context.Context, q *sqlcgen.Queries, kase sqlcgen.Case, editionID *string) (review.Facts, error) {
	var facts review.Facts

	threshold, err := q.PixelThresholdByProject(ctx, kase.ProjectID)
	if err != nil {
		return facts, translate("reading the pixel threshold", err)
	}
	facts.PixelThreshold = int(threshold)

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

	if editionID != nil {
		recordings, err := q.CaseRecordings(ctx, sqlcgen.CaseRecordingsParams{
			CaseID: kase.ID, EditionID: *editionID,
		})
		if err != nil {
			return facts, translate("reading the recordings", err)
		}
		judged, err := q.LastRecordingJudgments(ctx, sqlcgen.LastRecordingJudgmentsParams{
			CaseID: kase.ID, EditionID: *editionID,
		})
		if err != nil {
			return facts, translate("reading the recording judgments", err)
		}
		verdictByRecording := map[string]string{}
		for _, row := range judged {
			if row.Verdict != "taken-back" {
				verdictByRecording[row.RecordingID] = row.Verdict
			}
		}
		for _, r := range recordings {
			facts.Recordings = append(facts.Recordings, review.RecordingFact{
				ID: r.RecordingID, Verdict: verdictByRecording[r.RecordingID],
			})
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
