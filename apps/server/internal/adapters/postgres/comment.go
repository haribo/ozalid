package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	appcomment "github.com/haribo/ozalid/apps/server/internal/app/comment"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
)

func (r *Repository) Track(
	ctx context.Context, slug, commentID string, by actor.Actor, issue appcomment.IssueRef,
) (appcomment.Outcome, error) {
	// Attaching adds a row — a comment may split into several issues, each on
	// its own round (#138). Allowed as long as the comment is open.
	return r.move(ctx, slug, commentID, by, review.MoveTrack, "", func(ctx context.Context, q *sqlcgen.Queries, c sqlcgen.Comment) (review.CommentState, error) {
		if !review.CommentState(c.State).Open() {
			return "", review.ErrNotOpen
		}
		if _, err := q.CreateCommentIssue(ctx, sqlcgen.CreateCommentIssueParams{
			CommentID: commentID, IssueID: issue.ID,
			Url: nonEmpty(issue.URL), Title: nonEmpty(issue.Title),
		}); err != nil {
			return "", translate("attaching the issue", err)
		}
		return r.derive(ctx, q, c)
	})
}

func (r *Repository) Discard(
	ctx context.Context, slug, commentID string, by actor.Actor, reason string,
) (appcomment.Outcome, error) {
	return r.move(ctx, slug, commentID, by, review.MoveDiscard, reason, func(ctx context.Context, q *sqlcgen.Queries, c sqlcgen.Comment) (review.CommentState, error) {
		to, err := review.Transition(review.CommentState(c.State), review.MoveDiscard, reason)
		if err != nil {
			return "", err
		}
		if err := q.DiscardComment(ctx, sqlcgen.DiscardCommentParams{
			ID: commentID, State: string(to), DiscardReason: &reason,
		}); err != nil {
			return "", translate("discarding", err)
		}
		return to, nil
	})
}

func (r *Repository) Deliver(
	ctx context.Context, slug, commentID, issueRefID string, by actor.Actor,
) (appcomment.Outcome, error) {
	return r.move(ctx, slug, commentID, by, review.MoveDeliver, "", func(ctx context.Context, q *sqlcgen.Queries, c sqlcgen.Comment) (review.CommentState, error) {
		if _, err := r.moveRef(ctx, q, slug, commentID, issueRefID, review.MoveDeliver, ""); err != nil {
			return "", err
		}
		return r.derive(ctx, q, c)
	})
}

func (r *Repository) Judge(
	ctx context.Context, slug, commentID, issueRefID string, by actor.Actor, accept bool, remark string,
) (appcomment.Outcome, error) {
	move := review.MoveRefuse
	verdict := "refused"
	if accept {
		move, verdict = review.MoveAccept, "accepted"
	}

	return r.move(ctx, slug, commentID, by, move, remark, func(ctx context.Context, q *sqlcgen.Queries, c sqlcgen.Comment) (review.CommentState, error) {
		refID, err := r.moveRef(ctx, q, slug, commentID, issueRefID, move, remark)
		if err != nil {
			return "", err
		}
		// Every judgment is kept, not just the last: three round trips on one
		// ref is information (ADR 0012).
		if err := q.RecordJudgment(ctx, sqlcgen.RecordJudgmentParams{
			CommentID: commentID, CommentIssueID: &refID,
			Verdict: verdict, Remark: nonEmpty(remark), ActorID: by.ID,
		}); err != nil {
			return "", translate("recording the judgment", err)
		}
		return r.derive(ctx, q, c)
	})
}

// moveRef resolves which ref the caller means and applies the move to it.
//
// Named explicitly, or the comment's only one: with several attached, the
// server will not guess which fix was delivered or judged.
func (r *Repository) moveRef(
	ctx context.Context, q *sqlcgen.Queries, slug, commentID, issueRefID string, m review.Move, remark string,
) (string, error) {
	refs, err := q.GetCommentIssue(ctx, sqlcgen.GetCommentIssueParams{
		CommentID: commentID, Slug: slug, Column3: issueRefID,
	})
	if err != nil {
		return "", translate("reading the issue refs", err)
	}
	switch {
	case len(refs) == 0:
		return "", review.ErrMoveNotAllowed
	case len(refs) > 1:
		return "", appcomment.ErrAmbiguousIssue
	}
	ref := refs[0]

	to, err := review.TransitionRef(review.RefState(ref.State), m, remark)
	if err != nil {
		return "", err
	}
	if err := q.SetCommentIssueState(ctx, sqlcgen.SetCommentIssueStateParams{
		ID: ref.ID, State: string(to),
	}); err != nil {
		return "", translate("moving the ref", err)
	}
	return ref.ID, nil
}

