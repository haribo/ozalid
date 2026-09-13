package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres"
	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	"github.com/haribo/ozalid/apps/server/internal/app/session"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
	"github.com/haribo/ozalid/internal/contract"
)

// flow names a case and the steps one run captured for it.
type flow struct {
	caseID string
	steps  []string
}

// seedEdition takes in one run covering every flow given, each step carrying a
// light and a dark capture.
//
// One run, one edition, every case in it: a grid read without a lock shows the
// project's latest edition (ADR 0024), so seeding case by case would leave all
// but the last one with nothing on display.
func seedEdition(t *testing.T, ctx context.Context, repo *postgres.Repository, project sqlcgen.Project, flows ...flow) {
	t.Helper()
	manifest := contract.Manifest{}
	for _, f := range flows {
		entry := contract.ManifestCase{ID: f.caseID}
		for i, name := range f.steps {
			light := storeBlob(t, ctx, repo, fmt.Sprintf("light %s %s %d", t.Name(), f.caseID, i))
			dark := storeBlob(t, ctx, repo, fmt.Sprintf("dark %s %s %d", t.Name(), f.caseID, i))
			entry.Steps = append(entry.Steps, contract.ManifestStep{
				Name: name,
				Captures: []contract.ManifestCapture{
					{Variant: map[string]string{"theme": "light"}, Hash: light},
					{Variant: map[string]string{"theme": "dark"}, Hash: dark},
				},
			})
		}
		manifest.Cases = append(manifest.Cases, entry)
	}
	if _, err := repo.WriteEdition(ctx, project.Slug, manifest, nil); err != nil {
		t.Fatalf("taking the edition in: %v", err)
	}
}

// capturesOf names a case's captures as the review vocabulary addresses them.
func capturesOf(t *testing.T, ctx context.Context, repo *postgres.Repository, project sqlcgen.Project, caseID string) []review.Capture {
	t.Helper()
	grid, err := repo.CaseGrid(ctx, project.Slug, caseID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	var captures []review.Capture
	for _, step := range grid.Steps {
		for _, capture := range step.Captures {
			captures = append(captures, review.Capture{StepID: step.ID, VariantID: capture.VariantID})
		}
	}
	return captures
}

// openCase files a new case under a category, so a test can build a tree.
func openCase(t *testing.T, ctx context.Context, q *sqlcgen.Queries, project sqlcgen.Project, categoryID *string, title string) sqlcgen.Case {
	t.Helper()
	kase, err := q.CreateCase(ctx, sqlcgen.CreateCaseParams{
		ProjectID: project.ID, CategoryID: categoryID, Title: title,
	})
	if err != nil {
		t.Fatalf("opening %q: %v", title, err)
	}
	return kase
}

// The queue is every capture reading to-review or moved, and nothing else
// (product.md §3.6, #204).
func TestTheQueueHoldsWhatAwaitsTheReviewer(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	q := repo.Queries()
	nina := actor.Actor{ID: "nina", Kind: actor.Human}

	// A case nobody ever captured: outside the funnel, never in the queue.
	openCase(t, ctx, q, project, kase.CategoryID, "never instrumented")
	// An archived case keeps its captures and leaves the catalogue — and the
	// queue with it (ADR 0014).
	gone := openCase(t, ctx, q, project, kase.CategoryID, "archived")

	seedEdition(t, ctx, repo, project,
		flow{caseID: kase.ID, steps: []string{"opens"}},
		flow{caseID: gone.ID, steps: []string{"opens"}},
	)
	if _, err := repo.ArchiveCase(ctx, project.Slug, gone.ID); err != nil {
		t.Fatalf("archiving: %v", err)
	}

	// One of the two captures is judged; the other still awaits a verdict.
	captures := capturesOf(t, ctx, repo, project, kase.ID)
	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, nina, session.Save{
		Accepted: captures[:1],
	}); err != nil {
		t.Fatalf("accepting one capture: %v", err)
	}

	entries, err := repo.ReviewQueue(ctx, project.Slug, nil)
	if err != nil {
		t.Fatalf("reading the queue: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("queue = %d entries, want the single unjudged capture: %+v", len(entries), entries)
	}
	entry := entries[0]
	if entry.CaseID != kase.ID {
		t.Errorf("case = %q, want %q", entry.CaseID, kase.ID)
	}
	if entry.Capture.Status != string(review.CaptureToReview) {
		t.Errorf("status = %q, want to-review", entry.Capture.Status)
	}
	if entry.CaseTitle != kase.Title || entry.StepName == "" || entry.Variant.Label == "" {
		t.Errorf("entry = %+v, want it to carry the case, the step and the variant", entry)
	}
}

// The case state is a superset: a case awaiting only a recording's judgment
// reads to-review with no capture in that status, and must contribute nothing
// — or the queue advertises captures the walk cannot show (#204).
func TestACaseAwaitingOnlyARecordingContributesNoEntry(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	captures := seedGridWithRecordings(t, ctx, repo, project, kase, "first")
	nina := actor.Actor{ID: "nina", Kind: actor.Human}

	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, nina, session.Save{
		Accepted: captures,
	}); err != nil {
		t.Fatalf("accepting every capture: %v", err)
	}

	// The recording is left unjudged, so the case still awaits the reviewer.
	read, err := repo.CaseByID(ctx, project.Slug, kase.ID)
	if err != nil {
		t.Fatalf("reading the case: %v", err)
	}
	if read.State != review.CaseToReview {
		t.Fatalf("case state = %q, want to-review — the fixture no longer sets up the superset", read.State)
	}

	entries, err := repo.ReviewQueue(ctx, project.Slug, nil)
	if err != nil {
		t.Fatalf("reading the queue: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("queue = %+v, want empty: every capture is judged", entries)
	}
}

// The reach is the category on screen and everything beneath it, at any depth
// (product.md §3.6).
func TestTheQueueReachesUnderTheCategoryItIsReadFrom(t *testing.T) {
	ctx, repo, project, _ := intakeFixture(t)
	q := repo.Queries()

	top, err := q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: project.ID, Name: "checkout", Position: 0,
	})
	if err != nil {
		t.Fatalf("creating the top category: %v", err)
	}
	middle, err := q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: project.ID, ParentID: &top.ID, Name: "payment", Position: 0,
	})
	if err != nil {
		t.Fatalf("creating the middle category: %v", err)
	}
	deep, err := q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: project.ID, ParentID: &middle.ID, Name: "cards", Position: 0,
	})
	if err != nil {
		t.Fatalf("creating the deep category: %v", err)
	}
	sibling, err := q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: project.ID, ParentID: &top.ID, Name: "delivery", Position: 1,
	})
	if err != nil {
		t.Fatalf("creating the sibling category: %v", err)
	}

	buried := openCase(t, ctx, q, project, &deep.ID, "pay by saved card")
	elsewhere := openCase(t, ctx, q, project, &sibling.ID, "choose a carrier")
	seedEdition(t, ctx, repo, project,
		flow{caseID: buried.ID, steps: []string{"opens"}},
		flow{caseID: elsewhere.ID, steps: []string{"opens"}},
	)

	// Two levels above, the buried case is in reach and the sibling branch is
	// not.
	under, err := repo.ReviewQueue(ctx, project.Slug, &middle.ID)
	if err != nil {
		t.Fatalf("reading the queue under the middle category: %v", err)
	}
	for _, entry := range under {
		if entry.CaseID != buried.ID {
			t.Errorf("entry from %q under the middle category, want only the buried case", entry.CaseTitle)
		}
	}
	if len(under) == 0 {
		t.Error("the queue under the middle category is empty, want the buried case's captures")
	}

	// At the root, nothing is excluded.
	whole, err := repo.ReviewQueue(ctx, project.Slug, nil)
	if err != nil {
		t.Fatalf("reading the whole queue: %v", err)
	}
	seen := map[string]bool{}
	for _, entry := range whole {
		seen[entry.CaseID] = true
	}
	if !seen[buried.ID] || !seen[elsewhere.ID] {
		t.Errorf("whole queue covers %v, want both branches", seen)
	}
}

