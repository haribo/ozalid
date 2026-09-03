package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	app "github.com/haribo/ozalid/apps/server/internal/app/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	"github.com/haribo/ozalid/apps/server/internal/domain/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/domain/review"
	"github.com/haribo/ozalid/internal/contract"
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

func (r *Repository) CreateCase(ctx context.Context, projectID string, categoryID string, title string, description *string) (catalogue.Case, error) {
	row, err := r.q.CreateCase(ctx, sqlcgen.CreateCaseParams{
		ProjectID: projectID, CategoryID: &categoryID, Title: title, Description: description,
	})
	if err != nil {
		return catalogue.Case{}, translate("creating the case", err)
	}
	return toCase(row), nil
}

// CaseByID reads a case inside the project the caller named.
//
// A case belonging to another project is not found here rather than refused:
// the row does not exist for this caller, and a refusal would confirm that it
// exists elsewhere (#71).
// ProjectsFor returns the projects a caller may see: the ones they belong to,
// and every one on the instance when they administer it.
//
// A service account never administers, so its answer is the single project it
// belongs to, whatever the flag says (ADR 0018).
func (r *Repository) ProjectsFor(ctx context.Context, by actor.Actor, admin bool) ([]catalogue.Project, error) {
	out := []catalogue.Project{}

	switch by.Kind {
	case actor.Human:
		rows, err := r.q.ProjectsForUser(ctx, sqlcgen.ProjectsForUserParams{
			UserID: &by.ID, IsAdmin: admin,
		})
		if err != nil {
			return nil, translate("listing the projects", err)
		}
		for _, row := range rows {
			out = append(out, withCounts(sqlcgen.Project{
				ID: row.ID, Slug: row.Slug, Name: row.Name, IntakePolicy: row.IntakePolicy,
				CreatedAt: row.CreatedAt, PixelThreshold: row.PixelThreshold,
			}, row.People, row.Programs))
		}
	case actor.Machine:
		rows, err := r.q.ProjectsForServiceAccount(ctx, &by.ID)
		if err != nil {
			return nil, translate("listing the projects", err)
		}
		for _, row := range rows {
			out = append(out, withCounts(sqlcgen.Project{
				ID: row.ID, Slug: row.Slug, Name: row.Name, IntakePolicy: row.IntakePolicy,
				CreatedAt: row.CreatedAt, PixelThreshold: row.PixelThreshold,
			}, row.People, row.Programs))
		}
	default:
		// A kind nobody defined is nobody, and nobody sees nothing.
		return out, nil
	}
	return out, nil
}

