package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres"
	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	"github.com/haribo/ozalid/apps/server/internal/app/intake"
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

	got, err := repo.WriteEdition(ctx, project.Slug, m)
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
		if _, err := repo.WriteEdition(ctx, project.Slug, m); err != nil {
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

	if _, err := repo.WriteEdition(ctx, project.Slug, m); !errors.Is(err, intake.ErrUnknownCase) {
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

	_, err := repo.WriteEdition(ctx, project.Slug, m)
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
	if _, err := repo.WriteEdition(ctx, project.Slug, m); err != nil {
		t.Fatalf("taking the edition in: %v", err)
	}

	grid, err := repo.CaseGrid(ctx, kase.ID, nil)
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
	ctx, repo, _, kase := intakeFixture(t)

	// Not being instrumented is a legitimate state, not a failure (ADR 0012).
	grid, err := repo.CaseGrid(ctx, kase.ID, nil)
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
		})
		if err != nil {
			t.Fatalf("taking an edition in: %v", err)
		}
		return res.EditionID
	}

	first := push(before)
	push(after)

	// A case may sit on an older edition while it is being reviewed: an
	// incoming run must never destroy what a reviewer is looking at (ADR 0004).
	old, err := repo.CaseGrid(ctx, kase.ID, &first)
	if err != nil {
		t.Fatalf("reading the older edition: %v", err)
	}
	if old.Steps[0].Cells[0].Hash != before {
		t.Error("the older edition does not show the bytes it was taken with")
	}

	latest, err := repo.CaseGrid(ctx, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the latest edition: %v", err)
	}
	if latest.Steps[0].Cells[0].Hash != after {
		t.Error("the default read did not land on the most recent edition")
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
	}); err != nil {
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
	}); err != nil {
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
	if _, err := repo.WriteEdition(ctx, project.Slug, m); err != nil {
		t.Fatalf("first edition: %v", err)
	}

	// Pretend the reviewer judged it clean.
	if _, err := repo.Pool().Exec(ctx,
		"UPDATE cases SET state = 'reviewed' WHERE id = $1", kase.ID); err != nil {
		t.Fatalf("marking the case reviewed: %v", err)
	}

	if _, err := repo.WriteEdition(ctx, project.Slug, m); err != nil {
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
