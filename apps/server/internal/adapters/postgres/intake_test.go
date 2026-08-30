package postgres_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres"
	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	"github.com/haribo/ozalid/apps/server/internal/app/comment"
	"github.com/haribo/ozalid/apps/server/internal/app/intake"
	"github.com/haribo/ozalid/apps/server/internal/app/session"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
	"github.com/haribo/ozalid/internal/contract"
)

// intakeFixture gives a repository on a live database plus a project and a
// case to push against. It cleans up after itself so the suite is re-runnable.
func intakeFixture(t *testing.T) (context.Context, *postgres.Repository, sqlcgen.Project, sqlcgen.Case) {
	t.Helper()

	dsn := os.Getenv("OZALID_TEST_DSN")
	if dsn == "" {
		t.Skip("set OZALID_TEST_DSN to run the database tests (just db-test)")
	}

	ctx := context.Background()
	repo, err := postgres.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("opening the database: %v", err)
	}
	t.Cleanup(repo.Close)

	q := repo.Queries()
	// WriteEdition owns its own transaction, so these tests cannot be wrapped
	// in one and roll back. They write for real, which means the slug has to be
	// unique per run or the second run collides.
	slug := fmt.Sprintf("intake-%d", time.Now().UnixNano())
	project, err := q.CreateProject(ctx, sqlcgen.CreateProjectParams{
		Slug: slug, Name: t.Name(), IntakePolicy: "per-case",
	})
	if err != nil {
		t.Fatalf("creating the project: %v", err)
	}
	// WriteEdition owns its own transaction, so the fixture cannot wrap the
	// test in one. It deletes what it made instead.
	t.Cleanup(func() {
		if _, err := repo.Pool().Exec(ctx, "DELETE FROM projects WHERE id = $1", project.ID); err != nil {
			t.Errorf("cleaning up: %v", err)
		}
	})

	kase, err := q.CreateCase(ctx, sqlcgen.CreateCaseParams{
		ProjectID: project.ID, Title: "pay by card",
	})
	if err != nil {
		t.Fatalf("creating the case: %v", err)
	}
	return ctx, repo, project, kase
}

// storeBlob registers content so a manifest may reference it.
func storeBlob(t *testing.T, ctx context.Context, repo *postgres.Repository, body string) string {
	t.Helper()
	hash, size, err := contract.HashReader(strings.NewReader(body))
	if err != nil {
		t.Fatalf("hashing: %v", err)
	}
	if err := repo.RecordBlob(ctx, hash, size); err != nil {
		t.Fatalf("recording the content: %v", err)
	}
	return hash
}

func TestAnEditionIsWrittenWholeWithItsAxesAndVariants(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)

	light := storeBlob(t, ctx, repo, "the form, in light")
	dark := storeBlob(t, ctx, repo, "the form, in dark")

	m := contract.Manifest{
		Revision: "abc123",
		Cases: []contract.ManifestCase{{
			ID: kase.ID,
			Steps: []contract.ManifestStep{{
				Name: "opens the form",
				Captures: []contract.ManifestCapture{
					{Variant: map[string]string{"theme": "light"}, Hash: light},
					{Variant: map[string]string{"theme": "dark"}, Hash: dark},
				},
			}},
		}},
	}

	got, err := repo.WriteEdition(ctx, project.Slug, m, nil)
	if err != nil {
		t.Fatalf("taking the edition in: %v", err)
	}
	if got.Cases != 1 || got.Captures != 2 {
		t.Errorf("result = %+v, want 1 case and 2 captures", got)
	}

	// The axis was declared by first use — ozalid ships no list (ADR 0001).
	axes, err := repo.Pool().Query(ctx, "SELECT name FROM axes WHERE project_id = $1", project.ID)
	if err != nil {
		t.Fatalf("reading the axes: %v", err)
	}
	defer axes.Close()
	var names []string
	for axes.Next() {
		var n string
		if err := axes.Scan(&n); err != nil {
			t.Fatalf("scanning an axis: %v", err)
		}
		names = append(names, n)
	}
	if len(names) != 1 || names[0] != "theme" {
		t.Errorf("axes = %v, want [theme]", names)
	}
}

