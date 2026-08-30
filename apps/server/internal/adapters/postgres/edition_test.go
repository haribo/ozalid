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

// pushEdition takes in one capture of one step, from one named environment,
// and returns the address of its bytes.
func pushEdition(
	t *testing.T, ctx context.Context, repo *postgres.Repository,
	project sqlcgen.Project, kase sqlcgen.Case, body, environment string,
) string {
	t.Helper()
	hash := storeBlob(t, ctx, repo, body)
	if _, err := repo.WriteEdition(ctx, project.Slug, contract.Manifest{
		Cases: []contract.ManifestCase{{
			ID: kase.ID,
			Steps: []contract.ManifestStep{{
				Name: "opens",
				Captures: []contract.ManifestCapture{{
					Variant:    map[string]string{"theme": "light"},
					Hash:       hash,
					Provenance: contract.Provenance{EnvironmentID: environment},
				}},
			}},
		}},
	}, nil); err != nil {
		t.Fatalf("taking the edition in: %v", err)
	}
	return hash
}

func onlyCell(t *testing.T, ctx context.Context, repo *postgres.Repository, slug, caseID string) review.Cell {
	t.Helper()
	grid, err := repo.CaseGrid(ctx, slug, caseID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	if len(grid.Steps) != 1 || len(grid.Steps[0].Cells) != 1 {
		t.Fatalf("want one step with one cell, got %d steps", len(grid.Steps))
	}
	return review.Cell{StepID: grid.Steps[0].ID, VariantID: grid.Steps[0].Cells[0].VariantID}
}

func TestAnIntakeDoesNotMoveTheBytesUnderAReviewer(t *testing.T) {
	// The case sits at to-review: somebody is looking. A run landing now must
	// not change what they are judging (product.md §7).
	ctx, repo, project, kase := intakeFixture(t)
	first := pushEdition(t, ctx, repo, project, kase, "the form, first run", "ci")

	held, err := repo.Queries().GetCase(ctx, kase.ID)
	if err != nil {
		t.Fatalf("reading the case: %v", err)
	}
	if held.State != string(review.CaseToReview) {
		t.Fatalf("state = %q, want to-review after the first captures", held.State)
	}
	pinned := held.CurrentEditionID

	pushEdition(t, ctx, repo, project, kase, "the form, second run", "ci")

	after, err := repo.Queries().GetCase(ctx, kase.ID)
	if err != nil {
		t.Fatalf("re-reading the case: %v", err)
	}
	if after.CurrentEditionID == nil || pinned == nil || *after.CurrentEditionID != *pinned {
		t.Fatalf("the pointer moved while a reviewer held the case")
	}

	grid, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	if grid.Steps[0].Cells[0].Hash != first {
		t.Errorf("grid shows %q, want the bytes the reviewer opened", grid.Steps[0].Cells[0].Hash)
	}
}
func TestACaseCatchesUpOnceItsReviewEnds(t *testing.T) {
	ctx, repo, project, kase := intakeFixture(t)
	pushEdition(t, ctx, repo, project, kase, "the form, first run", "ci")
	second := pushEdition(t, ctx, repo, project, kase, "the form, second run", "ci")

	// The reviewer judges the edition they opened, and lets go.
	if _, err := repo.SaveReview(ctx, project.Slug, kase.ID, actor.Actor{ID: "nina", Kind: actor.Human}, session.Save{
		Validated: []review.Cell{onlyCell(t, ctx, repo, project.Slug, kase.ID)},
	}); err != nil {
		t.Fatalf("saving the review: %v", err)
	}

	grid, err := repo.CaseGrid(ctx, project.Slug, kase.ID, nil)
	if err != nil {
		t.Fatalf("reading the grid: %v", err)
	}
	if grid.Steps[0].Cells[0].Hash != second {
		t.Errorf("grid still shows the old edition; the case never caught up")
	}
}
