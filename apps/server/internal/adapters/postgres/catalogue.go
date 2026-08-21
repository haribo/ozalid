package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	app "github.com/haribo/ozalid/apps/server/internal/app/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/domain/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
)

// translate turns a driver error into the vocabulary the layers above match
// on, so no Postgres error code ever leaks past this file.
func translate(op string, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		return app.ErrNotFound
	default:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return fmt.Errorf("%s: %w", op, app.ErrConflict)
		}
		return fmt.Errorf("%s: %w", op, err)
	}
}

// uniqueViolation is the SQLSTATE Postgres reports for a duplicate key.
const uniqueViolation = "23505"

func (r *Repository) CreateProject(ctx context.Context, slug, name string, policy catalogue.IntakePolicy) (catalogue.Project, error) {
	row, err := r.q.CreateProject(ctx, sqlcgen.CreateProjectParams{
		Slug: slug, Name: name, IntakePolicy: string(policy),
	})
	if err != nil {
		return catalogue.Project{}, translate("creating the project", err)
	}
	return toProject(row), nil
}

func (r *Repository) ProjectBySlug(ctx context.Context, slug string) (catalogue.Project, error) {
	row, err := r.q.GetProjectBySlug(ctx, slug)
	if err != nil {
		return catalogue.Project{}, translate("reading the project", err)
	}
	return toProject(row), nil
}

func (r *Repository) CreateCase(ctx context.Context, projectID string, categoryID *string, title string, description *string) (catalogue.Case, error) {
	row, err := r.q.CreateCase(ctx, sqlcgen.CreateCaseParams{
		ProjectID: projectID, CategoryID: categoryID, Title: title, Description: description,
	})
	if err != nil {
		return catalogue.Case{}, translate("creating the case", err)
	}
	return toCase(row), nil
}

func (r *Repository) CaseByID(ctx context.Context, id string) (catalogue.Case, error) {
	row, err := r.q.GetCase(ctx, id)
	if err != nil {
		return catalogue.Case{}, translate("reading the case", err)
	}
	return toCase(row), nil
}

func (r *Repository) ListCases(ctx context.Context, projectID string, state, categoryID *string) ([]catalogue.Case, error) {
	rows, err := r.q.ListCases(ctx, sqlcgen.ListCasesParams{
		ProjectID: projectID, State: state, CategoryID: categoryID,
	})
	if err != nil {
		return nil, translate("listing the cases", err)
	}
	out := make([]catalogue.Case, 0, len(rows))
	for _, row := range rows {
		out = append(out, toCase(row))
	}
	return out, nil
}

func (r *Repository) UpdateCase(ctx context.Context, id, title string, description, categoryID *string) (catalogue.Case, error) {
	row, err := r.q.UpdateCaseDetails(ctx, sqlcgen.UpdateCaseDetailsParams{
		ID: id, Title: title, Description: description, CategoryID: categoryID,
	})
	if err != nil {
		return catalogue.Case{}, translate("updating the case", err)
	}
	return toCase(row), nil
}

func (r *Repository) ArchiveCase(ctx context.Context, id string) (bool, error) {
	rows, err := r.q.ArchiveCase(ctx, id)
	if err != nil {
		return false, translate("archiving the case", err)
	}
	return rows > 0, nil
}

func (r *Repository) CreateCategory(ctx context.Context, projectID string, parentID *string, name string, position int32) (catalogue.Category, error) {
	row, err := r.q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ProjectID: projectID, ParentID: parentID, Name: name, Position: position,
	})
	if err != nil {
		return catalogue.Category{}, translate("creating the category", err)
	}
	return toCategory(row), nil
}

func (r *Repository) ListCategories(ctx context.Context, projectID string) ([]catalogue.Category, error) {
	rows, err := r.q.ListCategories(ctx, projectID)
	if err != nil {
		return nil, translate("listing the categories", err)
	}
	out := make([]catalogue.Category, 0, len(rows))
	for _, row := range rows {
		out = append(out, toCategory(row))
	}
	return out, nil
}

func (r *Repository) DeleteEmptyCategory(ctx context.Context, id string) (bool, error) {
	rows, err := r.q.DeleteEmptyCategory(ctx, id)
	if err != nil {
		return false, translate("deleting the category", err)
	}
	return rows > 0, nil
}

// The translations below are the whole point of this file: generated rows stay
// inside the adapter, and the layers above see domain types only.

func toProject(row sqlcgen.Project) catalogue.Project {
	return catalogue.Project{
		ID:           row.ID,
		Slug:         row.Slug,
		Name:         row.Name,
		IntakePolicy: catalogue.IntakePolicy(row.IntakePolicy),
		CreatedAt:    row.CreatedAt.Time,
	}
}

func toCase(row sqlcgen.Case) catalogue.Case {
	c := catalogue.Case{
		ID:          row.ID,
		ProjectID:   row.ProjectID,
		CategoryID:  row.CategoryID,
		Title:       row.Title,
		Description: row.Description,
		State:       review.CaseState(row.State),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
	if row.ArchivedAt.Valid {
		at := row.ArchivedAt.Time
		c.ArchivedAt = &at
	}
	return c
}

func toCategory(row sqlcgen.Category) catalogue.Category {
	return catalogue.Category{
		ID:       row.ID,
		ParentID: row.ParentID,
		Name:     row.Name,
		Position: row.Position,
	}
}