func (r *Repository) CaseByID(ctx context.Context, slug, id string) (catalogue.Case, error) {
	row, err := r.q.CaseInProject(ctx, sqlcgen.CaseInProjectParams{ID: id, Slug: slug})
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

func (r *Repository) UpdateCase(ctx context.Context, slug, id, title string, description, categoryID *string) (catalogue.Case, error) {
	row, err := r.q.UpdateCaseDetails(ctx, sqlcgen.UpdateCaseDetailsParams{
		ID: id, Title: title, Description: description, CategoryID: categoryID, Slug: slug,
	})
	if err != nil {
		return catalogue.Case{}, translate("updating the case", err)
	}
	return toCase(row), nil
}

func (r *Repository) ArchiveCase(ctx context.Context, slug, id string) (bool, error) {
	rows, err := r.q.ArchiveCase(ctx, sqlcgen.ArchiveCaseParams{ID: id, Slug: slug})
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

func (r *Repository) DeleteEmptyCategory(ctx context.Context, slug, id string) (bool, error) {
	rows, err := r.q.DeleteEmptyCategory(ctx, sqlcgen.DeleteEmptyCategoryParams{ID: id, Slug: slug})
	if err != nil {
		return false, translate("deleting the category", err)
	}
	return rows > 0, nil
}

// The translations below are the whole point of this file: generated rows stay
// inside the adapter, and the layers above see domain types only.

// withCounts is toProject plus who reaches the project. The two counts come
// from the listing query and from nowhere else, so a project read on its own
// carries zeros rather than a number nobody computed.
func withCounts(row sqlcgen.Project, people, programs int64) catalogue.Project {
	out := toProject(row)
	out.People, out.Programs = int(people), int(programs)
	return out
}

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

func (r *Repository) CategoryTree(ctx context.Context, projectID string) ([]catalogue.CategoryNode, error) {
	rows, err := r.q.CategoryTreeWithCounts(ctx, projectID)
	if err != nil {
		return nil, translate("reading the category tree", err)
	}
	out := make([]catalogue.CategoryNode, 0, len(rows))
	for _, row := range rows {
		node := catalogue.CategoryNode{
			Category: catalogue.Category{
				ID: row.ID, ParentID: row.ParentID, Name: row.Name, Position: row.Position,
			},
			Cases: catalogue.StateCounts{
				NotInstrumented: row.NotInstrumented,
				ToReview:        row.ToReview,
				ToFix:           row.ToFix,
				Reviewed:        row.Reviewed,
			},
		}
		if row.LastActivity.Valid {
			at := row.LastActivity.Time
			node.LastActivity = &at
		}
		out = append(out, node)
	}
	return out, nil
}

func (r *Repository) SummariseCases(ctx context.Context, projectID string, categoryID *string) ([]catalogue.CaseSummary, error) {
	rows, err := r.q.CasesWithCaptureCounts(ctx, sqlcgen.CasesWithCaptureCountsParams{
		ProjectID: projectID, CategoryID: categoryID,
	})
	if err != nil {
		return nil, translate("summarising the cases", err)
	}
	out := make([]catalogue.CaseSummary, 0, len(rows))
	for _, row := range rows {
		summary := catalogue.CaseSummary{
			Case: catalogue.Case{
				ID: row.ID, ProjectID: row.ProjectID, CategoryID: row.CategoryID,
				Title: row.Title, Description: row.Description,
				State:     review.CaseState(row.State),
				CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
			},
			Captures: catalogue.CaptureCounts{
				Total: row.Captures, Validated: row.Validated,
				Commented: row.Commented, ToJudge: row.ToJudge,
			},
		}
		if row.ArchivedAt.Valid {
			at := row.ArchivedAt.Time
			summary.ArchivedAt = &at
		}
		if row.LastEdition.Valid {
			at := row.LastEdition.Time
			summary.LastEdition = &at
		}
		out = append(out, summary)
	}
	return out, nil
}

func (r *Repository) Axes(ctx context.Context, projectID string) ([]catalogue.Axis, error) {
	rows, err := r.q.ListAxes(ctx, projectID)
	if err != nil {
		return nil, translate("reading the axes", err)
	}
	out := make([]catalogue.Axis, 0, len(rows))
	for _, row := range rows {
		out = append(out, catalogue.Axis{Name: row.Name, Position: row.Position})
	}
	return out, nil
}

// OrderAxes places the named axes first, in the order given, and relabels every
// variant so the change is visible immediately.
//
// One transaction: an order applied without its relabelling would leave the
// grid reading one way and the project claiming another.
func (r *Repository) OrderAxes(ctx context.Context, projectID string, order []string) ([]catalogue.Axis, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning the reorder: %w", err)
	}
	defer func() {
		// best-effort: rolling back a committed transaction fails, and that
		// means the write went through.
		_ = tx.Rollback(ctx)
	}()
	q := r.q.WithTx(tx)

	existing, err := q.ListAxes(ctx, projectID)
	if err != nil {
		return nil, translate("reading the axes", err)
	}
	known := make(map[string]struct{}, len(existing))
	for _, axis := range existing {
		known[axis.Name] = struct{}{}
	}

	// The named axes come first, in the order given. Anything the project did
	// not name keeps its relative order after them, so omitting an axis never
	// shuffles it about.
	placed := make(map[string]struct{}, len(order))
	position := int32(0)
	final := make([]string, 0, len(existing))
	for _, name := range order {
		if _, ok := known[name]; !ok {
			// An axis nobody captured is not created here: it would put an
			// empty column in every grid.
			continue
		}
		if err := q.SetAxisPosition(ctx, sqlcgen.SetAxisPositionParams{
			ProjectID: projectID, Name: name, Position: position,
		}); err != nil {
			return nil, translate("placing an axis", err)
		}
		placed[name] = struct{}{}
		final = append(final, name)
		position++
	}
	for _, axis := range existing {
		if _, done := placed[axis.Name]; done {
			continue
		}
		if err := q.SetAxisPosition(ctx, sqlcgen.SetAxisPositionParams{
			ProjectID: projectID, Name: axis.Name, Position: position,
		}); err != nil {
			return nil, translate("placing an axis", err)
		}
		final = append(final, axis.Name)
		position++
	}

	if err := relabelVariants(ctx, q, projectID, final); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing the reorder: %w", err)
	}
	return r.Axes(ctx, projectID)
}

// relabelVariants rewrites every label to the new order. A label is a
// rendering of the values, never an identity, so rewriting one changes nothing
// about what it points at.
func relabelVariants(ctx context.Context, q *sqlcgen.Queries, projectID string, order []string) error {
	variants, err := q.ListVariants(ctx, projectID)
	if err != nil {
		return translate("reading the variants", err)
	}
	for _, variant := range variants {
		values := map[string]string{}
		if err := json.Unmarshal(variant.Values, &values); err != nil {
			return fmt.Errorf("decoding a variant: %w", err)
		}
		if err := q.RelabelVariant(ctx, sqlcgen.RelabelVariantParams{
			ID: variant.ID, Label: contract.VariantLabel(values, order),
		}); err != nil {
			return translate("relabelling a variant", err)
		}
	}
	return nil
}
