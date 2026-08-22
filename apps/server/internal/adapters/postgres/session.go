package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	"github.com/haribo/ozalid/apps/server/internal/app/session"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
)

// SaveReview writes a session and everything it implies, in one transaction.
//
// The comments, the verdicts they cover and the case's new state are one
// write: a state that disagrees with the comments is the single failure the
// whole model exists to make impossible (ADR 0002).
func (r *Repository) SaveReview(ctx context.Context, caseID, actorID string, save session.Save) (session.Result, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return session.Result{}, fmt.Errorf("beginning the review: %w", err)
	}
	defer func() {
		// best-effort: rolling back a committed transaction fails, and that
		// means the write went through.
		_ = tx.Rollback(ctx)
	}()
	q := r.q.WithTx(tx)

	kase, err := q.GetCase(ctx, caseID)
	if err != nil {
		return session.Result{}, translate("reading the case", err)
	}
	before := review.CaseState(kase.State)

	for _, c := range save.Comments {
		created, err := q.CreateComment(ctx, sqlcgen.CreateCommentParams{
			CaseID: caseID, StepID: c.StepID, Kind: c.Kind, Body: c.Body, AuthorID: actorID,
		})
		if err != nil {
			return session.Result{}, translate("recording a comment", err)
		}
		for _, variantID := range c.VariantIDs {
			if err := q.AttachCommentVariant(ctx, sqlcgen.AttachCommentVariantParams{
				CommentID: created.ID, VariantID: variantID,
			}); err != nil {
				return session.Result{}, translate("attaching a variant", err)
			}
		}
	}

	// Validated squares are written before the computation reads them back, so
	// what it sees is the whole session and not half of it.
	for _, cell := range save.Validated {
		if err := q.UpsertCaptureVerdict(ctx, sqlcgen.UpsertCaptureVerdictParams{
			CaseID: caseID, StepID: cell.StepID, VariantID: cell.VariantID,
			Status: string(review.CaptureValidated),
		}); err != nil {
			return session.Result{}, translate("recording a verdict", err)
		}
	}

	facts, err := gatherFacts(ctx, q, kase)
	if err != nil {
		return session.Result{}, err
	}
	outcome := review.Compute(facts)

	for cell, status := range outcome.Verdicts {
		if err := q.UpsertCaptureVerdict(ctx, sqlcgen.UpsertCaptureVerdictParams{
			CaseID: caseID, StepID: cell.StepID, VariantID: cell.VariantID,
			Status: string(status),
		}); err != nil {
			return session.Result{}, translate("recording a verdict", err)
		}
	}

	if outcome.State != before {
		if err := q.SetCaseState(ctx, sqlcgen.SetCaseStateParams{
			ID: caseID, State: string(outcome.State),
		}); err != nil {
			return session.Result{}, translate("moving the case", err)
		}
		// The fingerprint of what the computation consumed. Without it a
		// stored state is no regression oracle (ADR 0002).
		inputs, err := json.Marshal(map[string]any{
			"captures":  len(facts.Captures),
			"validated": len(facts.Validated),
			"comments":  len(facts.Comments),
		})
		if err != nil {
			return session.Result{}, fmt.Errorf("encoding the transition inputs: %w", err)
		}
		if err := q.RecordTransition(ctx, sqlcgen.RecordTransitionParams{
			ProjectID: kase.ProjectID, CaseID: &caseID,
			FromState: ptr(string(before)), ToState: ptr(string(outcome.State)),
			Cause: "review-saved", ActorID: actorID, ActorKind: "human",
			Inputs: inputs, RuleVersion: 1,
		}); err != nil {
			return session.Result{}, translate("journalling the transition", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return session.Result{}, fmt.Errorf("committing the review: %w", err)
	}
	return session.Result{
		State: outcome.State, Verdicts: outcome.Verdicts, Comments: len(save.Comments),
	}, nil
}

// gatherFacts reads everything the computation is allowed to see, and nothing
// else. What is not here cannot influence a state.
func gatherFacts(ctx context.Context, q *sqlcgen.Queries, kase sqlcgen.Case) (review.Facts, error) {
	var facts review.Facts

	edition, err := q.LatestEdition(ctx, kase.ProjectID)
	if err == nil {
		captures, err := q.CaseCaptureCells(ctx, sqlcgen.CaseCaptureCellsParams{
			CaseID: kase.ID, EditionID: edition.ID,
		})
		if err != nil {
			return facts, translate("reading the captures", err)
		}
		for _, c := range captures {
			facts.Captures = append(facts.Captures, review.Cell{StepID: c.StepID, VariantID: c.VariantID})
		}
	} else if !isNoRows(err) {
		return facts, translate("reading the edition", err)
	}

	validated, err := q.CaseValidatedCells(ctx, kase.ID)
	if err != nil {
		return facts, translate("reading the verdicts", err)
	}
	for _, v := range validated {
		facts.Validated = append(facts.Validated, review.Cell{StepID: v.StepID, VariantID: v.VariantID})
	}

	comments, err := q.CaseComments(ctx, kase.ID)
	if err != nil {
		return facts, translate("reading the comments", err)
	}
	for _, c := range comments {
		comment := review.Comment{State: review.CommentState(c.State)}
		for _, variantID := range c.VariantIds {
			comment.Cells = append(comment.Cells, review.Cell{StepID: c.StepID, VariantID: variantID})
		}
		facts.Comments = append(facts.Comments, comment)
	}

	return facts, nil
}

// isNoRows tells "the project has taken nothing in yet" from a real failure.
// The first is normal: a book starts empty (ADR 0008).
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
