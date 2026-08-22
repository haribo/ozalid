package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	appcomment "github.com/haribo/ozalid/apps/server/internal/app/comment"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
)

func (r *Repository) Track(ctx context.Context, commentID, actorID string, issue appcomment.IssueRef) (appcomment.Outcome, error) {
	return r.move(ctx, commentID, actorID, review.MoveTrack, "", func(ctx context.Context, q *sqlcgen.Queries, to review.CommentState) error {
		return q.AttachIssue(ctx, sqlcgen.AttachIssueParams{
			ID: commentID, State: string(to),
			IssueRef: &issue.ID, IssueUrl: nonEmpty(issue.URL), IssueTitle: nonEmpty(issue.Title),
		})
	})
}

func (r *Repository) Discard(ctx context.Context, commentID, actorID, reason string) (appcomment.Outcome, error) {
	return r.move(ctx, commentID, actorID, review.MoveDiscard, reason, func(ctx context.Context, q *sqlcgen.Queries, to review.CommentState) error {
		return q.DiscardComment(ctx, sqlcgen.DiscardCommentParams{
			ID: commentID, State: string(to), DiscardReason: &reason,
		})
	})
}

func (r *Repository) Deliver(ctx context.Context, commentID, actorID string) (appcomment.Outcome, error) {
	return r.move(ctx, commentID, actorID, review.MoveDeliver, "", nil)
}

func (r *Repository) Judge(ctx context.Context, commentID, actorID string, accept bool, remark string) (appcomment.Outcome, error) {
	move := review.MoveRefuse
	verdict := "refused"
	if accept {
		move, verdict = review.MoveAccept, "accepted"
	}

	return r.move(ctx, commentID, actorID, move, remark, func(ctx context.Context, q *sqlcgen.Queries, to review.CommentState) error {
		if err := q.SetCommentState(ctx, sqlcgen.SetCommentStateParams{ID: commentID, State: string(to)}); err != nil {
			return err
		}
		// Every judgment is kept, not just the last: three round trips on one
		// comment is information (ADR 0012).
		return q.RecordJudgment(ctx, sqlcgen.RecordJudgmentParams{
			CommentID: commentID, Verdict: verdict, Remark: nonEmpty(remark), ActorID: actorID,
		})
	})
}

// move applies one transition and everything it implies, in one transaction.
//
// The domain decides whether the move is allowed; this only writes what it
// decided, recomputes the case, and journals the result. Splitting those would
// let a comment move without its case following.
func (r *Repository) move(
	ctx context.Context,
	commentID, actorID string,
	m review.Move,
	reason string,
	write func(context.Context, *sqlcgen.Queries, review.CommentState) error,
) (appcomment.Outcome, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return appcomment.Outcome{}, fmt.Errorf("beginning the move: %w", err)
	}
	defer func() {
		// best-effort: rolling back a committed transaction fails, and that
		// means the write went through.
		_ = tx.Rollback(ctx)
	}()
	q := r.q.WithTx(tx)

	comment, err := q.GetComment(ctx, commentID)
	if err != nil {
		return appcomment.Outcome{}, translate("reading the comment", err)
	}

	to, err := review.Transition(review.CommentState(comment.State), m, reason)
	if err != nil {
		// A refused move is the domain's answer, not a database failure.
		return appcomment.Outcome{}, err
	}

	if write != nil {
		if err := write(ctx, q, to); err != nil {
			return appcomment.Outcome{}, translate("applying the move", err)
		}
	} else if err := q.SetCommentState(ctx, sqlcgen.SetCommentStateParams{
		ID: commentID, State: string(to),
	}); err != nil {
		return appcomment.Outcome{}, translate("applying the move", err)
	}

	kase, err := q.GetCase(ctx, comment.CaseID)
	if err != nil {
		return appcomment.Outcome{}, translate("reading the case", err)
	}
	before := review.CaseState(kase.State)

	facts, err := gatherFacts(ctx, q, kase)
	if err != nil {
		return appcomment.Outcome{}, err
	}
	outcome := review.Compute(facts)

	for cell, status := range outcome.Verdicts {
		if err := q.UpsertCaptureVerdict(ctx, sqlcgen.UpsertCaptureVerdictParams{
			CaseID: kase.ID, StepID: cell.StepID, VariantID: cell.VariantID,
			Status: string(status),
		}); err != nil {
			return appcomment.Outcome{}, translate("recording a verdict", err)
		}
	}

	if outcome.State != before {
		inputs, err := json.Marshal(map[string]any{
			"comment": commentID,
			"move":    string(m),
			"from":    comment.State,
			"to":      string(to),
		})
		if err != nil {
			return appcomment.Outcome{}, fmt.Errorf("encoding the transition inputs: %w", err)
		}
		if err := q.SetCaseState(ctx, sqlcgen.SetCaseStateParams{
			ID: kase.ID, State: string(outcome.State),
		}); err != nil {
			return appcomment.Outcome{}, translate("moving the case", err)
		}
		if err := q.RecordTransition(ctx, sqlcgen.RecordTransitionParams{
			ProjectID: kase.ProjectID, CaseID: &kase.ID,
			FromState: ptr(string(before)), ToState: ptr(string(outcome.State)),
			Cause: "comment-" + string(m), ActorID: actorID, ActorKind: actorKind(m),
			Inputs: inputs, RuleVersion: 1,
		}); err != nil {
			return appcomment.Outcome{}, translate("journalling the transition", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return appcomment.Outcome{}, fmt.Errorf("committing the move: %w", err)
	}
	return appcomment.Outcome{CommentState: to, CaseState: outcome.State}, nil
}

// actorKind records whether a move came from a person or a program. Delivering
// is typically automated from a pipeline; judging never is.
func actorKind(m review.Move) string {
	if m == review.MoveDeliver {
		return "machine"
	}
	return "human"
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// OfCase reads every comment on a case, with its variants and its judgments.
func (r *Repository) OfCase(ctx context.Context, caseID string) ([]appcomment.Record, error) {
	rows, err := r.q.CaseComments(ctx, caseID)
	if err != nil {
		return nil, translate("reading the comments", err)
	}

	out := make([]appcomment.Record, 0, len(rows))
	for _, row := range rows {
		record := appcomment.Record{
			ID: row.ID, StepID: row.StepID, Kind: row.Kind, Body: row.Body,
			State:      review.CommentState(row.State),
			VariantIDs: row.VariantIds,
			AuthorID:   row.AuthorID,
			CreatedAt:  row.CreatedAt.Time,
		}
		if row.IssueRef != nil {
			record.Issue = &appcomment.IssueRef{ID: *row.IssueRef}
			if row.IssueUrl != nil {
				record.Issue.URL = *row.IssueUrl
			}
			if row.IssueTitle != nil {
				record.Issue.Title = *row.IssueTitle
			}
		}
		if row.DiscardReason != nil {
			record.DiscardReason = *row.DiscardReason
		}

		judgments, err := r.q.CommentJudgments(ctx, row.ID)
		if err != nil {
			return nil, translate("reading the judgments", err)
		}
		for _, j := range judgments {
			judgment := appcomment.Judgment{
				Verdict: j.Verdict, ActorID: j.ActorID, At: j.CreatedAt.Time,
			}
			if j.Remark != nil {
				judgment.Remark = *j.Remark
			}
			record.Judgments = append(record.Judgments, judgment)
		}
		out = append(out, record)
	}
	return out, nil
}
