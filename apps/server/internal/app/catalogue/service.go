package catalogue

import (
	"context"
	"fmt"
	"strings"

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

// CaseByID reads a case, archived or not, inside the project the caller named.
//
// The project is not a filter applied afterwards — it is part of the lookup,
// so a case belonging to somebody else is simply not there (product.md §8.1).
func (s *Service) CaseByID(ctx context.Context, slug, id string) (catalogue.Case, error) {
	return s.repo.CaseByID(ctx, slug, id)
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
func (s *Service) UpdateCase(ctx context.Context, slug, id, title string, description, categoryID *string) (catalogue.Case, error) {
	cleaned, err := catalogue.CleanTitle(title)
	if err != nil {
		return catalogue.Case{}, err
	}
	return s.repo.UpdateCase(ctx, slug, id, cleaned, description, categoryID)
}

// ArchiveCase takes a case out of the catalogue without destroying it: its
// captures, comments and journal survive (ADR 0014).
func (s *Service) ArchiveCase(ctx context.Context, slug, id string) error {
	archived, err := s.repo.ArchiveCase(ctx, slug, id)
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

// CategoryTree returns the whole tree, each node carrying what its entire
// branch holds — not just its direct children.
//
// Aggregating on the descendance is the point: a branch in trouble has to be
// visible from the root, or one has to descend everywhere to know where to
// descend.
func (s *Service) CategoryTree(ctx context.Context, projectID string) ([]catalogue.CategoryNode, error) {
	return s.repo.CategoryTree(ctx, projectID)
}

// Axes returns the project's rendering axes, in the order they read in.
func (s *Service) Axes(ctx context.Context, projectID string) ([]catalogue.Axis, error) {
	return s.repo.Axes(ctx, projectID)
}

// OrderAxes declares the order axes read in, and relabels the variants that
// already exist.
//
// Only the order: an axis is created by first use at intake, never here. A
// name the project does not know is ignored rather than created, because
// inventing an axis nobody captured would put an empty column in every grid.
func (s *Service) OrderAxes(ctx context.Context, projectID string, order []string) ([]catalogue.Axis, error) {
	seen := make(map[string]struct{}, len(order))
	cleaned := make([]string, 0, len(order))
	for _, name := range order {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		if _, dup := seen[trimmed]; dup {
			return nil, fmt.Errorf("catalogue: %q appears twice in the order", trimmed)
		}
		seen[trimmed] = struct{}{}
		cleaned = append(cleaned, trimmed)
	}
	return s.repo.OrderAxes(ctx, projectID, cleaned)
}

// SummariseCases returns the cases with the state of their captures, so a
// listing draws its rows without asking a second question per row.
func (s *Service) SummariseCases(ctx context.Context, projectID string, categoryID *string) ([]catalogue.CaseSummary, error) {
	return s.repo.SummariseCases(ctx, projectID, categoryID)
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
