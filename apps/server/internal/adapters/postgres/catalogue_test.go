package postgres_test

import (
	"errors"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	app "github.com/haribo/ozalid/apps/server/internal/app/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/domain/catalogue"
)

// Renaming and moving a node in one call is what #179 asked for: the vilajo
// catalogue's language fix cost 6 recreations and 17 re-parented cases.
func TestACategoryIsRenamedAndMovedInOneCall(t *testing.T) {
	ctx, repo, project, _ := intakeFixture(t)
	q := repo.Queries()

	root, err := q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: project.ID, Name: "compte", Position: 0,
	})
	if err != nil {
		t.Fatalf("creating the root: %v", err)
	}
	child, err := q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: project.ID, ParentID: &root.ID, Name: "connexion", Position: 1,
	})
	if err != nil {
		t.Fatalf("creating the child: %v", err)
	}

	name := "signing in"
	renamed, err := repo.UpdateCategory(ctx, project.Slug, child.ID, app.CategoryPatch{Name: &name})
	if err != nil {
		t.Fatalf("renaming: %v", err)
	}
	if renamed.Name != "signing in" || renamed.ParentID == nil || *renamed.ParentID != root.ID {
		t.Errorf("after the rename: %+v — the name moves, nothing else", renamed)
	}

	// And to the root, with the empty-parent form.
	moved, err := repo.UpdateCategory(ctx, project.Slug, child.ID, app.CategoryPatch{Parent: &app.CategoryParent{}})
	if err != nil {
		t.Fatalf("moving to the root: %v", err)
	}
	if moved.ParentID != nil {
		t.Errorf("parent = %v after the move, want the root", *moved.ParentID)
	}
}

func TestACategoryCannotBecomeItsOwnAncestor(t *testing.T) {
	ctx, repo, project, _ := intakeFixture(t)
	q := repo.Queries()

	top, err := q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: project.ID, Name: "top", Position: 0,
	})
	if err != nil {
		t.Fatalf("creating top: %v", err)
	}
	middle, err := q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: project.ID, ParentID: &top.ID, Name: "middle", Position: 0,
	})
	if err != nil {
		t.Fatalf("creating middle: %v", err)
	}

	// top under middle would make top its own ancestor.
	if _, err := repo.UpdateCategory(ctx, project.Slug, top.ID, app.CategoryPatch{
		Parent: &app.CategoryParent{ID: &middle.ID},
	}); !errors.Is(err, catalogue.ErrCategoryCycle) {
		t.Errorf("err = %v, want ErrCategoryCycle", err)
	}
}

func TestASiblingNameCollisionIsAConflict(t *testing.T) {
	ctx, repo, project, _ := intakeFixture(t)
	q := repo.Queries()

	if _, err := q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: project.ID, Name: "account", Position: 0,
	}); err != nil {
		t.Fatalf("creating the first: %v", err)
	}
	second, err := q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: project.ID, Name: "billing", Position: 1,
	})
	if err != nil {
		t.Fatalf("creating the second: %v", err)
	}

	name := "account"
	if _, err := repo.UpdateCategory(ctx, project.Slug, second.ID, app.CategoryPatch{Name: &name}); err == nil {
		t.Error("renaming onto a sibling's name was accepted, want a conflict")
	}
}
