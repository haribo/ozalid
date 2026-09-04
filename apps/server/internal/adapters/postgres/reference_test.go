package postgres_test

import (
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	appcomment "github.com/haribo/ozalid/apps/server/internal/app/comment"
	"github.com/haribo/ozalid/apps/server/internal/app/session"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
	"github.com/haribo/ozalid/internal/contract"
)

func TestValidatingASquareRemembersTheBytesThatWereApproved(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	hash := pushEdition(t, ctx, repo, project, kase, "the form on ci", "ci")
	cell := onlyCell(t, ctx, repo, project.Slug, kase.ID)

	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{
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
	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{
		Validated: []review.Cell{onlyCell(t, ctx, repo, project.Slug, kase.ID)},
	}); err != nil {
		t.Fatalf("saving the first review: %v", err)
	}

	fromLaptop := pushEdition(t, ctx, repo, project, kase, "the form on a laptop", "laptop")
	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{
		Validated: []review.Cell{onlyCell(t, ctx, repo, project.Slug, kase.ID)},
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
	cell := onlyCell(t, ctx, repo, project.Slug, kase.ID)

	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{
		Comments: []session.NewComment{{
			StepID: cell.StepID, Kind: "defect", Body: "the button is cropped",
			VariantIDs: []string{cell.VariantID},
		}},
	}); err != nil {
		t.Fatalf("saving the review: %v", err)
	}
	comments, err := repo.Queries().CaseComments(ctx, sqlcgen.CaseCommentsParams{CaseID: kase.ID, ProjectID: project.ID})
	if err != nil {
		t.Fatalf("reading the comments: %v", err)
	}
	if _, err := repo.Discard(ctx, project.Slug, comments[0].ID, actor.Actor{ID: "nina", Kind: actor.Human}, "intentional"); err != nil {
		t.Fatalf("discarding: %v", err)
	}

	after, err := repo.Queries().CaseInProject(ctx, sqlcgen.CaseInProjectParams{ID: kase.ID, Slug: project.Slug})
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

func TestTheJournalRecordsWhatTheActorSaysItIs(t *testing.T) {
	// The kind used to be inferred from the move: delivering was recorded as a
	// program, everything else as a person. A developer marking a fix delivered
	// by hand was written into the journal as a machine. The actor says what it
	// is, and nothing guesses (ADR 0018).
	ctx, repo, project, kase := intakeFixture(t)
	pushEdition(t, ctx, repo, project, kase, "the form on ci", "ci")
	cell := onlyCell(t, ctx, repo, project.Slug, kase.ID)

	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{
		Comments: []session.NewComment{{
			StepID: cell.StepID, Kind: "defect", Body: "the button is cropped",
			VariantIDs: []string{cell.VariantID},
		}},
	}); err != nil {
		t.Fatalf("saving the review: %v", err)
	}
	comments, err := repo.Queries().CaseComments(ctx, sqlcgen.CaseCommentsParams{CaseID: kase.ID, ProjectID: project.ID})
	if err != nil {
		t.Fatalf("reading the comments: %v", err)
	}

	// A person marks the fix delivered, which used to be recorded as a program.
	byHand := actor.Actor{ID: "dev", Kind: actor.Human}
	if _, err := repo.Track(ctx, project.Slug, comments[0].ID, byHand, appcomment.IssueRef{ID: "142"}); err != nil {
		t.Fatalf("tracking: %v", err)
	}
	if _, err := repo.Deliver(ctx, project.Slug, comments[0].ID, "", byHand); err != nil {
		t.Fatalf("delivering: %v", err)
	}

	var kind, id string
	if err := repo.Pool().QueryRow(ctx,
		`SELECT actor_kind, actor_id FROM journal
		 WHERE case_id = $1 AND cause = 'comment-deliver' ORDER BY at DESC LIMIT 1`,
		kase.ID,
	).Scan(&kind, &id); err != nil {
		t.Fatalf("reading the journal: %v", err)
	}
	if kind != string(actor.Human) {
		t.Errorf("actor kind = %q, want human — a person delivered it", kind)
	}
	if id != "dev" {
		t.Errorf("actor id = %q, want the actor that was carried", id)
	}
}

// A delivery advances the case onto the edition that carries the fix: judging
// it means reading those bytes, and the pin was showing the reviewer the
// screen from before it (#142).
func TestADeliveryAdvancesTheCaseOntoItsEdition(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)

	before := storeBlob(t, ctx, repo, "before the fix")
	after := storeBlob(t, ctx, repo, "after the fix")
	manifest := func(rev, hash string) contract.Manifest {
		return contract.Manifest{Revision: rev, Cases: []contract.ManifestCase{{
			ID: kase.ID,
			Steps: []contract.ManifestStep{{Name: "the door", Captures: []contract.ManifestCapture{
				{Variant: map[string]string{"theme": "light"}, Hash: hash},
			}}},
		}}}
	}
	first, err := repo.WriteEdition(ctx, project.Slug, manifest("one", before), nil)
	if err != nil {
		t.Fatalf("first edition: %v", err)
	}

	// A comment, tracked — the case sits at to-review and is pinned there.
	comments, err := repo.OfCase(ctx, project.Slug, kase.ID)
	if err != nil {
		t.Fatalf("reading comments: %v", err)
	}
	_ = comments
	q := sqlcgen.New(repo.Pool())
	author := person(t, ctx, q, "delivery-author", false)
	var stepID, variantID string
	if err := repo.Pool().QueryRow(ctx,
		`SELECT c.step_id, c.variant_id FROM captures c WHERE c.edition_id = $1`, first.EditionID,
	).Scan(&stepID, &variantID); err != nil {
		t.Fatalf("finding the capture: %v", err)
	}
	created, err := q.CreateComment(ctx, sqlcgen.CreateCommentParams{
		CaseID: kase.ID, StepID: stepID, Kind: "defect", Body: "off", AuthorID: author.ID,
	})
	if err != nil {
		t.Fatalf("creating the comment: %v", err)
	}
	byHand := actor.Actor{ID: author.ID, Kind: actor.Human}
	if _, err := repo.Track(ctx, project.Slug, created.ID, byHand, appcomment.IssueRef{ID: "9"}); err != nil {
		t.Fatalf("tracking: %v", err)
	}

	// The fix lands as a second edition; the pinned case does not follow yet.
	if _, err := repo.WriteEdition(ctx, project.Slug, manifest("two", after), nil); err != nil {
		t.Fatalf("second edition: %v", err)
	}

	// Delivering is what moves it: the reviewer must read the fix's bytes.
	if _, err := repo.Deliver(ctx, project.Slug, created.ID, "", byHand); err != nil {
		t.Fatalf("delivering: %v", err)
	}
	grid, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	if len(grid.Steps) != 1 || len(grid.Steps[0].Cells) != 1 {
		t.Fatalf("grid = %+v, want the single fixed cell", grid.Steps)
	}
	if grid.Steps[0].Cells[0].Hash != after {
		t.Errorf("the case still shows %s, want the delivered %s", grid.Steps[0].Cells[0].Hash, after)
	}
}
