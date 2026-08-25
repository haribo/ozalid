package postgres_test

import (
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/app/session"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
)

func TestValidatingASquareRemembersTheBytesThatWereApproved(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	hash := pushEdition(t, ctx, repo, project, kase, "the form on ci", "ci")
	cell := onlyCell(t, ctx, repo, kase.ID)

	if _, err := repo.SaveReview(ctx, kase.ID, "nina", session.Save{
		Validated: []review.Cell{cell},
	}); err != nil {
		t.Fatalf("saving the review: %v", err)
	}

	refs, err := repo.Queries().CaseReferences(ctx, kase.ID)
	if err != nil {
		t.Fatalf("reading the references: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("want one reference, got %d", len(refs))
	}
	if refs[0].BlobHash != hash {
		t.Errorf("reference holds %q, want the bytes that were on screen", refs[0].BlobHash)
	}
	if refs[0].EnvironmentID != "ci" {
		t.Errorf("environment = %q, want the capture's own (ADR 0017)", refs[0].EnvironmentID)
	}
	if refs[0].ApprovedBy != "nina" {
		t.Errorf("approved by %q — \"who approved this\" must stay honest", refs[0].ApprovedBy)
	}
}

func TestEachEnvironmentKeepsItsOwnReference(t *testing.T) {
	// Two machines rendering the same screen produce different bytes. One row
	// per square would make every alternation between them look like a change
	// (ADR 0017).
	ctx, repo, project, kase := intakeFixture(t)

	fromCI := pushEdition(t, ctx, repo, project, kase, "the form on ci", "ci")
	if _, err := repo.SaveReview(ctx, kase.ID, "nina", session.Save{
		Validated: []review.Cell{onlyCell(t, ctx, repo, kase.ID)},
	}); err != nil {
		t.Fatalf("saving the first review: %v", err)
	}

	fromLaptop := pushEdition(t, ctx, repo, project, kase, "the form on a laptop", "laptop")
	if _, err := repo.SaveReview(ctx, kase.ID, "nina", session.Save{
		Validated: []review.Cell{onlyCell(t, ctx, repo, kase.ID)},
	}); err != nil {
		t.Fatalf("saving the second review: %v", err)
	}

	refs, err := repo.Queries().CaseReferences(ctx, kase.ID)
	if err != nil {
		t.Fatalf("reading the references: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("want one reference per environment, got %d", len(refs))
	}
	held := map[string]string{}
	for _, r := range refs {
		held[r.EnvironmentID] = r.BlobHash
	}
	if held["ci"] != fromCI {
		t.Errorf("ci reference = %q, want %q — the laptop overwrote it", held["ci"], fromCI)
	}
	if held["laptop"] != fromLaptop {
		t.Errorf("laptop reference = %q, want %q", held["laptop"], fromLaptop)
	}
}

func TestASquareThatNobodyLookedAtIsNeverStamped(t *testing.T) {
	// A square turns `validated` when its last comment is settled. Nobody
	// approved those bytes, so nothing is remembered about them.
	ctx, repo, project, kase := intakeFixture(t)
	pushEdition(t, ctx, repo, project, kase, "the form on ci", "ci")
	cell := onlyCell(t, ctx, repo, kase.ID)

	if _, err := repo.SaveReview(ctx, kase.ID, "nina", session.Save{
		Comments: []session.NewComment{{
			StepID: cell.StepID, Kind: "defect", Body: "the button is cropped",
			VariantIDs: []string{cell.VariantID},
		}},
	}); err != nil {
		t.Fatalf("saving the review: %v", err)
	}
	comments, err := repo.Queries().CaseComments(ctx, kase.ID)
	if err != nil {
		t.Fatalf("reading the comments: %v", err)
	}
	if _, err := repo.Discard(ctx, comments[0].ID, "nina", "intentional"); err != nil {
		t.Fatalf("discarding: %v", err)
	}

	after, err := repo.Queries().GetCase(ctx, kase.ID)
	if err != nil {
		t.Fatalf("re-reading the case: %v", err)
	}
	if after.State != string(review.CaseReviewed) {
		t.Fatalf("state = %q, want reviewed once the only comment is settled", after.State)
	}

	refs, err := repo.Queries().CaseReferences(ctx, kase.ID)
	if err != nil {
		t.Fatalf("reading the references: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("got %d references, want none — nobody approved those bytes", len(refs))
	}
}
