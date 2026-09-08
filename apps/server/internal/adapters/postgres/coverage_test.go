package postgres_test

import (
	"context"
	"slices"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres"
	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	appcomment "github.com/haribo/ozalid/apps/server/internal/app/comment"
	"github.com/haribo/ozalid/apps/server/internal/app/session"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
)

// coverageFixture reports one remark over both variants of a step, tracks it
// and delivers the fix: the exact state a per-variant judgment starts from
// (ADR 0022).
func coverageFixture(t *testing.T) (context.Context, *postgres.Repository, sqlcgen.Project, sqlcgen.Case, []review.Capture, string) {
	t.Helper()
	ctx, repo, project, kase := intakeFixture(t)
	captures := seedGrid(t, ctx, repo, project, kase)
	nina := actor.Actor{ID: "nina", Kind: actor.Human}

	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, nina, session.Save{
		Comments: []session.NewComment{{
			StepID: captures[0].StepID, Body: "the label overflows",
			VariantIDs: []string{captures[0].VariantID, captures[1].VariantID},
		}},
	}); err != nil {
		t.Fatalf("reporting over both variants: %v", err)
	}
	comments, err := repo.OfCase(ctx, project.Slug, kase.ID)
	if err != nil || len(comments) != 1 {
		t.Fatalf("comments = %v, %v", comments, err)
	}
	id := comments[0].ID
	if _, err := repo.Track(ctx, project.Slug, id, nina, appcomment.IssueRef{ID: "208"}); err != nil {
		t.Fatalf("tracking: %v", err)
	}
	if _, err := repo.Deliver(ctx, project.Slug, id, "", nina); err != nil {
		t.Fatalf("delivering: %v", err)
	}
	return ctx, repo, project, kase, captures, id
}

// coverageOf reads which variants the comment still claims.
func coverageOf(t *testing.T, ctx context.Context, repo *postgres.Repository, slug, caseID, commentID string) []string {
	t.Helper()
	comments, err := repo.OfCase(ctx, slug, caseID)
	if err != nil {
		t.Fatalf("reading the comments: %v", err)
	}
	for _, c := range comments {
		if c.ID == commentID {
			ids := slices.Clone(c.VariantIDs)
			slices.Sort(ids)
			return ids
		}
	}
	t.Fatalf("comment %s not found", commentID)
	return nil
}

// Accepting the fix on one capture judges that capture only: the variant
// leaves the remark's coverage and reads accepted, while the other variant
// and the issue ref stay exactly where they were (ADR 0022, #208). On
// production rc.19, accepting one theme silently accepted the other.
func TestAcceptingAFixReleasesOneVariant(t *testing.T) {
	ctx, repo, project, kase, captures, id := coverageFixture(t)
	nina := actor.Actor{ID: "nina", Kind: actor.Human}

	if _, err := repo.Judge(ctx, project.Slug, id, "", captures[0].VariantID, nina, true, ""); err != nil {
		t.Fatalf("accepting on the first variant: %v", err)
	}

	if got := coverageOf(t, ctx, repo, project.Slug, kase.ID, id); !slices.Equal(got, []string{captures[1].VariantID}) {
		t.Errorf("coverage = %v, want only the unjudged variant %q", got, captures[1].VariantID)
	}
	if status := statusOf(t, ctx, repo, project.Slug, kase.ID, captures[0]); status != "accepted" {
		t.Errorf("judged variant = %q, want accepted", status)
	}
	if status := statusOf(t, ctx, repo, project.Slug, kase.ID, captures[1]); status == "accepted" {
		t.Error("the other variant reads accepted: the judgment leaked across variants")
	}
	states, err := repo.Queries().CommentIssueStates(ctx, id)
	if err != nil {
		t.Fatalf("reading the ref states: %v", err)
	}
	if len(states) != 1 || review.RefState(states[0]) == review.RefAccepted {
		t.Errorf("ref states = %v, want the ref still open for the remaining variant", states)
	}
}

