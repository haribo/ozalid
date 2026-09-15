package catalogue_test

import (
	"context"
	"errors"
	"testing"

	app "github.com/haribo/ozalid/apps/server/internal/app/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/domain/catalogue"
)

// stubRepo records what the service asked for and answers whatever the test
// set up. The use cases are testable without a database because they depend on
// the interface they declared, not on an adapter (backend ADR 0001).
type stubRepo struct {
	app.Repository // unimplemented methods panic if a test reaches one

	gotTitle    string
	gotName     string
	gotPatch    *app.CasePatch
	archiveRows bool
	deleteRows  bool
}

func (s *stubRepo) UpdateCase(_ context.Context, _, _ string, patch app.CasePatch) (catalogue.Case, error) {
	s.gotPatch = &patch
	return catalogue.Case{ID: "abc123456789"}, nil
}

func (s *stubRepo) CreateCase(_ context.Context, projectID string, categoryID string, title string, description *string) (catalogue.Case, error) {
	s.gotTitle = title
	return catalogue.Case{ID: "abc123456789", Title: title}, nil
}

func (s *stubRepo) CreateCategory(_ context.Context, projectID string, parentID *string, name string, position int32) (catalogue.Category, error) {
	s.gotName = name
	return catalogue.Category{ID: "cat123456789", Name: name}, nil
}

func (s *stubRepo) ArchiveCase(context.Context, string, string) (bool, error) {
	return s.archiveRows, nil
}

func (s *stubRepo) DeleteEmptyCategory(context.Context, string, string) (bool, error) {
	return s.deleteRows, nil
}

func TestATitleOfSpacesIsAMissingTitle(t *testing.T) {
	repo := &stubRepo{}
	svc := app.New(repo)

	_, err := svc.CreateCase(context.Background(), "p", "cat123456789", "   \t\n ", nil)
	if !errors.Is(err, catalogue.ErrTitleRequired) {
		t.Errorf("err = %v, want ErrTitleRequired", err)
	}
	if repo.gotTitle != "" {
		t.Error("the repository was called with a blank title")
	}
}

func TestSurroundingSpaceIsTrimmedBeforeStoring(t *testing.T) {
	repo := &stubRepo{}
	svc := app.New(repo)

	if _, err := svc.CreateCase(context.Background(), "p", "cat123456789", "  pay by card  ", nil); err != nil {
		t.Fatalf("creating the case: %v", err)
	}
	if repo.gotTitle != "pay by card" {
		t.Errorf("stored title = %q, want it trimmed", repo.gotTitle)
	}
}

// An update may move a case, never unfile it: a blank category is a malformed
// request, answered as such rather than written (#115, #229).
func TestAPatchCannotEmptyTheCategory(t *testing.T) {
	repo := &stubRepo{}
	svc := app.New(repo)

	blank := "   "
	_, err := svc.UpdateCase(context.Background(), "atlas", "abc123456789", app.CasePatch{CategoryID: &blank})
	if !errors.Is(err, catalogue.ErrCategoryRequired) {
		t.Errorf("err = %v, want ErrCategoryRequired", err)
	}
	if repo.gotPatch != nil {
		t.Error("the repository was called with a blank category")
	}
}

// A title is cleaned when the patch carries one, and left alone when it does
// not — an update of the description must not have to repeat the title (#229).
func TestAPatchedTitleIsTrimmedAndAnAbsentOneIsLeftAlone(t *testing.T) {
	repo := &stubRepo{}
	svc := app.New(repo)

	title := "  pay by card  "
	if _, err := svc.UpdateCase(context.Background(), "atlas", "abc123456789", app.CasePatch{Title: &title}); err != nil {
		t.Fatalf("patching the title: %v", err)
	}
	if repo.gotPatch == nil || repo.gotPatch.Title == nil || *repo.gotPatch.Title != "pay by card" {
		t.Errorf("patched title = %v, want it trimmed", repo.gotPatch)
	}

	description := "with a saved card"
	if _, err := svc.UpdateCase(context.Background(), "atlas", "abc123456789", app.CasePatch{Description: &description}); err != nil {
		t.Fatalf("patching the description: %v", err)
	}
	if repo.gotPatch.Title != nil {
		t.Errorf("title = %q on a patch that never named it, want nil", *repo.gotPatch.Title)
	}
}

func TestArchivingATwiceArchivedCaseIsReported(t *testing.T) {
	svc := app.New(&stubRepo{archiveRows: false})

	err := svc.ArchiveCase(context.Background(), "atlas", "abc")
	if !errors.Is(err, catalogue.ErrCaseAlreadyArchived) {
		t.Errorf("err = %v, want ErrCaseAlreadyArchived", err)
	}
}

func TestDeletingANonEmptyCategoryIsReportedRatherThanSilentlyIgnored(t *testing.T) {
	// The query deletes nothing when the category holds something; without
	// this the caller would get a cheerful 204 and the category would still be
	// there (ADR 0014).
	svc := app.New(&stubRepo{deleteRows: false})

	err := svc.DeleteCategory(context.Background(), "atlas", "cat")
	if !errors.Is(err, catalogue.ErrCategoryNotEmpty) {
		t.Errorf("err = %v, want ErrCategoryNotEmpty", err)
	}
}

func TestAnUnknownIntakePolicyIsRefused(t *testing.T) {
	svc := app.New(&stubRepo{})

	_, err := svc.CreateProject(context.Background(), "slug", "Name", catalogue.IntakePolicy("whenever"))
	if err == nil {
		t.Error("an unknown intake policy was accepted")
	}
}