func TestTheSameImageAcrossTwoEditionsPointsAtOneBlob(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	unchanged := storeBlob(t, ctx, repo, "a screen that did not move")

	m := contract.Manifest{Cases: []contract.ManifestCase{{
		ID: kase.ID,
		Steps: []contract.ManifestStep{{
			Name:     "opens the form",
			Captures: []contract.ManifestCapture{{Variant: map[string]string{"theme": "light"}, Hash: unchanged}},
		}},
	}}}

	for i := range 2 {
		if _, err := repo.WriteEdition(ctx, project.Slug, m, nil); err != nil {
			t.Fatalf("edition %d: %v", i+1, err)
		}
	}

	// Two editions, two capture rows, one blob: this is what makes a full
	// visual history affordable (ADR 0004).
	var blobs int
	if err := repo.Pool().QueryRow(ctx,
		`SELECT count(DISTINCT c.blob_hash) FROM captures c
		 JOIN editions e ON e.id = c.edition_id WHERE e.project_id = $1`,
		project.ID).Scan(&blobs); err != nil {
		t.Fatalf("counting the blobs: %v", err)
	}
	if blobs != 1 {
		t.Errorf("counted %d distinct blobs across two editions, want 1", blobs)
	}
}

func TestAManifestNamingAnUnknownCaseIsRefusedAndLeavesNoEdition(t *testing.T) {
	ctx, repo, project, _ := intakeFixture(t)
	hash := storeBlob(t, ctx, repo, "orphan")

	m := contract.Manifest{Cases: []contract.ManifestCase{{
		ID: "000000000000",
		Steps: []contract.ManifestStep{{
			Name:     "opens the form",
			Captures: []contract.ManifestCapture{{Hash: hash}},
		}},
	}}}

	if _, err := repo.WriteEdition(ctx, project.Slug, m, nil); !errors.Is(err, intake.ErrUnknownCase) {
		t.Fatalf("err = %v, want ErrUnknownCase", err)
	}

	assertNoEdition(t, ctx, repo, project.ID)
}

func TestAManifestReferencingAbsentContentIsRefusedAndNamesIt(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)

	absent, _, _ := contract.HashReader(strings.NewReader("never uploaded"))
	m := contract.Manifest{Cases: []contract.ManifestCase{{
		ID: kase.ID,
		Steps: []contract.ManifestStep{{
			Name:     "opens the form",
			Captures: []contract.ManifestCapture{{Hash: absent}},
		}},
	}}}

	_, err := repo.WriteEdition(ctx, project.Slug, m, nil)
	var missing *intake.MissingContent
	if !errors.As(err, &missing) {
		t.Fatalf("err = %v, want MissingContent", err)
	}
	// Naming them is the point: the client uploads exactly those.
	if len(missing.Hashes) != 1 || missing.Hashes[0] != absent {
		t.Errorf("missing = %v, want the one absent address", missing.Hashes)
	}

	assertNoEdition(t, ctx, repo, project.ID)
}

func assertNoEdition(t *testing.T, ctx context.Context, repo *postgres.Repository, projectID string) {
	t.Helper()
	var editions int
	if err := repo.Pool().QueryRow(ctx,
		"SELECT count(*) FROM editions WHERE project_id = $1", projectID).Scan(&editions); err != nil {
		t.Fatalf("counting the editions: %v", err)
	}
	if editions != 0 {
		t.Errorf("a refused manifest left %d edition(s) behind", editions)
	}
}

func TestTheGridComesBackInStepOrderWithOnlyTheVariantsThatExist(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)

	light := storeBlob(t, ctx, repo, "step one, light")
	dark := storeBlob(t, ctx, repo, "step one, dark")
	second := storeBlob(t, ctx, repo, "step two, light only")

	m := contract.Manifest{
		Revision: "rev-1",
		Cases: []contract.ManifestCase{{
			ID: kase.ID,
			Steps: []contract.ManifestStep{
				{Name: "opens the form", Captures: []contract.ManifestCapture{
					{Variant: map[string]string{"theme": "light"}, Hash: light,
						Provenance: contract.Provenance{Browser: "chromium", Resolution: "1920x1080"}},
					{Variant: map[string]string{"theme": "dark"}, Hash: dark},
				}},
				// Not every variant exists at every step.
				{Name: "submits", Captures: []contract.ManifestCapture{
					{Variant: map[string]string{"theme": "light"}, Hash: second},
				}},
			},
		}},
	}
	if _, err := repo.WriteEdition(ctx, project.Slug, m, nil); err != nil {
		t.Fatalf("taking the edition in: %v", err)
	}

	grid, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}

	if grid.Revision != "rev-1" {
		t.Errorf("revision = %q, want it carried through", grid.Revision)
	}
	if len(grid.Steps) != 2 {
		t.Fatalf("got %d steps, want 2", len(grid.Steps))
	}
	// The order the manifest gave, not whatever the database felt like.
	if grid.Steps[0].Name != "opens the form" || grid.Steps[1].Name != "submits" {
		t.Errorf("steps = %q then %q, want manifest order", grid.Steps[0].Name, grid.Steps[1].Name)
	}
	if len(grid.Steps[0].Cells) != 2 || len(grid.Steps[1].Cells) != 1 {
		t.Errorf("cells = %d and %d, want 2 then 1 — a missing cell is absent, not null",
			len(grid.Steps[0].Cells), len(grid.Steps[1].Cells))
	}
	if len(grid.Variants) != 2 {
		t.Errorf("got %d variants, want the 2 that exist", len(grid.Variants))
	}
	if grid.Variants[0].Label >= grid.Variants[1].Label {
		t.Error("variants are not in label order, so the columns would move between reads")
	}

	// Provenance survives the round trip: byte comparison only means something
	// within one environment (ADR 0004).
	var found bool
	for _, cell := range grid.Steps[0].Cells {
		if cell.Provenance.Browser == "chromium" && cell.Provenance.Resolution == "1920x1080" {
			found = true
		}
	}
	if !found {
		t.Error("the provenance recorded at intake did not come back")
	}
}

