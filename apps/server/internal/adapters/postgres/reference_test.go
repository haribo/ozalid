package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres"
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
		Accepted: []review.Cell{cell},
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
		Accepted: []review.Cell{onlyCell(t, ctx, repo, project.Slug, kase.ID)},
	}); err != nil {
		t.Fatalf("saving the first review: %v", err)
	}

	fromLaptop := pushEdition(t, ctx, repo, project, kase, "the form on a laptop", "laptop")
	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{
		Accepted: []review.Cell{onlyCell(t, ctx, repo, project.Slug, kase.ID)},
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
			StepID: cell.StepID, Body: "the button is cropped",
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
			StepID: cell.StepID, Body: "the button is cropped",
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
		CaseID: kase.ID, StepID: stepID, Body: "off", AuthorID: author.ID,
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

// The verdict is a toggle until the review ends (#156): a misclick is taken
// back with the same key, the journal keeps both moves, and the reference
// stamp stays — freshness is history, not a verdict.
func TestAcceptingIsAToggle(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	pushEdition(t, ctx, repo, project, kase, "the form to toggle", "ci")
	cell := onlyCell(t, ctx, repo, project.Slug, kase.ID)
	nina := actor.Actor{ID: "nina", Kind: actor.Human}

	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, nina, session.Save{
		Accepted: []review.Cell{cell},
	}); err != nil {
		t.Fatalf("validating: %v", err)
	}

	out, err := repo.SaveReview(ctx, project.Slug, kase.ID, nina, session.Save{
		Unaccepted: []review.Cell{cell},
	})
	if err != nil {
		t.Fatalf("taking it back: %v", err)
	}
	if out.State != review.CaseToReview {
		t.Errorf("case = %q after the take-back, want to-review", out.State)
	}

	grid, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	for _, s := range grid.Steps {
		for _, c := range s.Cells {
			if c.VariantID == cell.VariantID && c.Status == "accepted" {
				t.Error("the cell still reads validated")
			}
		}
	}

	// The reference written at validation stays: it is what "has it moved
	// since somebody approved it" is measured from, whoever took the verdict
	// back afterwards.
	var refs int
	if err := repo.Pool().QueryRow(ctx,
		"SELECT count(*) FROM capture_references WHERE case_id = $1", kase.ID,
	).Scan(&refs); err != nil {
		t.Fatalf("counting references: %v", err)
	}
	if refs == 0 {
		t.Error("the reference stamp vanished with the verdict")
	}
}

// A capture can read validated without anyone having validated it: accepting
// the issue that covered it settles the reference, and settling was the
// judgment. Taking that validation back must then take the judgment back —
// the reference returns to to-review and the capture with it (#167). Before
// the fix, unvalidate deleted a verdict that did not exist and the recompute
// stamped validated right back: the button did nothing, forever.
func TestUnacceptTakesASettledJudgmentBack(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	pushEdition(t, ctx, repo, project, kase, "the door before judging", "ci")
	cell := onlyCell(t, ctx, repo, project.Slug, kase.ID)
	nina := actor.Actor{ID: "nina", Kind: actor.Human}
	q := repo.Queries()

	created, err := q.CreateComment(ctx, sqlcgen.CreateCommentParams{
		CaseID: kase.ID, StepID: cell.StepID, Body: "label inside the frame", AuthorID: nina.ID,
	})
	if err != nil {
		t.Fatalf("creating the comment: %v", err)
	}
	if err := q.AttachCommentVariant(ctx, sqlcgen.AttachCommentVariantParams{
		CommentID: created.ID, VariantID: cell.VariantID,
	}); err != nil {
		t.Fatalf("attaching the variant: %v", err)
	}
	if _, err := repo.Track(ctx, project.Slug, created.ID, nina, appcomment.IssueRef{ID: "129"}); err != nil {
		t.Fatalf("tracking: %v", err)
	}
	if _, err := repo.Deliver(ctx, project.Slug, created.ID, "", nina); err != nil {
		t.Fatalf("delivering: %v", err)
	}
	if _, err := repo.Judge(ctx, project.Slug, created.ID, "", nina, true, ""); err != nil {
		t.Fatalf("accepting: %v", err)
	}

	// Accepting settled the reference, so the capture derives validated — the
	// exact state the production case sat in.
	if status := statusOf(t, ctx, repo, project.Slug, kase.ID, cell); status != "accepted" {
		t.Fatalf("after accepting, cell = %q, want validated", status)
	}

	out, err := repo.SaveReview(ctx, project.Slug, kase.ID, nina, session.Save{
		Unaccepted: []review.Cell{cell},
	})
	if err != nil {
		t.Fatalf("taking the validation back: %v", err)
	}

	if status := statusOf(t, ctx, repo, project.Slug, kase.ID, cell); status == "accepted" {
		t.Error("the capture still reads validated: the take-back did nothing")
	}
	if out.State != review.CaseToReview {
		t.Errorf("case = %q after the take-back, want to-review", out.State)
	}

	states, err := q.CommentIssueStates(ctx, created.ID)
	if err != nil {
		t.Fatalf("reading the ref states: %v", err)
	}
	if len(states) != 1 || states[0] != "to-review" {
		t.Errorf("ref states = %v, want the acceptance taken back to to-review", states)
	}

	comment, err := q.GetComment(ctx, sqlcgen.GetCommentParams{ID: created.ID, Slug: project.Slug})
	if err != nil {
		t.Fatalf("reading the comment: %v", err)
	}
	if comment.State != "to-review" {
		t.Errorf("comment = %q, want to-review derived from its ref", comment.State)
	}

	// The take-back joins the judgment history: who un-accepted, and when, is
	// information exactly like the acceptance was (ADR 0012).
	judgments, err := q.CommentJudgments(ctx, created.ID)
	if err != nil {
		t.Fatalf("reading the judgments: %v", err)
	}
	if len(judgments) != 2 || judgments[len(judgments)-1].Verdict != "taken-back" {
		got := make([]string, len(judgments))
		for i, j := range judgments {
			got[i] = j.Verdict
		}
		t.Errorf("judgments = %v, want [accepted taken-back]", got)
	}
}

