package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres"
	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	app "github.com/haribo/ozalid/apps/server/internal/app/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/app/session"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
)

// A second project, sharing the instance with the fixture's own.
//
// It exists so the tests below can name a real slug that the case does not
// belong to. Naming a slug nobody has would pass for the wrong reason: a
// missing project and a case in somebody else's project must both read as
// nothing, and only the second one is the invariant #71 is about.
func neighbour(t *testing.T, ctx context.Context, repo *postgres.Repository) sqlcgen.Project {
	t.Helper()
	project, err := repo.Queries().CreateProject(ctx, sqlcgen.CreateProjectParams{
		Slug:         fmt.Sprintf("neighbour-%d", time.Now().UnixNano()),
		Name:         t.Name() + " — the other team",
		IntakePolicy: "per-case",
	})
	if err != nil {
		t.Fatalf("creating the neighbouring project: %v", err)
	}
	t.Cleanup(func() {
		if _, err := repo.Pool().Exec(ctx, "DELETE FROM projects WHERE id = $1", project.ID); err != nil {
			t.Errorf("cleaning up the neighbouring project: %v", err)
		}
	})
	return project
}

func TestACaseIsNotFoundUnderAProjectItDoesNotBelongTo(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	other := neighbour(t, ctx, repo)

	// The case is there under its own project: what follows is about the slug,
	// not about the case existing.
	if _, err := repo.CaseByID(ctx, project.Slug, kase.ID); err != nil {
		t.Fatalf("reading the case under its own project: %v", err)
	}

	_, err := repo.CaseByID(ctx, other.Slug, kase.ID)
	if !errors.Is(err, app.ErrNotFound) {
		t.Errorf("CaseByID under the wrong project = %v, want ErrNotFound", err)
	}
}

func TestAGridIsNotReadableThroughSomebodyElsesProject(t *testing.T) {
	ctx, repo, blobs, project, kase := freshnessFixture(t)
	other := neighbour(t, ctx, repo)

	if err := takeIn(t, ctx, repo, blobs, project, kase, screen(t, ctx, repo, blobs, 10, 0)); err != nil {
		t.Fatalf("taking the edition in: %v", err)
	}

	// Under its own project the grid has the capture that was just pushed.
	grid, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	if len(grid.Steps) == 0 {
		t.Fatal("the fixture produced no step, so this test proves nothing")
	}

	if _, err := repo.CaseGrid(ctx, other.Slug, kase.ID, nil); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("CaseGrid under the wrong project = %v, want ErrNotFound", err)
	}
}

func TestAVerdictCannotBeRecordedThroughSomebodyElsesProject(t *testing.T) {
	ctx, repo, blobs, project, kase := freshnessFixture(t)
	other := neighbour(t, ctx, repo)

	if err := takeIn(t, ctx, repo, blobs, project, kase, screen(t, ctx, repo, blobs, 10, 0)); err != nil {
		t.Fatalf("taking the edition in: %v", err)
	}
	cell := onlyCell(t, ctx, repo, project.Slug, kase.ID)

	_, err := repo.SaveReview(ctx, other.Slug, kase.ID, actor.Actor{ID: "nina", Kind: actor.Human},
		session.Save{Validated: []review.Cell{cell}})
	if !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("SaveReview under the wrong project = %v, want ErrNotFound", err)
	}

	// And nothing was written: the refusal happens before the transaction
	// records anything.
	grid, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the grid back: %v", err)
	}
	if got := grid.Steps[0].Cells[0].Status; got != "to-review" {
		t.Errorf("status = %q, want it untouched at %q", got, "to-review")
	}
}

func TestCommentsAreNotReadableThroughSomebodyElsesProject(t *testing.T) {
	ctx, repo, blobs, project, kase := freshnessFixture(t)
	other := neighbour(t, ctx, repo)

	if err := takeIn(t, ctx, repo, blobs, project, kase, screen(t, ctx, repo, blobs, 10, 0)); err != nil {
		t.Fatalf("taking the edition in: %v", err)
	}
	commentOn(t, ctx, repo, project.Slug, kase.ID, onlyCell(t, ctx, repo, project.Slug, kase.ID))

	said, err := repo.OfCase(ctx, project.Slug, kase.ID)
	if err != nil || len(said) != 1 {
		t.Fatalf("OfCase under its own project = %d comments, %v; want 1 and no error", len(said), err)
	}

	if _, err := repo.OfCase(ctx, other.Slug, kase.ID); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("OfCase under the wrong project = %v, want ErrNotFound", err)
	}
}
