package catalogue

import (
	"context"
	"fmt"

	"github.com/haribo/ozalid/apps/server/internal/domain/catalogue"
)

// Service carries the catalogue use cases.
type Service struct {
	repo Repository
}

// New returns a Service backed by repo.
func New(repo Repository) *Service { return &Service{repo: repo} }

// CreateProject opens a book.
func (s *Service) CreateProject(ctx context.Context, slug, name string, policy catalogue.IntakePolicy) (catalogue.Project, error) {
	cleaned, err := catalogue.CleanName(name)
	if err != nil {
		return catalogue.Project{}, err
	}
	if policy == "" {
		policy = catalogue.PolicyPerCase
	}
	if !policy.Valid() {
		return catalogue.Project{}, fmt.Errorf("catalogue: %q is not an intake policy", policy)
	}
	return s.repo.CreateProject(ctx, slug, cleaned, policy)
}

// ProjectBySlug reads a project by its slug.
func (s *Service) ProjectBySlug(ctx context.Context, slug string) (catalogue.Project, error) {
	return s.repo.ProjectBySlug(ctx, slug)
}

// CreateCase opens a case and returns the id the server generated for it.
//
// The client stores that id and sends it on every subsequent edition; it never
// invents one (ADR 0014).
func (s *Service) CreateCase(ctx context.Context, projectID string, categoryID *string, title string, description *string) (catalogue.Case, error) {
	cleaned, err := catalogue.CleanTitle(title)
	if err != nil {
		return catalogue.Case{}, err
	}
	return s.repo.CreateCase(ctx, projectID, categoryID, cleaned, description)
}

// CaseByID reads a case, archived or not.
func (s *Service) CaseByID(ctx context.Context, id string) (catalogue.Case, error) {
	return s.repo.CaseByID(ctx, id)
}

// ListCases returns the catalogue, optionally filtered on the stored state.
//
// Filtering happens in the query rather than here: storing the state is what
// makes that possible without a scan (ADR 0002).
func (s *Service) ListCases(ctx context.Context, projectID string, state, categoryID *string) ([]catalogue.Case, error) {
	return s.repo.ListCases(ctx, projectID, state, categoryID)
}

// UpdateCase changes what is mutable about a case. Its id and its state are
// not part of that.
func (s *Service) UpdateCase(ctx context.Context, id, title string, description, categoryID *string) (catalogue.Case, error) {
	cleaned, err := catalogue.CleanTitle(title)
	if err != nil {
		return catalogue.Case{}, err
	}
	return s.repo.UpdateCase(ctx, id, cleaned, description, categoryID)
}

// ArchiveCase takes a case out of the catalogue without destroying it: its
// captures, comments and journal survive (ADR 0014).
func (s *Service) ArchiveCase(ctx context.Context, id string) error {
	archived, err := s.repo.ArchiveCase(ctx, id)
	if err != nil {
		return err
	}
	if !archived {
		return catalogue.ErrCaseAlreadyArchived
	}
	return nil
}

// CreateCategory adds a node to the tree.
func (s *Service) CreateCategory(ctx context.Context, projectID string, parentID *string, name string, position int32) (catalogue.Category, error) {
	cleaned, err := catalogue.CleanName(name)
	if err != nil {
		return catalogue.Category{}, err
	}
	return s.repo.CreateCategory(ctx, projectID, parentID, cleaned, position)
}

// ListCategories returns the whole tree, parents before children.
func (s *Service) ListCategories(ctx context.Context, projectID string) ([]catalogue.Category, error) {
	return s.repo.ListCategories(ctx, projectID)
}

// DeleteCategory removes an empty node.
//
// Only an empty one: deleting a filing drawer must not silently move what was
// inside it (ADR 0014).
func (s *Service) DeleteCategory(ctx context.Context, id string) error {
	deleted, err := s.repo.DeleteEmptyCategory(ctx, id)
	if err != nil {
		return err
	}
	if !deleted {
		return catalogue.ErrCategoryNotEmpty
	}
	return nil
}