// Refusing the fix on the remaining capture opens a partial round: the
// refusal stands on what is still covered, and the variant already accepted
// keeps its acceptance (ADR 0022, #208).
func TestRefusingAFixKeepsTheRemainingCoverage(t *testing.T) {
	ctx, repo, project, kase, captures, id := coverageFixture(t)
	nina := actor.Actor{ID: "nina", Kind: actor.Human}

	if _, err := repo.Judge(ctx, project.Slug, id, "", captures[0].VariantID, nina, true, ""); err != nil {
		t.Fatalf("accepting on the first variant: %v", err)
	}
	if _, err := repo.Judge(ctx, project.Slug, id, "", captures[1].VariantID, nina, false, "still overflows here"); err != nil {
		t.Fatalf("refusing on the second variant: %v", err)
	}

	if got := coverageOf(t, ctx, repo, project.Slug, kase.ID, id); !slices.Equal(got, []string{captures[1].VariantID}) {
		t.Errorf("coverage = %v, want the refused variant kept, the accepted one released", got)
	}
	states, err := repo.Queries().CommentIssueStates(ctx, id)
	if err != nil {
		t.Fatalf("reading the ref states: %v", err)
	}
	if len(states) != 1 || review.RefState(states[0]) != review.RefRefused {
		t.Errorf("ref states = %v, want refused for the partial round", states)
	}
	if status := statusOf(t, ctx, repo, project.Slug, kase.ID, captures[0]); status != "accepted" {
		t.Errorf("accepted variant = %q after the partial refusal, want its acceptance kept", status)
	}
}

// When the last covered variant is accepted, nothing is claimed any more:
// the ref settles, the remark with it, and the case closes (ADR 0022, #208).
func TestTheLastAcceptedVariantSettlesTheRemark(t *testing.T) {
	ctx, repo, project, kase, captures, id := coverageFixture(t)
	nina := actor.Actor{ID: "nina", Kind: actor.Human}

	if _, err := repo.Judge(ctx, project.Slug, id, "", captures[0].VariantID, nina, true, ""); err != nil {
		t.Fatalf("accepting on the first variant: %v", err)
	}
	out, err := repo.Judge(ctx, project.Slug, id, "", captures[1].VariantID, nina, true, "")
	if err != nil {
		t.Fatalf("accepting on the last variant: %v", err)
	}

	if out.CommentState != review.CommentAccepted {
		t.Errorf("comment = %q after the last acceptance, want accepted", out.CommentState)
	}
	if out.CaseState != review.CaseAccepted {
		t.Errorf("case = %q, want reviewed once nothing is claimed", out.CaseState)
	}
	for _, capture := range captures {
		if status := statusOf(t, ctx, repo, project.Slug, kase.ID, capture); status != "accepted" {
			t.Errorf("capture %v = %q, want accepted", capture, status)
		}
	}
}

// Taking one variant's acceptance back restores exactly that coverage: the
// variant is claimed again, anchored to the bytes on display, and the other
// variant's standing is untouched (ADR 0022, #208).
func TestUnjudgingOneVariantRestoresItsCoverage(t *testing.T) {
	ctx, repo, project, kase, captures, id := coverageFixture(t)
	nina := actor.Actor{ID: "nina", Kind: actor.Human}

	if _, err := repo.Judge(ctx, project.Slug, id, "", captures[0].VariantID, nina, true, ""); err != nil {
		t.Fatalf("accepting on the first variant: %v", err)
	}
	if _, err := repo.Unjudge(ctx, project.Slug, id, "", captures[0].VariantID, nina); err != nil {
		t.Fatalf("taking the acceptance back: %v", err)
	}

	want := []string{captures[0].VariantID, captures[1].VariantID}
	slices.Sort(want)
	if got := coverageOf(t, ctx, repo, project.Slug, kase.ID, id); !slices.Equal(got, want) {
		t.Errorf("coverage = %v, want both variants claimed again", got)
	}
	if status := statusOf(t, ctx, repo, project.Slug, kase.ID, captures[0]); status == "accepted" {
		t.Error("the variant still reads accepted: the take-back did nothing")
	}
	comments, err := repo.OfCase(ctx, project.Slug, kase.ID)
	if err != nil || len(comments) != 1 {
		t.Fatalf("comments = %v, %v", comments, err)
	}
	judgments := comments[0].Judgments
	last := judgments[len(judgments)-1]
	if last.Verdict != "taken-back" || last.VariantID != captures[0].VariantID {
		t.Errorf("last judgment = %+v, want taken-back naming the variant", last)
	}
}