// statusOf reads one cell's status off the grid, as the client would.
func statusOf(t *testing.T, ctx context.Context, repo *postgres.Repository, slug, caseID string, cell review.Cell) string {
	t.Helper()
	grid, err := repo.CaseGrid(ctx, slug, caseID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	for _, s := range grid.Steps {
		for _, c := range s.Cells {
			if s.ID == cell.StepID && c.VariantID == cell.VariantID {
				return c.Status
			}
		}
	}
	t.Fatalf("cell %v not on the grid", cell)
	return ""
}

// The symmetry of the take-back (#171): a refusal is reconsidered exactly like
// an acceptance — the ref returns to to-review, the history keeps the move.
func TestARefusalCanBeTakenBack(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	pushEdition(t, ctx, repo, project, kase, "the door to refuse", "ci")
	cell := onlyCell(t, ctx, repo, project.Slug, kase.ID)
	nina := actor.Actor{ID: "nina", Kind: actor.Human}
	q := repo.Queries()

	created, err := q.CreateComment(ctx, sqlcgen.CreateCommentParams{
		CaseID: kase.ID, StepID: cell.StepID, Body: "too much green", AuthorID: nina.ID,
	})
	if err != nil {
		t.Fatalf("creating the comment: %v", err)
	}
	if err := q.AttachCommentVariant(ctx, sqlcgen.AttachCommentVariantParams{
		CommentID: created.ID, VariantID: cell.VariantID,
	}); err != nil {
		t.Fatalf("attaching the variant: %v", err)
	}
	if _, err := repo.Track(ctx, project.Slug, created.ID, nina, appcomment.IssueRef{ID: "130"}); err != nil {
		t.Fatalf("tracking: %v", err)
	}
	if _, err := repo.Deliver(ctx, project.Slug, created.ID, "", nina); err != nil {
		t.Fatalf("delivering: %v", err)
	}
	if _, err := repo.Judge(ctx, project.Slug, created.ID, "", nina, false, "still three green things"); err != nil {
		t.Fatalf("refusing: %v", err)
	}

	out, err := repo.Unjudge(ctx, project.Slug, created.ID, "", nina)
	if err != nil {
		t.Fatalf("taking the refusal back: %v", err)
	}
	if out.CommentState != review.CommentToReview {
		t.Errorf("comment = %q after the take-back, want to-review", out.CommentState)
	}
	if out.CaseState != review.CaseToReview {
		t.Errorf("case = %q after the take-back, want to-review", out.CaseState)
	}

	states, err := q.CommentIssueStates(ctx, created.ID)
	if err != nil {
		t.Fatalf("reading the ref states: %v", err)
	}
	if len(states) != 1 || states[0] != "to-review" {
		t.Errorf("ref states = %v, want the refusal taken back to to-review", states)
	}

	judgments, err := q.CommentJudgments(ctx, created.ID)
	if err != nil {
		t.Fatalf("reading the judgments: %v", err)
	}
	if len(judgments) != 2 || judgments[len(judgments)-1].Verdict != "taken-back" {
		got := make([]string, len(judgments))
		for i, j := range judgments {
			got[i] = j.Verdict
		}
		t.Errorf("judgments = %v, want [refused taken-back]", got)
	}
}

// Switching verdicts is one gesture, never a trip through "not judged"
// (ADR 0020): the client sends both moves in one save, and the server writes
// them in order.
func TestSwitchingVerdictsIsOneGesture(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	pushEdition(t, ctx, repo, project, kase, "the door to flip on", "ci")
	cell := onlyCell(t, ctx, repo, project.Slug, kase.ID)
	nina := actor.Actor{ID: "nina", Kind: actor.Human}

	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, nina, session.Save{
		Accepted: []review.Cell{cell},
	}); err != nil {
		t.Fatalf("accepting: %v", err)
	}

	// Accepted → refused: the refusal's remark and the acceptance take-back
	// travel together.
	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, nina, session.Save{
		Unaccepted: []review.Cell{cell},
		Comments: []session.NewComment{{
			StepID: cell.StepID, Body: "too much green", VariantIDs: []string{cell.VariantID},
		}},
	}); err != nil {
		t.Fatalf("switching to refused: %v", err)
	}
	if status := statusOf(t, ctx, repo, project.Slug, kase.ID, cell); status != "refused" {
		t.Fatalf("after the switch, cell = %q, want refused", status)
	}

	// Refused → accepted: the acceptance and the draft withdrawal travel
	// together, no sheet, no intermediate state.
	out, err := repo.SaveReview(ctx, project.Slug, kase.ID, nina, session.Save{
		Accepted:  []review.Cell{cell},
		Unrefused: []review.Cell{cell},
	})
	if err != nil {
		t.Fatalf("switching back: %v", err)
	}
	if status := statusOf(t, ctx, repo, project.Slug, kase.ID, cell); status != "accepted" {
		t.Errorf("after the switch back, cell = %q, want accepted", status)
	}
	if out.State != review.CaseReviewed {
		t.Errorf("case = %q, want reviewed — the draft went with its refusal", out.State)
	}
}