func TestACaseThatWasNeverCapturedReadsAsAnEmptyGrid(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)

	// Not being instrumented is a legitimate state, not a failure (ADR 0012).
	grid, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	if grid.EditionID != "" || len(grid.Steps) != 0 || len(grid.Variants) != 0 {
		t.Errorf("grid = %+v, want it empty", grid)
	}
}

func TestAnOlderEditionCanStillBeRead(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)

	before := storeBlob(t, ctx, repo, "before the change")
	after := storeBlob(t, ctx, repo, "after the change")

	push := func(hash string) string {
		t.Helper()
		res, err := repo.WriteEdition(ctx, project.Slug, contract.Manifest{
			Cases: []contract.ManifestCase{{
				ID: kase.ID,
				Steps: []contract.ManifestStep{{
					Name:     "opens the form",
					Captures: []contract.ManifestCapture{{Variant: map[string]string{"theme": "light"}, Hash: hash}},
				}},
			}},
		}, nil)
		if err != nil {
			t.Fatalf("taking an edition in: %v", err)
		}
		return res.EditionID
	}

	first := push(before)
	push(after)

	// A case may sit on an older edition while it is being reviewed: an
	// incoming run must never destroy what a reviewer is looking at (ADR 0004).
	old, err := repo.CaseGrid(ctx, project.Slug, kase.ID, &first)
	if err != nil {
		t.Fatalf("reading the older edition: %v", err)
	}
	if old.Steps[0].Cells[0].Hash != before {
		t.Error("the older edition does not show the bytes it was taken with")
	}

	// And the default read stays there too. The case went to to-review on the
	// first captures, so a reviewer holds it; the second edition is stored but
	// does not become what they are judging (product.md §7, ADR 0017).
	// TestACaseCatchesUpOnceItsReviewEnds covers the other half: the case moves
	// onto the newest edition once the review ends.
	byDefault, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the default edition: %v", err)
	}
	if byDefault.Steps[0].Cells[0].Hash != before {
		t.Error("an incoming run moved the bytes under the reviewer")
	}
}

func TestACategoryCountsItsWholeDescendanceNotJustItsChildren(t *testing.T) {
	ctx, repo, project, _ := intakeFixture(t)
	q := repo.Queries()

	root, err := q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: project.ID, Name: "core", Position: 0,
	})
	if err != nil {
		t.Fatalf("creating the root: %v", err)
	}
	child, err := q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: project.ID, ParentID: &root.ID, Name: "account", Position: 0,
	})
	if err != nil {
		t.Fatalf("creating the child: %v", err)
	}
	grandchild, err := q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: project.ID, ParentID: &child.ID, Name: "signup", Position: 0,
	})
	if err != nil {
		t.Fatalf("creating the grandchild: %v", err)
	}

	// One case at each depth.
	for _, cat := range []string{root.ID, child.ID, grandchild.ID} {
		id := cat
		if _, err := q.CreateCase(ctx, sqlcgen.CreateCaseParams{
			ProjectID: project.ID, CategoryID: &id, Title: "case in " + id,
		}); err != nil {
			t.Fatalf("creating a case: %v", err)
		}
	}

	nodes, err := repo.CategoryTree(ctx, project.ID)
	if err != nil {
		t.Fatalf("reading the tree: %v", err)
	}

	counts := map[string]int64{}
	for _, n := range nodes {
		counts[n.Name] = n.Cases.Total()
	}
	// A branch in trouble has to be visible from the root, so the root counts
	// its grandchildren.
	if counts["core"] != 3 {
		t.Errorf("core counts %d cases, want 3 across the whole branch", counts["core"])
	}
	if counts["account"] != 2 {
		t.Errorf("account counts %d, want 2", counts["account"])
	}
	if counts["signup"] != 1 {
		t.Errorf("signup counts %d, want 1", counts["signup"])
	}
}

