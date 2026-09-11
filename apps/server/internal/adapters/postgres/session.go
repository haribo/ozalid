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
	// A held case takes no verdict but its holder's (ADR 0005, #95).
	if err := r.refuseHeld(ctx, q, caseID, by); err != nil {
		return session.Result{}, err
	}
	before := review.CaseState(kase.State)

	// What the saver is looking at (ADR 0024): their lock's stamped edition,
	// or the latest when nothing pins.
	shown, err := r.displayedEdition(ctx, q, kase)
	if err != nil {
		return session.Result{}, err
	}

	for _, c := range save.Comments {
		created, err := q.CreateComment(ctx, sqlcgen.CreateCommentParams{
			CaseID: caseID, StepID: c.StepID, Body: c.Body, AuthorID: by.ID,
		})
		if err != nil {
			return session.Result{}, translate("recording a comment", err)
		}
		for _, variantID := range c.VariantIDs {
			// Anchored to the bytes the reviewer is looking at (ADR 0024).
			anchor := ""
			if shown != nil {
				anchor = *shown
			}
			if err := q.AttachCommentVariant(ctx, sqlcgen.AttachCommentVariantParams{
				CommentID: created.ID, VariantID: variantID, EditionID: anchor,
			}); err != nil {
				return session.Result{}, translate("attaching a variant", err)
			}
		}
	}

	// Accepted captures are written before the computation reads them back, so
	// what it sees is the whole session and not half of it.
	for _, capture := range save.Accepted {
		if err := q.RecordCaptureAcceptance(ctx, sqlcgen.RecordCaptureAcceptanceParams{
			CaseID: caseID, StepID: capture.StepID, VariantID: capture.VariantID,
			AcceptedBy: &by.ID,
		}); err != nil {
			return session.Result{}, translate("recording the acceptance", err)
		}

		// And the bytes behind it are remembered, so a later run can say
		// whether this exact image moved. Only what the reviewer validated in
		// this sitting is stamped: a capture that turned `validated` because its
		// last comment was settled was never looked at, and claiming otherwise
		// would make "who approved this" a lie.
		//
		// A case showing no edition has nothing to remember yet.
		if shown == nil {
			continue
		}
		if err := q.StampCaptureReference(ctx, sqlcgen.StampCaptureReferenceParams{
			CaseID: caseID, StepID: capture.StepID, VariantID: capture.VariantID,
			EditionID: *shown, ApprovedBy: by.ID,
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
	// keeps every move, and the reference stamp stays — the reference is history,
	// not a verdict.
	for _, capture := range save.Unaccepted {
		if err := q.DeleteCaptureAcceptance(ctx, sqlcgen.DeleteCaptureAcceptanceParams{
			CaseID: caseID, StepID: capture.StepID, VariantID: capture.VariantID,
		}); err != nil {
			return session.Result{}, translate("taking the acceptance back", err)
		}

		refs, err := q.SettledRefsOnCapture(ctx, sqlcgen.SettledRefsOnCaptureParams{
			CaseID: caseID, StepID: capture.StepID, VariantID: capture.VariantID,
		})
		if err != nil {
			return session.Result{}, translate("reading the settled refs", err)
		}
		for _, ref := range refs {
			// Accepting had released the variant from the coverage
			// (ADR 0022); the take-back restores it, anchored to the bytes
			// on display at the pinned edition.
			if shown != nil {
				if err := q.RestoreCommentVariant(ctx, sqlcgen.RestoreCommentVariantParams{
					CommentID: ref.CommentID, VariantID: capture.VariantID,
					EditionID: *shown,
				}); err != nil {
					return session.Result{}, translate("restoring the coverage", err)
				}
			}
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
				Verdict: "taken-back", ActorID: by.ID, VariantID: &capture.VariantID,
			}); err != nil {
				return session.Result{}, translate("recording the take-back", err)
			}
			if _, err := r.derive(ctx, q, sqlcgen.Comment{
				ID: ref.CommentID, State: ref.CommentState,
			}); err != nil {
				return session.Result{}, err
			}
		}

		// The same rule for ref-less remarks (#175): the acceptance that
		// settled them is taken back, comment-level, history kept.
		remarks, err := q.SettledRemarksOnCapture(ctx, sqlcgen.SettledRemarksOnCaptureParams{
			CaseID: caseID, StepID: capture.StepID, VariantID: capture.VariantID,
		})
		if err != nil {
			return session.Result{}, translate("reading the settled remarks", err)
		}
		for _, remark := range remarks {
			if shown != nil {
				if err := q.RestoreCommentVariant(ctx, sqlcgen.RestoreCommentVariantParams{
					CommentID: remark.ID, VariantID: capture.VariantID,
					EditionID: *shown,
				}); err != nil {
					return session.Result{}, translate("restoring the coverage", err)
				}
			}
			to, err := review.Transition(review.CommentState(remark.State), review.MoveUnjudge, "")
			if err != nil {
				return session.Result{}, err
			}
			if err := q.SetCommentState(ctx, sqlcgen.SetCommentStateParams{
				ID: remark.ID, State: string(to),
			}); err != nil {
				return session.Result{}, translate("taking the judgment back", err)
			}
			if err := q.RecordJudgment(ctx, sqlcgen.RecordJudgmentParams{
				CommentID: remark.ID, Verdict: "taken-back", ActorID: by.ID,
				VariantID: &capture.VariantID,
			}); err != nil {
				return session.Result{}, translate("recording the take-back", err)
			}
		}
	}

	// Withdrawing a draft refusal takes the reviewer's own remark with it:
	// with no issue attached it never counted anywhere (ADR 0020, the explicit
	// exception to "nothing is deleted"). Tracked remarks go through the
	// judgment take-back instead.
	for _, capture := range save.Unrefused {
		drafts, err := q.DraftCommentsOnCapture(ctx, sqlcgen.DraftCommentsOnCaptureParams{
			CaseID: caseID, StepID: capture.StepID, VariantID: capture.VariantID, AuthorID: by.ID,
		})
		if err != nil {
			return session.Result{}, translate("reading the drafts", err)
		}
		for _, id := range drafts {
			if err := q.DeleteComment(ctx, id); err != nil {
				return session.Result{}, translate("withdrawing the remark", err)
			}
		}
	}

	facts, err := factsOfAt(ctx, q, kase, shown)
	if err != nil {
		return session.Result{}, err
	}
	outcome := review.Compute(facts)

	if outcome.State != before && before == review.CaseToReview {
		// When a review settles, the state re-derives against the latest
		// edition, past the saver's own lock (ADR 0024): the edition already
		// waiting can carry unjudged videos (ADR 0023), and nothing later
		// flips a state stamped blind to it.
		latest, err := r.latestEditionID(ctx, q, kase.ProjectID)
		if err != nil {
			return session.Result{}, err
		}
		facts, err = factsOfAt(ctx, q, kase, latest)
		if err != nil {
			return session.Result{}, err
		}
		outcome = review.Compute(facts)
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
			"captures": len(facts.Captures),
			"accepted": len(facts.Accepted),
			"comments": len(facts.Comments),
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

// isNoRows tells "the project has taken nothing in yet" from a real failure.
// The first is normal: a book starts empty (ADR 0008).
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
