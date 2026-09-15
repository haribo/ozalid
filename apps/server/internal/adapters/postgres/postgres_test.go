package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres"
	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
)

// withTx gives each test its own transaction and rolls it back, so the suite is
// re-runnable and two tests never see each other's rows.
func withTx(t *testing.T) (context.Context, *sqlcgen.Queries) {
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

	tx, err := repo.Begin(ctx)
	if err != nil {
		t.Fatalf("beginning the transaction: %v", err)
	}
	t.Cleanup(func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rolling back: %v", err)
		}
	})

	return ctx, repo.Queries().WithTx(tx)
}

func seedProject(t *testing.T, ctx context.Context, q *sqlcgen.Queries) sqlcgen.Project {
	t.Helper()
	p, err := q.CreateProject(ctx, sqlcgen.CreateProjectParams{
		Slug: "test-" + t.Name(), Name: t.Name(), IntakePolicy: "per-case",
	})
	if err != nil {
		t.Fatalf("creating the project: %v", err)
	}
	return p
}

// seedCategory gives a project the branch every case now has to hang from: a
// case belongs to exactly one category, and the insert refuses one that is not
// this project's (#115).
func seedCategory(t *testing.T, ctx context.Context, q *sqlcgen.Queries, projectID string) *string {
	t.Helper()
	c, err := q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: projectID, Name: "seeded", Position: 0,
	})
	if err != nil {
		t.Fatalf("creating the category: %v", err)
	}
	return &c.ID
}

func TestACaseIsCreatedOutsideTheFunnelAndKeepsItsGeneratedID(t *testing.T) {
	ctx, q := withTx(t)
	project := seedProject(t, ctx, q)

	category, err := q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: project.ID, Name: "checkout", Position: 0,
	})
	if err != nil {
		t.Fatalf("creating the category: %v", err)
	}

	created, err := q.CreateCase(ctx, sqlcgen.CreateCaseParams{
		ProjectID: project.ID, CategoryID: &category.ID, Title: "pay by card",
	})
	if err != nil {
		t.Fatalf("creating the case: %v", err)
	}

	// The server generates the id; the client never invents it (ADR 0014).
	if len(created.ID) != 12 {
		t.Errorf("case id = %q, want 12 characters", created.ID)
	}
	// No capture and no verdict means outside the funnel (ADR 0012).
	if created.State != "not-instrumented" {
		t.Errorf("state = %q, want not-instrumented on a fresh case", created.State)
	}
}

func TestArchivingRemovesACaseFromTheCatalogueButNotFromTheBook(t *testing.T) {
	ctx, q := withTx(t)
	project := seedProject(t, ctx, q)

	created, err := q.CreateCase(ctx, sqlcgen.CreateCaseParams{
		ProjectID: project.ID, CategoryID: seedCategory(t, ctx, q, project.ID), Title: "pay by card",
	})
	if err != nil {
		t.Fatalf("creating the case: %v", err)
	}

	if _, err := q.ArchiveCase(ctx, sqlcgen.ArchiveCaseParams{ID: created.ID, Slug: project.Slug}); err != nil {
		t.Fatalf("archiving the case: %v", err)
	}

	listed, err := q.ListCases(ctx, sqlcgen.ListCasesParams{ProjectID: project.ID})
	if err != nil {
		t.Fatalf("listing the cases: %v", err)
	}
	if len(listed) != 0 {
		t.Errorf("listed %d cases after archiving, want none", len(listed))
	}

	// Archived, never deleted: the captures, comments and journal survive
	// (ADR 0014).
	if _, err := q.CaseInProject(ctx, sqlcgen.CaseInProjectParams{ID: created.ID, Slug: project.Slug}); err != nil {
		t.Errorf("an archived case must stay readable: %v", err)
	}
}

func TestACategoryHoldingACaseCannotBeDeleted(t *testing.T) {
	ctx, q := withTx(t)
	project := seedProject(t, ctx, q)

	category, err := q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: project.ID, Name: "occupied", Position: 0,
	})
	if err != nil {
		t.Fatalf("creating the category: %v", err)
	}
	if _, err := q.CreateCase(ctx, sqlcgen.CreateCaseParams{
		ProjectID: project.ID, CategoryID: &category.ID, Title: "occupant",
	}); err != nil {
		t.Fatalf("creating the case: %v", err)
	}

	rows, err := q.DeleteEmptyCategory(ctx, sqlcgen.DeleteEmptyCategoryParams{ID: category.ID, Slug: project.Slug})
	if err != nil {
		t.Fatalf("attempting the deletion: %v", err)
	}
	if rows != 0 {
		t.Errorf("deleted %d rows, want 0: a category holding a case must survive", rows)
	}
}

func TestAnEmptyCategoryIsDeleted(t *testing.T) {
	ctx, q := withTx(t)
	project := seedProject(t, ctx, q)

	category, err := q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: project.ID, Name: "empty", Position: 0,
	})
	if err != nil {
		t.Fatalf("creating the category: %v", err)
	}

	rows, err := q.DeleteEmptyCategory(ctx, sqlcgen.DeleteEmptyCategoryParams{ID: category.ID, Slug: project.Slug})
	if err != nil {
		t.Fatalf("deleting the category: %v", err)
	}
	if rows != 1 {
		t.Errorf("deleted %d rows, want 1", rows)
	}
}

func TestSiblingCategoriesCannotShareANameAtTheRoot(t *testing.T) {
	ctx, q := withTx(t)
	project := seedProject(t, ctx, q)

	if _, err := q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: project.ID, Name: "core", Position: 0,
	}); err != nil {
		t.Fatalf("creating the first category: %v", err)
	}

	// The root is where parent_id is null, and null is not equal to null in
	// SQL: without NULLS NOT DISTINCT this duplicate would be accepted.
	_, err := q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: project.ID, Name: "core", Position: 1,
	})
	if err == nil {
		t.Error("a duplicate root category was accepted, want a unique violation")
	}
}

func TestACategoryHoldingOnlyAnArchivedCaseStillCannotBeDeleted(t *testing.T) {
	ctx, q := withTx(t)
	project := seedProject(t, ctx, q)

	category, err := q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: project.ID, Name: "emptied", Position: 0,
	})
	if err != nil {
		t.Fatalf("creating the category: %v", err)
	}
	created, err := q.CreateCase(ctx, sqlcgen.CreateCaseParams{
		ProjectID: project.ID, CategoryID: &category.ID, Title: "was here",
	})
	if err != nil {
		t.Fatalf("creating the case: %v", err)
	}
	if _, err := q.ArchiveCase(ctx, sqlcgen.ArchiveCaseParams{ID: created.ID, Slug: project.Slug}); err != nil {
		t.Fatalf("archiving the case: %v", err)
	}

	// The case left the catalogue but still records where it was filed, and an
	// archived case is meant to stay whole (ADR 0014).
	rows, err := q.DeleteEmptyCategory(ctx, sqlcgen.DeleteEmptyCategoryParams{ID: category.ID, Slug: project.Slug})
	if err != nil {
		t.Fatalf("attempting the deletion: %v", err)
	}
	if rows != 0 {
		t.Error("a category holding an archived case was deleted")
	}

	counts, err := q.CountCategoryContents(ctx, &category.ID)
	if err != nil {
		t.Fatalf("counting the contents: %v", err)
	}
	if counts.Cases != 1 || counts.ArchivedCases != 1 {
		t.Errorf("counted %d cases (%d archived), want 1 and 1", counts.Cases, counts.ArchivedCases)
	}
}
