// Package catalogue holds the use cases operating on a project's book.
//
// It depends only on the interfaces declared here; the concrete adapters are
// wired in cmd/server (backend ADR 0001). That is what lets a use case be
// tested without a database.
package catalogue

import (
	"context"
	"errors"

	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	"github.com/haribo/ozalid/apps/server/internal/domain/catalogue"
)

// Repository is the outbound port this package needs.
type Repository interface {
	CreateProject(ctx context.Context, slug, name string, policy catalogue.IntakePolicy) (catalogue.Project, error)
	ProjectBySlug(ctx context.Context, slug string) (catalogue.Project, error)
	ProjectsFor(ctx context.Context, by actor.Actor, admin bool) ([]catalogue.Project, error)

	CreateCase(ctx context.Context, projectID string, categoryID string, title string, description *string) (catalogue.Case, error)
	CaseByID(ctx context.Context, slug, id string) (catalogue.Case, error)
	ListCases(ctx context.Context, projectID string, state, categoryID *string) ([]catalogue.Case, error)
	UpdateCase(ctx context.Context, slug, id string, title string, description *string, categoryID *string) (catalogue.Case, error)
	ArchiveCase(ctx context.Context, slug, id string) (bool, error)

	CreateCategory(ctx context.Context, projectID string, parentID *string, name string, position int32) (catalogue.Category, error)
	ListCategories(ctx context.Context, projectID string) ([]catalogue.Category, error)
	CategoryTree(ctx context.Context, projectID string) ([]catalogue.CategoryNode, error)

	Axes(ctx context.Context, projectID string) ([]catalogue.Axis, error)
	OrderAxes(ctx context.Context, projectID string, order []string) ([]catalogue.Axis, error)
	SummariseCases(ctx context.Context, projectID string, categoryID *string) ([]catalogue.CaseSummary, error)
	DeleteEmptyCategory(ctx context.Context, slug, id string) (bool, error)
}

// Actor identifies whoever caused a write. Every recorded fact states one, so
// the journal can always answer whether an action came from a person or a
// program (ADR 0002).
type Actor struct {
	ID   string
	Kind string // "human" or "machine"
}

// Errors the layers above match on. The adapter translates whatever its driver
// reports into one of these, so no layer above it ever knows a Postgres error
// code.
var (
	// ErrNotFound means the row the caller named does not exist.
	ErrNotFound = errors.New("catalogue: no such row")
	// ErrConflict means the write collided with something already stored — a
	// taken slug, a sibling of the same name.
	ErrConflict = errors.New("catalogue: already exists")
)
