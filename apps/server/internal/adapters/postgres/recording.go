package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
)

// JudgeRecording renders a verdict on one flow video (ADR 0023): the judgment
// lands on the recording on screen, the row is kept forever, and the case is
// recomputed from the facts in the same transaction.
func (r *Repository) JudgeRecording(
	ctx context.Context, slug, recordingID string, by actor.Actor, accept bool, remark string,
) (review.CaseState, error) {
	verdict := "accepted"
	if !accept {
		if remark == "" {
			return "", review.ErrRemarkRequired
		}
		verdict = "refused"
	}
	return r.moveRecording(ctx, slug, recordingID, by, verdict, remark)
}

// UnjudgeRecording takes a video's verdict back, symmetrically: the recording
// returns to the reviewer, the history keeps the move (ADR 0023).
func (r *Repository) UnjudgeRecording(
	ctx context.Context, slug, recordingID string, by actor.Actor,
) (review.CaseState, error) {
	return r.moveRecording(ctx, slug, recordingID, by, "taken-back", "")
}

func (r *Repository) moveRecording(
	ctx context.Context, slug, recordingID string, by actor.Actor, verdict, remark string,
) (review.CaseState, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("beginning the judgment: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.q.WithTx(tx)

	// Scoped by the project, so somebody else's video is not found rather
	// than judged (#71).
	found, err := q.RecordingInProject(ctx, sqlcgen.RecordingInProjectParams{ID: recordingID, Slug: slug})
	if err != nil {
		if isNoRows(err) {
			return "", review.ErrMoveNotAllowed
		}
		return "", translate("finding the recording", err)
	}
	kase, err := q.CaseInProject(ctx, sqlcgen.CaseInProjectParams{ID: found.CaseID, Slug: slug})
	if err != nil {
		return "", translate("reading the case", err)
	}
	// A held case takes no verdict but its holder's (ADR 0005, #95).
	if err := r.refuseHeld(ctx, q, kase.ID, by); err != nil {
		return "", err
	}
	before := review.CaseState(kase.State)

	var remarkPtr *string
	if remark != "" {
		remarkPtr = &remark
	}
	if err := q.InsertRecordingJudgment(ctx, sqlcgen.InsertRecordingJudgmentParams{
		RecordingID: recordingID, Verdict: verdict, Remark: remarkPtr, ActorID: by.ID,
	}); err != nil {
		return "", translate("recording the judgment", err)
	}

	facts, err := r.factsOf(ctx, q, kase)
	if err != nil {
		return "", err
	}
	outcome := review.Compute(facts)
	if outcome.State != before {
		if err := q.SetCaseState(ctx, sqlcgen.SetCaseStateParams{
			ID: kase.ID, State: string(outcome.State),
		}); err != nil {
			return "", translate("moving the case", err)
		}
		inputs, err := json.Marshal(map[string]any{
			"recording": recordingID,
			"verdict":   verdict,
		})
		if err != nil {
			return "", fmt.Errorf("encoding the transition inputs: %w", err)
		}
		if err := q.RecordTransition(ctx, sqlcgen.RecordTransitionParams{
			ProjectID: kase.ProjectID, CaseID: &kase.ID,
			FromState: ptr(string(before)), ToState: ptr(string(outcome.State)),
			Cause: "recording-judged", ActorID: by.ID, ActorKind: string(by.Kind),
			Inputs: inputs, RuleVersion: 1,
		}); err != nil {
			return "", translate("journalling the transition", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("committing the judgment: %w", err)
	}
	return outcome.State, nil
}