func TestACaseSummaryCountsItsCapturesAndReportsZeroWhenItHasNone(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)

	// Before any intake: present in the listing, with zeroes rather than absent.
	before, err := repo.SummariseCases(ctx, project.ID, nil)
	if err != nil {
		t.Fatalf("summarising: %v", err)
	}
	if len(before) != 1 || before[0].Captures.Total != 0 {
		t.Fatalf("summary = %+v, want the case present with no capture", before)
	}

	light := storeBlob(t, ctx, repo, "light")
	dark := storeBlob(t, ctx, repo, "dark")
	if _, err := repo.WriteEdition(ctx, project.Slug, contract.Manifest{
		Cases: []contract.ManifestCase{{
			ID: kase.ID,
			Steps: []contract.ManifestStep{{
				Name: "opens", Captures: []contract.ManifestCapture{
					{Variant: map[string]string{"theme": "light"}, Hash: light},
					{Variant: map[string]string{"theme": "dark"}, Hash: dark},
				},
			}},
		}},
	}, nil); err != nil {
		t.Fatalf("taking the edition in: %v", err)
	}

	after, err := repo.SummariseCases(ctx, project.ID, nil)
	if err != nil {
		t.Fatalf("summarising after intake: %v", err)
	}
	if after[0].Captures.Total != 2 {
		t.Errorf("counted %d captures, want 2", after[0].Captures.Total)
	}
	// No verdict has been written, so both are still waiting for a look.
	if after[0].Captures.ToJudge != 2 || after[0].Captures.Validated != 0 {
		t.Errorf("counts = %+v, want both still to judge", after[0].Captures)
	}
	if after[0].LastEdition == nil {
		t.Error("the summary does not carry the date of the edition it points at")
	}
}

func TestTheFirstCapturesTakeACaseOutOfTheFunnelsEdgeAndJournalIt(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)

	if kase.State != "not-instrumented" {
		t.Fatalf("the fixture starts at %q, want not-instrumented", kase.State)
	}

	hash := storeBlob(t, ctx, repo, "the first evidence")
	if _, err := repo.WriteEdition(ctx, project.Slug, contract.Manifest{
		Revision: "rev-1",
		Cases: []contract.ManifestCase{{
			ID: kase.ID,
			Steps: []contract.ManifestStep{{
				Name:     "opens",
				Captures: []contract.ManifestCapture{{Variant: map[string]string{"theme": "light"}, Hash: hash}},
			}},
		}},
	}, nil); err != nil {
		t.Fatalf("taking the edition in: %v", err)
	}

	after, err := repo.Queries().GetCase(ctx, kase.ID)
	if err != nil {
		t.Fatalf("re-reading the case: %v", err)
	}
	// Evidence exists and nobody has judged it: the reviewer holds the ball
	// (ADR 0012).
	if after.State != "to-review" {
		t.Errorf("state = %q, want to-review once captures exist", after.State)
	}

	// The transition is journalled, with what the computation consumed —
	// without that, a stored state is no regression oracle (ADR 0002).
	var from, to, cause, actorKind string
	var ruleVersion int
	if err := repo.Pool().QueryRow(ctx,
		`SELECT from_state, to_state, cause, actor_kind, rule_version
		 FROM journal WHERE case_id = $1 ORDER BY at DESC LIMIT 1`, kase.ID,
	).Scan(&from, &to, &cause, &actorKind, &ruleVersion); err != nil {
		t.Fatalf("reading the journal: %v", err)
	}
	if from != "not-instrumented" || to != "to-review" {
		t.Errorf("journalled %s → %s, want not-instrumented → to-review", from, to)
	}
	if cause != "edition-accepted" {
		t.Errorf("cause = %q, want the fact that caused it", cause)
	}
	if actorKind != "machine" {
		t.Errorf("actor kind = %q: intake is a program, and the journal must say so", actorKind)
	}
	if ruleVersion != 1 {
		t.Errorf("rule version = %d, want it recorded so a replay knows what it compares against", ruleVersion)
	}
}

