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
	archiveRows bool
	deleteRows  bool
}

func (s *stubRepo) CreateCase(_ context.Context, projectID string, categoryID *string, title string, description *string) (catalogue.Case, error) {
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

func (s *stubRepo) DeleteEmptyCategory(context.Context, string) (bool, error) {
	return s.deleteRows, nil
}

func TestATitleOfSpacesIsAMissingTitle(t *testing.T) {
	repo := &stubRepo{}
	svc := app.New(repo)

	_, err := svc.CreateCase(context.Background(), "p", nil, "   \t\n ", nil)
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

	if _, err := svc.CreateCase(context.Background(), "p", nil, "  pay by card  ", nil); err != nil {
		t.Fatalf("creating the case: %v", err)
	}
	if repo.gotTitle != "pay by card" {
		t.Errorf("stored title = %q, want it trimmed", repo.gotTitle)
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

	err := svc.DeleteCategory(context.Background(), "cat")
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