// A case is finished before the next begins, and inside it the walk follows
// the flow: step position, then variant label (product.md §3.6).
func TestTheQueueOrdersByCaseThenStep(t *testing.T) {
	ctx, repo, project, first := intakeFixture(t)
	q := repo.Queries()

	// The catalogue orders cases by title, so "a…" is walked before "z…"
	// whatever order they were opened in.
	last := openCase(t, ctx, q, project, first.CategoryID, "zebra crossing")
	early := openCase(t, ctx, q, project, first.CategoryID, "a first flow")
	seedEdition(t, ctx, repo, project,
		flow{caseID: last.ID, steps: []string{"opens"}},
		flow{caseID: early.ID, steps: []string{"opens", "submits"}},
	)

	entries, err := repo.ReviewQueue(ctx, project.Slug, nil)
	if err != nil {
		t.Fatalf("reading the queue: %v", err)
	}
	if len(entries) != 6 {
		t.Fatalf("queue = %d entries, want 6: two steps and one step, two variants each", len(entries))
	}

	// The early case first, its two steps in position order.
	for i, want := range []struct {
		caseID   string
		stepName string
		variant  string
	}{
		{early.ID, "opens", "dark"},
		{early.ID, "opens", "light"},
		{early.ID, "submits", "dark"},
		{early.ID, "submits", "light"},
		{last.ID, "opens", "dark"},
		{last.ID, "opens", "light"},
	} {
		got := entries[i]
		if got.CaseID != want.caseID || got.StepName != want.stepName || got.Variant.Label != want.variant {
			t.Errorf("entry %d = %s/%s/%s, want %s/%s", i, got.CaseTitle, got.StepName, got.Variant.Label, want.stepName, want.variant)
		}
	}
}

// A clean book is the point of the exercise: it answers an empty queue, not an
// error (#204).
func TestAProjectWithNothingToReviewAnswersAnEmptyQueue(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	captures := seedGrid(t, ctx, repo, project, kase)

	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, actor.Actor{ID: "nina", Kind: actor.Human},
		session.Save{Accepted: captures}); err != nil {
		t.Fatalf("accepting everything: %v", err)
	}

	entries, err := repo.ReviewQueue(ctx, project.Slug, nil)
	if err != nil {
		t.Fatalf("reading the queue: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("queue = %+v, want empty", entries)
	}
}