func TestASecondEditionDoesNotReopenAJudgedCase(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	hash := storeBlob(t, ctx, repo, "evidence")

	m := contract.Manifest{Cases: []contract.ManifestCase{{
		ID: kase.ID,
		Steps: []contract.ManifestStep{{
			Name:     "opens",
			Captures: []contract.ManifestCapture{{Variant: map[string]string{"theme": "light"}, Hash: hash}},
		}},
	}}}
	if _, err := repo.WriteEdition(ctx, project.Slug, m, nil); err != nil {
		t.Fatalf("first edition: %v", err)
	}

	// Pretend the reviewer judged it clean.
	if _, err := repo.Pool().Exec(ctx,
		"UPDATE cases SET state = 'reviewed' WHERE id = $1", kase.ID); err != nil {
		t.Fatalf("marking the case reviewed: %v", err)
	}

	if _, err := repo.WriteEdition(ctx, project.Slug, m, nil); err != nil {
		t.Fatalf("second edition: %v", err)
	}

	after, err := repo.Queries().GetCase(ctx, kase.ID)
	if err != nil {
		t.Fatalf("re-reading: %v", err)
	}
	// An edition never moves the cycle: it only changes freshness. A reviewed
	// case stays reviewed until the reviewer says otherwise (ADR 0012).
	if after.State != "reviewed" {
		t.Errorf("state = %q, want reviewed: an incoming edition must not re-open a judged case", after.State)
	}
}

func TestOrderingTheAxesRelabelsTheVariantsAlreadyStored(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	hash := storeBlob(t, ctx, repo, "a screen")

	if _, err := repo.WriteEdition(ctx, project.Slug, contract.Manifest{
		Cases: []contract.ManifestCase{{
			ID: kase.ID,
			Steps: []contract.ManifestStep{{
				Name: "opens",
				Captures: []contract.ManifestCapture{{
					Variant: map[string]string{"theme": "dark", "viewport": "desktop"},
					Hash:    hash,
				}},
			}},
		}},
	}, nil); err != nil {
		t.Fatalf("taking the edition in: %v", err)
	}

	before, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	// Alphabetical by default — theme before viewport — which reads backwards.
	if before.Variants[0].Label != "dark·desktop" {
		t.Fatalf("initial label = %q, want the alphabetical default", before.Variants[0].Label)
	}
	variantID := before.Variants[0].ID

	if _, err := repo.OrderAxes(ctx, project.ID, []string{"viewport", "theme"}); err != nil {
		t.Fatalf("ordering the axes: %v", err)
	}

	after, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("re-reading the grid: %v", err)
	}
	if after.Variants[0].Label != "desktop·dark" {
		t.Errorf("label = %q, want it to follow the declared order", after.Variants[0].Label)
	}
	// A label is a rendering, never an identity: the row must be the same one.
	if after.Variants[0].ID != variantID {
		t.Error("relabelling created a new variant instead of renaming the existing one")
	}
}

func TestAnAxisTheProjectDidNotNameKeepsItsPlaceAfterThoseItDid(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	hash := storeBlob(t, ctx, repo, "a screen")

	if _, err := repo.WriteEdition(ctx, project.Slug, contract.Manifest{
		Cases: []contract.ManifestCase{{
			ID: kase.ID,
			Steps: []contract.ManifestStep{{
				Name: "opens",
				Captures: []contract.ManifestCapture{{
					Variant: map[string]string{"theme": "dark", "viewport": "mobile", "locale": "fr"},
					Hash:    hash,
				}},
			}},
		}},
	}, nil); err != nil {
		t.Fatalf("taking the edition in: %v", err)
	}

	// Only two of the three are named; the third must not vanish or jump ahead.
	axes, err := repo.OrderAxes(ctx, project.ID, []string{"viewport", "theme"})
	if err != nil {
		t.Fatalf("ordering the axes: %v", err)
	}
	if len(axes) != 3 {
		t.Fatalf("got %d axes, want the three that exist", len(axes))
	}
	if axes[0].Name != "viewport" || axes[1].Name != "theme" || axes[2].Name != "locale" {
		t.Errorf("order = %v, want the named ones first then the rest", axes)
	}

	grid, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	if grid.Variants[0].Label != "mobile·dark·fr" {
		t.Errorf("label = %q, want %q", grid.Variants[0].Label, "mobile·dark·fr")
	}
}

