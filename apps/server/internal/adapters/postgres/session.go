package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
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
func (r *Repository) SaveReview(
	ctx context.Context, slug, caseID string, by actor.Actor, save session.Save,
) (session.Result, error) {
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

	// Scoped by the project, so a case belonging to somebody else is not found
	// rather than judged (#71).
	kase, err := q.CaseInProject(ctx, sqlcgen.CaseInProjectParams{ID: caseID, Slug: slug})
	if err != nil {
		return session.Result{}, translate("reading the case", err)
	}
	before := review.CaseState(kase.State)

	for _, c := range save.Comments {
		created, err := q.CreateComment(ctx, sqlcgen.CreateCommentParams{
			CaseID: caseID, StepID: c.StepID, Kind: c.Kind, Body: c.Body, AuthorID: by.ID,
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

		// And the bytes behind it are remembered, so a later run can say
		// whether this exact image moved. Only what the reviewer validated in
		// this sitting is stamped: a square that turned `validated` because its
		// last comment was settled was never looked at, and claiming otherwise
		// would make "who approved this" a lie.
		//
		// A case pointing at no edition has nothing to remember yet.
		if kase.CurrentEditionID == nil {
			continue
		}
		if err := q.StampCaptureReference(ctx, sqlcgen.StampCaptureReferenceParams{
			CaseID: caseID, StepID: cell.StepID, VariantID: cell.VariantID,
			EditionID: *kase.CurrentEditionID, ApprovedBy: by.ID,
		}); err != nil {
			return session.Result{}, translate("stamping the reference", err)
		}
	}

	// Taking a validation back applies whatever made the capture validated
	// (#156, #167). An explicit validation is a row, and the row goes. A
	// validation derived from a settled reference has no row to delete —
	// deleting blindly left the recompute to stamp validated right back — so
	// the judgment itself is taken back: the ref returns to to-review, the
	// comment re-derives, and the capture follows. Either way the journal
	// keeps every move, and the reference stamp stays — freshness is history,
	// not a verdict.
	for _, cell := range save.Unvalidated {
		if err := q.DeleteCaptureVerdict(ctx, sqlcgen.DeleteCaptureVerdictParams{
			CaseID: caseID, StepID: cell.StepID, VariantID: cell.VariantID,
		}); err != nil {
			return session.Result{}, translate("taking a verdict back", err)
		}

		refs, err := q.SettledRefsOnCell(ctx, sqlcgen.SettledRefsOnCellParams{
			CaseID: caseID, StepID: cell.StepID, VariantID: cell.VariantID,
		})
		if err != nil {
			return session.Result{}, translate("reading the settled refs", err)
		}
		for _, ref := range refs {
			to, err := review.TransitionRef(review.RefState(ref.State), review.MoveUnjudge, "")
			if err != nil {
				return session.Result{}, err
			}
			if err := q.SetCommentIssueState(ctx, sqlcgen.SetCommentIssueStateParams{
				ID: ref.ID, State: string(to),
			}); err != nil {
				return session.Result{}, translate("taking the judgment back", err)
			}
			if err := q.RecordJudgment(ctx, sqlcgen.RecordJudgmentParams{
				CommentID: ref.CommentID, CommentIssueID: &ref.ID,
				Verdict: "taken-back", ActorID: by.ID,
			}); err != nil {
				return session.Result{}, translate("recording the take-back", err)
			}
			if _, err := r.derive(ctx, q, sqlcgen.Comment{
				ID: ref.CommentID, State: ref.CommentState,
			}); err != nil {
				return session.Result{}, err
			}
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

		// The reviewer has let go, so the case catches up with whatever landed
		// while they were looking. It was only held back to keep one fixed set
		// of bytes under them (product.md §7).
		if before == review.CaseToReview {
			if err := q.ReleaseToLatestEdition(ctx, caseID); err != nil {
				return session.Result{}, translate("releasing the case onto the latest edition", err)
			}
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
			// The actor says what it is; nothing here infers it (ADR 0018).
			Cause: "review-saved", ActorID: by.ID, ActorKind: string(by.Kind),
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

	comments, err := q.CaseComments(ctx, sqlcgen.CaseCommentsParams{CaseID: kase.ID, ProjectID: kase.ProjectID})
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