// derive reads the comment's state off its refs and writes it back.
func (r *Repository) derive(ctx context.Context, q *sqlcgen.Queries, c sqlcgen.Comment) (review.CommentState, error) {
	states, err := q.CommentIssueStates(ctx, c.ID)
	if err != nil {
		return "", translate("reading the ref states", err)
	}
	refs := make([]review.RefState, len(states))
	for i, s := range states {
		refs[i] = review.RefState(s)
	}
	to := review.DeriveComment(review.CommentState(c.State), refs)
	if err := q.SetCommentState(ctx, sqlcgen.SetCommentStateParams{ID: c.ID, State: string(to)}); err != nil {
		return "", translate("writing the derived state", err)
	}
	return to, nil
}

// move applies one transition and everything it implies, in one transaction.
//
// The domain decides whether the move is allowed; this only writes what it
// decided, recomputes the case, and journals the result. Splitting those would
// let a comment move without its case following.
func (r *Repository) move(
	ctx context.Context,
	slug string,
	commentID string,
	by actor.Actor,
	m review.Move,
	reason string,
	write func(context.Context, *sqlcgen.Queries, sqlcgen.Comment) (review.CommentState, error),
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

	// Scoped by the project, so a comment belonging to somebody else is not
	// found rather than moved (#71).
	comment, err := q.GetComment(ctx, sqlcgen.GetCommentParams{ID: commentID, Slug: slug})
	if err != nil {
		return appcomment.Outcome{}, translate("reading the comment", err)
	}

	// Each move decides for itself: discard consults the comment machine, the
	// ref moves consult the ref machine and derive the comment from its refs.
	// A refused move is the domain's answer, not a database failure.
	to, err := write(ctx, q, comment)
	if err != nil {
		return appcomment.Outcome{}, err
	}

	// The comment was already proven to belong to this project, so its case
	// does too.
	kase, err := q.CaseInProject(ctx, sqlcgen.CaseInProjectParams{ID: comment.CaseID, Slug: slug})
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
			// The actor says what it is. Inferring it from the move recorded a
			// developer who delivered by hand as a program (ADR 0018).
			Cause: "comment-" + string(m), ActorID: by.ID, ActorKind: string(by.Kind),
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

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// OfCase reads every comment on a case, with its variants and its judgments.
//
// Scoped by the project: a case id from elsewhere reads as a case with no
// comments, never as somebody else's remarks (#71).
func (r *Repository) OfCase(ctx context.Context, slug, caseID string) ([]appcomment.Record, error) {
	kase, err := r.q.CaseInProject(ctx, sqlcgen.CaseInProjectParams{ID: caseID, Slug: slug})
	if err != nil {
		return nil, translate("reading the case", err)
	}

	rows, err := r.q.CaseComments(ctx, sqlcgen.CaseCommentsParams{
		CaseID: caseID, ProjectID: kase.ProjectID,
	})
	if err != nil {
		return nil, translate("reading the comments", err)
	}

	caseRefs, err := r.q.CaseCommentIssues(ctx, caseID)
	if err != nil {
		return nil, translate("reading the issue refs", err)
	}
	refsByComment := map[string][]sqlcgen.CaseCommentIssuesRow{}
	for _, ref := range caseRefs {
		refsByComment[ref.CommentID] = append(refsByComment[ref.CommentID], ref)
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
		for _, ref := range refsByComment[row.ID] {
			tracking := appcomment.IssueTracking{
				RefID: ref.ID, ID: ref.IssueID, State: review.RefState(ref.State),
			}
			if ref.Url != nil {
				tracking.URL = *ref.Url
			}
			if ref.Title != nil {
				tracking.Title = *ref.Title
			}
			if ref.LastRefusal != nil {
				tracking.LastRefusal = *ref.LastRefusal
			}
			record.Issues = append(record.Issues, tracking)
		}
		// The first ref doubles as the old single `issue`, for old readers.
		if len(record.Issues) > 0 {
			first := record.Issues[0]
			record.Issue = &appcomment.IssueRef{ID: first.ID, URL: first.URL, Title: first.Title}
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