func TestNamingAnAxisNobodyCapturedDoesNotCreateIt(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	hash := storeBlob(t, ctx, repo, "a screen")

	if _, err := repo.WriteEdition(ctx, project.Slug, contract.Manifest{
		Cases: []contract.ManifestCase{{
			ID: kase.ID,
			Steps: []contract.ManifestStep{{
				Name: "opens",
				Captures: []contract.ManifestCapture{{
					Variant: map[string]string{"theme": "dark"}, Hash: hash,
				}},
			}},
		}},
	}, nil); err != nil {
		t.Fatalf("taking the edition in: %v", err)
	}

	// An axis exists because a capture mentioned it (ADR 0001). Inventing one
	// here would put an empty column in every grid.
	axes, err := repo.OrderAxes(ctx, project.ID, []string{"density", "theme"})
	if err != nil {
		t.Fatalf("ordering the axes: %v", err)
	}
	if len(axes) != 1 || axes[0].Name != "theme" {
		t.Errorf("axes = %v, want only the one that was captured", axes)
	}
}

// seedGrid gives a case two steps in two variants, all captured.
func seedGrid(t *testing.T, ctx context.Context, repo *postgres.Repository, project sqlcgen.Project, kase sqlcgen.Case) []review.Cell {
	t.Helper()
	light := storeBlob(t, ctx, repo, "light "+t.Name())
	dark := storeBlob(t, ctx, repo, "dark "+t.Name())

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
		}},
	}, nil); err != nil {
		t.Fatalf("taking the edition in: %v", err)
	}

	grid, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	cells := make([]review.Cell, 0, 2)
	for _, cell := range grid.Steps[0].Cells {
		cells = append(cells, review.Cell{StepID: grid.Steps[0].ID, VariantID: cell.VariantID})
	}
	return cells
}

func TestValidatingEverySquareWithNothingToSayClosesTheCase(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	cells := seedGrid(t, ctx, repo, project, kase)

	got, err := repo.SaveReview(ctx, project.Slug, kase.ID, actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{Validated: cells})
	if err != nil {
		t.Fatalf("saving the review: %v", err)
	}
	// reviewed is the only clean state (ADR 0012).
	if got.State != review.CaseReviewed {
		t.Errorf("state = %q, want reviewed", got.State)
	}

	after, err := repo.Queries().GetCase(ctx, kase.ID)
	if err != nil {
		t.Fatalf("re-reading: %v", err)
	}
	if after.State != "reviewed" {
		t.Errorf("stored state = %q, want it to match what was computed", after.State)
	}
}

func TestACommentPutsTheBallInTheDevsCourtAndMarksItsCells(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	cells := seedGrid(t, ctx, repo, project, kase)

	got, err := repo.SaveReview(ctx, project.Slug, kase.ID, actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{
		Validated: cells[:1],
		Comments: []session.NewComment{{
			StepID: cells[1].StepID, Kind: "defect",
			Body:       "the button is cropped in dark",
			VariantIDs: []string{cells[1].VariantID},
		}},
	})
	if err != nil {
		t.Fatalf("saving the review: %v", err)
	}
	if got.State != review.CaseToFix {
		t.Errorf("state = %q, want to-fix", got.State)
	}
	if got.Verdicts[cells[0]] != review.CaptureValidated {
		t.Errorf("the validated cell reads %q", got.Verdicts[cells[0]])
	}
	if got.Verdicts[cells[1]] != review.CaptureToFix {
		t.Errorf("the commented cell reads %q, want to-fix", got.Verdicts[cells[1]])
	}
}

func TestLeavingOneSquareUnjudgedKeepsTheCaseWaitingOnTheReviewer(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	cells := seedGrid(t, ctx, repo, project, kase)

	got, err := repo.SaveReview(ctx, project.Slug, kase.ID, actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{Validated: cells[:1]})
	if err != nil {
		t.Fatalf("saving the review: %v", err)
	}
	if got.State != review.CaseToReview {
		t.Errorf("state = %q, want to-review: an unjudged square is unfinished work", got.State)
	}
}

func TestTheStateChangeIsJournalledWithWhatTheComputationRead(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	cells := seedGrid(t, ctx, repo, project, kase)

	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{Validated: cells}); err != nil {
		t.Fatalf("saving the review: %v", err)
	}

	var from, to, cause, actor, kind string
	var inputs []byte
	if err := repo.Pool().QueryRow(ctx,
		`SELECT from_state, to_state, cause, actor_id, actor_kind, inputs
		 FROM journal WHERE case_id = $1 ORDER BY at DESC LIMIT 1`, kase.ID,
	).Scan(&from, &to, &cause, &actor, &kind, &inputs); err != nil {
		t.Fatalf("reading the journal: %v", err)
	}
	if from != "to-review" || to != "reviewed" {
		t.Errorf("journalled %s → %s", from, to)
	}
	if cause != "review-saved" || actor != "nina" || kind != "human" {
		t.Errorf("journalled cause=%q actor=%q kind=%q", cause, actor, kind)
	}
	// Without the inputs a stored state cannot serve as a regression oracle
	// (ADR 0002).
	var read map[string]int
	if err := json.Unmarshal(inputs, &read); err != nil {
		t.Fatalf("decoding the inputs: %v", err)
	}
	if read["captures"] != 2 || read["validated"] != 2 {
		t.Errorf("inputs = %v, want what the computation actually read", read)
	}
}