// A draft refusal withdrawn takes the reviewer's own remark with it: it never
// counted anywhere (ADR 0020, the explicit exception to "nothing is deleted").
func TestADraftRefusalWithdrawsItsRemark(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	pushEdition(t, ctx, repo, project, kase, "the door to refuse", "ci")
	cell := onlyCell(t, ctx, repo, project.Slug, kase.ID)
	nina := actor.Actor{ID: "nina", Kind: actor.Human}

	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, nina, session.Save{
		Comments: []session.NewComment{{
			StepID: cell.StepID, Body: "a remark I regret", VariantIDs: []string{cell.VariantID},
		}},
	}); err != nil {
		t.Fatalf("refusing: %v", err)
	}
	if status := statusOf(t, ctx, repo, project.Slug, kase.ID, cell); status != "refused" {
		t.Fatalf("after refusing, cell = %q, want refused", status)
	}

	out, err := repo.SaveReview(ctx, project.Slug, kase.ID, nina, session.Save{
		Unrefused: []review.Cell{cell},
	})
	if err != nil {
		t.Fatalf("withdrawing: %v", err)
	}
	if status := statusOf(t, ctx, repo, project.Slug, kase.ID, cell); status != "to-review" {
		t.Errorf("cell = %q after the withdrawal, want to-review", status)
	}
	if out.State != review.CaseToReview {
		t.Errorf("case = %q, want to-review", out.State)
	}
	comments, err := repo.OfCase(ctx, project.Slug, kase.ID)
	if err != nil {
		t.Fatalf("reading the comments: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("%d comment(s) remain — the withdrawn draft should be gone", len(comments))
	}
}

// A draft remark is the reviewer's own: editable — text and variants — while
// no issue is attached, and locked the moment one is (ADR 0020).
func TestADraftRemarkIsEditable(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	pushEdition(t, ctx, repo, project, kase, "the door to edit on", "ci")
	cell := onlyCell(t, ctx, repo, project.Slug, kase.ID)
	nina := actor.Actor{ID: "nina", Kind: actor.Human}

	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, nina, session.Save{
		Comments: []session.NewComment{{
			StepID: cell.StepID, Body: "too much gren", VariantIDs: []string{cell.VariantID},
		}},
	}); err != nil {
		t.Fatalf("refusing: %v", err)
	}
	comments, err := repo.OfCase(ctx, project.Slug, kase.ID)
	if err != nil || len(comments) != 1 {
		t.Fatalf("comments = %v, %v", comments, err)
	}
	id := comments[0].ID

	if _, err := repo.Edit(ctx, project.Slug, id, nina, "too much green", []string{cell.VariantID}); err != nil {
		t.Fatalf("editing: %v", err)
	}
	comments, _ = repo.OfCase(ctx, project.Slug, kase.ID)
	if comments[0].Body != "too much green" {
		t.Errorf("body = %q, want the edit", comments[0].Body)
	}

	// Someone else's draft is not editable.
	other := actor.Actor{ID: "sam", Kind: actor.Human}
	if _, err := repo.Edit(ctx, project.Slug, id, other, "mine now", []string{cell.VariantID}); !errors.Is(err, appcomment.ErrNotTheAuthor) {
		t.Errorf("err = %v, want ErrNotTheAuthor", err)
	}

	// Once tracked, the issue's title speaks: the draft is locked.
	if _, err := repo.Track(ctx, project.Slug, id, nina, appcomment.IssueRef{ID: "9"}); err != nil {
		t.Fatalf("tracking: %v", err)
	}
	if _, err := repo.Edit(ctx, project.Slug, id, nina, "rewriting history", []string{cell.VariantID}); !errors.Is(err, appcomment.ErrNotADraft) {
		t.Errorf("err = %v, want ErrNotADraft", err)
	}
}
