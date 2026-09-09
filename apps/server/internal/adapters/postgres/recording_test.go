package postgres_test

import (
	"context"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres"
	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	"github.com/haribo/ozalid/apps/server/internal/app/session"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
	"github.com/haribo/ozalid/internal/contract"
)

// seedGridWithRecordings is seedGrid plus one flow video per variant.
func seedGridWithRecordings(
	t *testing.T, ctx context.Context, repo *postgres.Repository,
	project sqlcgen.Project, kase sqlcgen.Case, take string,
) []review.Capture {
	t.Helper()
	light := storeBlob(t, ctx, repo, "light "+t.Name())
	dark := storeBlob(t, ctx, repo, "dark "+t.Name())
	videoLight := storeBlob(t, ctx, repo, "video light "+take+t.Name())
	videoDark := storeBlob(t, ctx, repo, "video dark "+take+t.Name())

	if _, err := repo.WriteEdition(ctx, project.Slug, contract.Manifest{
		Cases: []contract.ManifestCase{{
			ID: kase.ID,
			Steps: []contract.ManifestStep{{
				Name: "opens",
				Captures: []contract.ManifestCapture{
					{Variant: map[string]string{"theme": "light"}, Hash: light},
					{Variant: map[string]string{"theme": "dark"}, Hash: dark},
				},
			}},
			Recordings: []contract.ManifestRecording{
				{Variant: map[string]string{"theme": "light"}, Hash: videoLight},
				{Variant: map[string]string{"theme": "dark"}, Hash: videoDark},
			},
		}},
	}, nil); err != nil {
		t.Fatalf("taking the edition in: %v", err)
	}

	grid, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	captures := make([]review.Capture, 0, 2)
	for _, capture := range grid.Steps[0].Captures {
		captures = append(captures, review.Capture{StepID: grid.Steps[0].ID, VariantID: capture.VariantID})
	}
	return captures
}

func recordingStates(
	t *testing.T, ctx context.Context, repo *postgres.Repository, slug, caseID string,
) map[string]string {
	t.Helper()
	grid, err := repo.CaseGrid(ctx, slug, caseID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	out := map[string]string{}
	for _, r := range grid.Recordings {
		out[r.VariantID] = r.Status
	}
	return out
}

func caseState(t *testing.T, ctx context.Context, repo *postgres.Repository, slug, caseID string) string {
	t.Helper()
	kase, err := repo.Queries().CaseInProject(ctx, sqlcgen.CaseInProjectParams{ID: caseID, Slug: slug})
	if err != nil {
		t.Fatalf("re-reading the case: %v", err)
	}
	return kase.State
}

// A recording is evidence, and evidence is judged (ADR 0023, #226): an
// unjudged video keeps the case to-review even when every capture is
// accepted; the last accepted video settles it; a refusal hands the case to
// the dev with its remark.
func TestARecordingIsJudgedAndTheCaseFollows(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	captures := seedGridWithRecordings(t, ctx, repo, project, kase, "one")
	nina := actor.Actor{ID: "nina", Kind: actor.Human}

	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, nina, session.Save{Accepted: captures}); err != nil {
		t.Fatalf("accepting every capture: %v", err)
	}
	if state := caseState(t, ctx, repo, project.Slug, kase.ID); state != "to-review" {
		t.Fatalf("case = %q with two unjudged recordings, want to-review", state)
	}

	grid, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil || len(grid.Recordings) != 2 {
		t.Fatalf("recordings = %v, %v", grid.Recordings, err)
	}
	first, second := grid.Recordings[0], grid.Recordings[1]

	if _, err := repo.JudgeRecording(ctx, project.Slug, first.ID, nina, true, ""); err != nil {
		t.Fatalf("accepting the first video: %v", err)
	}
	if state := caseState(t, ctx, repo, project.Slug, kase.ID); state != "to-review" {
		t.Errorf("case = %q with one video still unjudged, want to-review", state)
	}
	if _, err := repo.JudgeRecording(ctx, project.Slug, second.ID, nina, true, ""); err != nil {
		t.Fatalf("accepting the second video: %v", err)
	}
	if state := caseState(t, ctx, repo, project.Slug, kase.ID); state != "accepted" {
		t.Errorf("case = %q once everything is judged, want accepted", state)
	}

	// Refusing one video hands the case to the dev, remark carried.
	if _, err := repo.JudgeRecording(ctx, project.Slug, first.ID, nina, false, "the flow stutters at the door"); err != nil {
		t.Fatalf("refusing the first video: %v", err)
	}
	if state := caseState(t, ctx, repo, project.Slug, kase.ID); state != "refused" {
		t.Errorf("case = %q with a refused video, want refused", state)
	}
	states := recordingStates(t, ctx, repo, project.Slug, kase.ID)
	if states[first.VariantID] != "refused" || states[second.VariantID] != "accepted" {
		t.Errorf("recording states = %v, want the refusal on its own variant", states)
	}

	// The take-back is symmetric: the video returns to the reviewer.
	if _, err := repo.UnjudgeRecording(ctx, project.Slug, first.ID, nina); err != nil {
		t.Fatalf("taking the refusal back: %v", err)
	}
	if states := recordingStates(t, ctx, repo, project.Slug, kase.ID); states[first.VariantID] != "to-review" {
		t.Errorf("recording = %q after the take-back, want to-review", states[first.VariantID])
	}
}

// Every push brings new bytes (ADR 0023): the accepted video of one edition
// says nothing about the next one's.
func TestANewEditionResetsTheRecordingToReview(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	captures := seedGridWithRecordings(t, ctx, repo, project, kase, "one")
	nina := actor.Actor{ID: "nina", Kind: actor.Human}

	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, nina, session.Save{Accepted: captures}); err != nil {
		t.Fatalf("accepting every capture: %v", err)
	}
	grid, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	for _, r := range grid.Recordings {
		if _, err := repo.JudgeRecording(ctx, project.Slug, r.ID, nina, true, ""); err != nil {
			t.Fatalf("accepting a video: %v", err)
		}
	}
	if state := caseState(t, ctx, repo, project.Slug, kase.ID); state != "accepted" {
		t.Fatalf("case = %q before the new edition, want accepted", state)
	}

	// Same captures, new video bytes — what a real pushing run does.
	seedGridWithRecordings(t, ctx, repo, project, kase, "two")

	if state := caseState(t, ctx, repo, project.Slug, kase.ID); state != "to-review" {
		t.Errorf("case = %q after the new edition, want to-review — nobody watched these bytes", state)
	}
	for variant, status := range recordingStates(t, ctx, repo, project.Slug, kase.ID) {
		if status != "to-review" {
			t.Errorf("recording %s = %q on the new edition, want to-review", variant, status)
		}
	}
}