func TestSavingTwiceLeavesTheCaseWhereTheFactsPutIt(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	cells := seedGrid(t, ctx, repo, project, kase)

	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{Validated: cells}); err != nil {
		t.Fatalf("first save: %v", err)
	}
	// The same session again must not move anything: the state is a function
	// of the facts, not of how often it was computed.
	got, err := repo.SaveReview(ctx, project.Slug, kase.ID, actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{Validated: cells})
	if err != nil {
		t.Fatalf("second save: %v", err)
	}
	if got.State != review.CaseReviewed {
		t.Errorf("state = %q after a repeat save, want reviewed", got.State)
	}

	var transitions int
	if err := repo.Pool().QueryRow(ctx,
		"SELECT count(*) FROM journal WHERE case_id = $1 AND cause = 'review-saved'", kase.ID,
	).Scan(&transitions); err != nil {
		t.Fatalf("counting the transitions: %v", err)
	}
	// Only the move is journalled, not every save: a journal full of
	// no-op entries is one nobody reads.
	if transitions != 1 {
		t.Errorf("journalled %d transitions, want 1 — the second save moved nothing", transitions)
	}
}

// commentOn puts one comment on a case and returns its id.
func commentOn(t *testing.T, ctx context.Context, repo *postgres.Repository, slug, caseID string, cell review.Cell) string {
	t.Helper()
	if _, err := repo.SaveReview(ctx, slug, caseID, actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{
		Comments: []session.NewComment{{
			StepID: cell.StepID, Kind: "defect",
			Body: "the button is cropped", VariantIDs: []string{cell.VariantID},
		}},
	}); err != nil {
		t.Fatalf("writing the comment: %v", err)
	}
	comments, err := repo.OfCase(ctx, slug, caseID)
	if err != nil {
		t.Fatalf("reading the comments: %v", err)
	}
	return comments[len(comments)-1].ID
}

func TestACommentTravelsFromReportToClosureAndTakesTheCaseWithIt(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	cells := seedGrid(t, ctx, repo, project, kase)

	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{Validated: cells[:1]}); err != nil {
		t.Fatalf("validating the first cell: %v", err)
	}
	id := commentOn(t, ctx, repo, project.Slug, kase.ID, cells[1])

	// Reported, nothing tracked: the dev has to triage it.
	assertCaseState(t, ctx, repo, kase.ID, review.CaseToFix)

	out, err := repo.Track(ctx, id, actor.Actor{ID: "dev", Kind: actor.Human}, comment.IssueRef{ID: "142", URL: "https://example.test/142", Title: "Fix the cropped button"})
	if err != nil {
		t.Fatalf("tracking: %v", err)
	}
	if out.CommentState != review.CommentTracked || out.CaseState != review.CaseToFix {
		t.Errorf("after tracking: %+v, want tracked and the case still with the dev", out)
	}

	out, err = repo.Deliver(ctx, id, actor.Actor{ID: "ci", Kind: actor.Human})
	if err != nil {
		t.Fatalf("delivering: %v", err)
	}
	// The evidence is in: the ball goes back to the reviewer.
	if out.CommentState != review.CommentToReview || out.CaseState != review.CaseToReview {
		t.Errorf("after delivery: %+v, want the reviewer to hold the ball", out)
	}

	out, err = repo.Judge(ctx, id, actor.Actor{ID: "nina", Kind: actor.Human}, true, "")
	if err != nil {
		t.Fatalf("accepting: %v", err)
	}
	if out.CommentState != review.CommentValidated {
		t.Errorf("after accepting: %+v", out)
	}
	// The last open comment closed, and every square was judged: nothing left.
	if out.CaseState != review.CaseReviewed {
		t.Errorf("case = %q, want reviewed once the last comment closed", out.CaseState)
	}
}

func TestARefusalSendsItBackAndIsKeptForever(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	cells := seedGrid(t, ctx, repo, project, kase)
	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{Validated: cells[:1]}); err != nil {
		t.Fatalf("validating: %v", err)
	}
	id := commentOn(t, ctx, repo, project.Slug, kase.ID, cells[1])

	if _, err := repo.Track(ctx, id, actor.Actor{ID: "dev", Kind: actor.Human}, comment.IssueRef{ID: "142"}); err != nil {
		t.Fatalf("tracking: %v", err)
	}
	if _, err := repo.Deliver(ctx, id, actor.Actor{ID: "ci", Kind: actor.Human}); err != nil {
		t.Fatalf("delivering: %v", err)
	}

	out, err := repo.Judge(ctx, id, actor.Actor{ID: "nina", Kind: actor.Human}, false, "still cropped on iPhone SE")
	if err != nil {
		t.Fatalf("refusing: %v", err)
	}
	// A refusal is not a way to die: the ball returns to the dev.
	if out.CommentState != review.CommentRefused || out.CaseState != review.CaseToFix {
		t.Errorf("after refusing: %+v, want the dev to hold the ball", out)
	}

	if _, err := repo.Deliver(ctx, id, actor.Actor{ID: "ci", Kind: actor.Human}); err != nil {
		t.Fatalf("delivering again: %v", err)
	}
	if _, err := repo.Judge(ctx, id, actor.Actor{ID: "nina", Kind: actor.Human}, true, ""); err != nil {
		t.Fatalf("accepting the second try: %v", err)
	}

	comments, err := repo.OfCase(ctx, project.Slug, kase.ID)
	if err != nil {
		t.Fatalf("reading the comments: %v", err)
	}
	// Both judgments are kept: three round trips on one comment is
	// information, and only the trail shows it (ADR 0012).
	if len(comments[0].Judgments) != 2 {
		t.Fatalf("kept %d judgments, want both", len(comments[0].Judgments))
	}
	if comments[0].Judgments[0].Verdict != "refused" || comments[0].Judgments[0].Remark != "still cropped on iPhone SE" {
		t.Errorf("the refusal's remark did not survive: %+v", comments[0].Judgments[0])
	}
}

func TestADiscardedCommentStopsBlockingAndStaysVisible(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	cells := seedGrid(t, ctx, repo, project, kase)
	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{Validated: cells}); err != nil {
		t.Fatalf("validating: %v", err)
	}
	id := commentOn(t, ctx, repo, project.Slug, kase.ID, cells[1])

	out, err := repo.Discard(ctx, id, actor.Actor{ID: "nina", Kind: actor.Human}, "agreed it is intentional")
	if err != nil {
		t.Fatalf("discarding: %v", err)
	}
	if out.CaseState != review.CaseReviewed {
		t.Errorf("case = %q, want reviewed once nothing is open", out.CaseState)
	}

	comments, err := repo.OfCase(ctx, project.Slug, kase.ID)
	if err != nil {
		t.Fatalf("reading the comments: %v", err)
	}
	// Nothing is deleted: "who removed this, and why?" must have an answer
	// (ADR 0006).
	if len(comments) != 1 {
		t.Fatalf("got %d comments, want the discarded one still there", len(comments))
	}
	if comments[0].DiscardReason != "agreed it is intentional" || comments[0].AuthorID == "" {
		t.Errorf("the reason or its author was lost: %+v", comments[0])
	}
}

func TestAMoveTheStateDoesNotAllowIsRefusedWithoutTouchingAnything(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	cells := seedGrid(t, ctx, repo, project, kase)
	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{Validated: cells[:1]}); err != nil {
		t.Fatalf("validating: %v", err)
	}
	id := commentOn(t, ctx, repo, project.Slug, kase.ID, cells[1])

	// Nothing can be delivered before it is tracked.
	if _, err := repo.Deliver(ctx, id, actor.Actor{ID: "ci", Kind: actor.Human}); !errors.Is(err, review.ErrMoveNotAllowed) {
		t.Errorf("err = %v, want ErrMoveNotAllowed", err)
	}
	// Nor judged before it is delivered.
	if _, err := repo.Judge(ctx, id, actor.Actor{ID: "nina", Kind: actor.Human}, true, ""); !errors.Is(err, review.ErrMoveNotAllowed) {
		t.Errorf("err = %v, want ErrMoveNotAllowed", err)
	}

	comments, err := repo.OfCase(ctx, project.Slug, kase.ID)
	if err != nil {
		t.Fatalf("reading the comments: %v", err)
	}
	if comments[0].State != review.CommentToTrack {
		t.Errorf("state = %q after two refused moves, want it untouched", comments[0].State)
	}
}

func assertCaseState(t *testing.T, ctx context.Context, repo *postgres.Repository, caseID string, want review.CaseState) {
	t.Helper()
	got, err := repo.Queries().GetCase(ctx, caseID)
	if err != nil {
		t.Fatalf("reading the case: %v", err)
	}
	if review.CaseState(got.State) != want {
		t.Errorf("case state = %q, want %q", got.State, want)
	}
}
