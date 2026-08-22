package http

import (
	"context"
	"errors"
	"net/http"

	app "github.com/haribo/ozalid/apps/server/internal/app/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/domain/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/ports/http/openapi"
)

// CreateProject opens a book.
func (s *Server) CreateProject(ctx context.Context, request openapi.CreateProjectRequestObject) (openapi.CreateProjectResponseObject, error) {
	policy := catalogue.PolicyPerCase
	if request.Body.IntakePolicy != nil {
		policy = catalogue.IntakePolicy(*request.Body.IntakePolicy)
	}

	project, err := s.catalogue.CreateProject(ctx, request.Body.Slug, request.Body.Name, policy)
	switch {
	case errors.Is(err, catalogue.ErrNameRequired):
		return openapi.CreateProject400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: openapi.BadRequestApplicationProblemPlusJSONResponse(
				problem("invalid-project", "A project needs a name", http.StatusBadRequest, ""),
			),
		}, nil
	case errors.Is(err, app.ErrConflict):
		return openapi.CreateProject409ApplicationProblemPlusJSONResponse(
			problem("slug-taken", "That slug is taken", http.StatusConflict,
				"Another project in this instance already uses this slug."),
		), nil
	case err != nil:
		return nil, err
	}
	return openapi.CreateProject201JSONResponse(toAPIProject(project)), nil
}

// GetProject reads a project by slug.
func (s *Server) GetProject(ctx context.Context, request openapi.GetProjectRequestObject) (openapi.GetProjectResponseObject, error) {
	project, err := s.catalogue.ProjectBySlug(ctx, request.Slug)
	if errors.Is(err, app.ErrNotFound) {
		return openapi.GetProject404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound("project")}, nil
	}
	if err != nil {
		return nil, err
	}
	return openapi.GetProject200JSONResponse(toAPIProject(project)), nil
}

// CreateCase opens a case and returns the id the server generated.
func (s *Server) CreateCase(ctx context.Context, request openapi.CreateCaseRequestObject) (openapi.CreateCaseResponseObject, error) {
	project, err := s.catalogue.ProjectBySlug(ctx, request.Slug)
	if errors.Is(err, app.ErrNotFound) {
		return openapi.CreateCase404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound("project")}, nil
	}
	if err != nil {
		return nil, err
	}

	created, err := s.catalogue.CreateCase(ctx, project.ID, request.Body.CategoryId, request.Body.Title, request.Body.Description)
	switch {
	case errors.Is(err, catalogue.ErrTitleRequired):
		return openapi.CreateCase400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: openapi.BadRequestApplicationProblemPlusJSONResponse(
				problem("invalid-case", "A case needs a title", http.StatusBadRequest, ""),
			),
		}, nil
	case err != nil:
		return nil, err
	}
	return openapi.CreateCase201JSONResponse(toAPICase(created)), nil
}

// ListCases returns the catalogue, filtered on the stored state.
func (s *Server) ListCases(ctx context.Context, request openapi.ListCasesRequestObject) (openapi.ListCasesResponseObject, error) {
	project, err := s.catalogue.ProjectBySlug(ctx, request.Slug)
	if errors.Is(err, app.ErrNotFound) {
		return openapi.ListCases404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound("project")}, nil
	}
	if err != nil {
		return nil, err
	}

	var state *string
	if request.Params.State != nil {
		v := string(*request.Params.State)
		state = &v
	}

	// One query carrying the capture counts too: a listing must not ask a
	// second question per row to draw its gauge.
	cases, err := s.catalogue.SummariseCases(ctx, project.ID, request.Params.CategoryId)
	if err != nil {
		return nil, err
	}

	out := make([]openapi.Case, 0, len(cases))
	for _, c := range cases {
		if state != nil && string(c.State) != *state {
			continue
		}
		out = append(out, toAPISummary(c))
	}
	return openapi.ListCases200JSONResponse(out), nil
}

// GetCase reads a case, archived or not.
func (s *Server) GetCase(ctx context.Context, request openapi.GetCaseRequestObject) (openapi.GetCaseResponseObject, error) {
	found, err := s.catalogue.CaseByID(ctx, request.CaseId)
	if errors.Is(err, app.ErrNotFound) {
		return openapi.GetCase404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound("case")}, nil
	}
	if err != nil {
		return nil, err
	}
	return openapi.GetCase200JSONResponse(toAPICase(found)), nil
}

// UpdateCase changes what is mutable about a case.
func (s *Server) UpdateCase(ctx context.Context, request openapi.UpdateCaseRequestObject) (openapi.UpdateCaseResponseObject, error) {
	updated, err := s.catalogue.UpdateCase(ctx, request.CaseId, request.Body.Title, request.Body.Description, request.Body.CategoryId)
	switch {
	case errors.Is(err, catalogue.ErrTitleRequired):
		return openapi.UpdateCase400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: openapi.BadRequestApplicationProblemPlusJSONResponse(
				problem("invalid-case", "A case needs a title", http.StatusBadRequest, ""),
			),
		}, nil
	case errors.Is(err, app.ErrNotFound):
		return openapi.UpdateCase404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound("case")}, nil
	case err != nil:
		return nil, err
	}
	return openapi.UpdateCase200JSONResponse(toAPICase(updated)), nil
}

// ArchiveCase takes a case out of the catalogue without destroying it.
func (s *Server) ArchiveCase(ctx context.Context, request openapi.ArchiveCaseRequestObject) (openapi.ArchiveCaseResponseObject, error) {
	if _, err := s.catalogue.CaseByID(ctx, request.CaseId); errors.Is(err, app.ErrNotFound) {
		return openapi.ArchiveCase404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound("case")}, nil
	} else if err != nil {
		return nil, err
	}

	err := s.catalogue.ArchiveCase(ctx, request.CaseId)
	switch {
	case errors.Is(err, catalogue.ErrCaseAlreadyArchived):
		return openapi.ArchiveCase409ApplicationProblemPlusJSONResponse(
			problem("already-archived", "The case is already archived", http.StatusConflict, ""),
		), nil
	case err != nil:
		return nil, err
	}
	return openapi.ArchiveCase204Response{}, nil
}

// ListCategories returns the whole tree.
func (s *Server) ListCategories(ctx context.Context, request openapi.ListCategoriesRequestObject) (openapi.ListCategoriesResponseObject, error) {
	project, err := s.catalogue.ProjectBySlug(ctx, request.Slug)
	if errors.Is(err, app.ErrNotFound) {
		return openapi.ListCategories404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound("project")}, nil
	}
	if err != nil {
		return nil, err
	}

	found, err := s.catalogue.CategoryTree(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	out := make([]openapi.Category, 0, len(found))
	for _, c := range found {
		out = append(out, toAPINode(c))
	}
	return openapi.ListCategories200JSONResponse(out), nil
}

// CreateCategory adds a node to the tree.
func (s *Server) CreateCategory(ctx context.Context, request openapi.CreateCategoryRequestObject) (openapi.CreateCategoryResponseObject, error) {
	project, err := s.catalogue.ProjectBySlug(ctx, request.Slug)
	if errors.Is(err, app.ErrNotFound) {
		return openapi.CreateCategory404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse: notFound("project")}, nil
	}
	if err != nil {
		return nil, err
	}

	var position int32
	if request.Body.Position != nil {
		position = int32(*request.Body.Position)
	}

	created, err := s.catalogue.CreateCategory(ctx, project.ID, request.Body.ParentId, request.Body.Name, position)
	switch {
	case errors.Is(err, catalogue.ErrNameRequired):
		return openapi.CreateCategory400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: openapi.BadRequestApplicationProblemPlusJSONResponse(
				problem("invalid-category", "A category needs a name", http.StatusBadRequest, ""),
			),
		}, nil
	case errors.Is(err, app.ErrConflict):
		return openapi.CreateCategory409ApplicationProblemPlusJSONResponse(
			problem("sibling-name-taken", "A sibling already carries that name", http.StatusConflict, ""),
		), nil
	case err != nil:
		return nil, err
	}
	return openapi.CreateCategory201JSONResponse(toAPICategory(created)), nil
}

// DeleteCategory removes an empty node.
func (s *Server) DeleteCategory(ctx context.Context, request openapi.DeleteCategoryRequestObject) (openapi.DeleteCategoryResponseObject, error) {
	err := s.catalogue.DeleteCategory(ctx, request.CategoryId)
	switch {
	case errors.Is(err, catalogue.ErrCategoryNotEmpty):
		// Deliberately not distinguished from "no such category": both mean
		// the row is not there to delete, and telling them apart would leak
		// whether an id exists.
		return openapi.DeleteCategory409ApplicationProblemPlusJSONResponse(
			problem("category-not-empty", "The category is not empty", http.StatusConflict,
				"It holds a sub-category or a case. An archived case counts: it still records where it was filed."),
		), nil
	case err != nil:
		return nil, err
	}
	return openapi.DeleteCategory204Response{}, nil
}

func notFound(what string) openapi.NotFoundApplicationProblemPlusJSONResponse {
	return openapi.NotFoundApplicationProblemPlusJSONResponse(
		problem(what+"-not-found", "No such "+what, http.StatusNotFound, ""),
	)
}

func toAPIProject(p catalogue.Project) openapi.Project {
	return openapi.Project{
		Id:           p.ID,
		Slug:         p.Slug,
		Name:         p.Name,
		IntakePolicy: openapi.ProjectIntakePolicy(p.IntakePolicy),
		CreatedAt:    p.CreatedAt,
	}
}

func toAPICase(c catalogue.Case) openapi.Case {
	return openapi.Case{
		Id:          c.ID,
		ProjectId:   c.ProjectID,
		CategoryId:  c.CategoryID,
		Title:       c.Title,
		Description: c.Description,
		State:       openapi.CaseState(c.State),
		Archived:    c.Archived(),
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

func toAPICategory(c catalogue.Category) openapi.Category {
	return openapi.Category{
		Id:       c.ID,
		ParentId: c.ParentID,
		Name:     c.Name,
		Position: int(c.Position),
	}
}

// toAPINode carries what the whole branch holds, which is what a landing page
// draws its gauge from.
func toAPINode(n catalogue.CategoryNode) openapi.Category {
	out := toAPICategory(n.Category)
	out.Cases = openapi.StateCounts{
		NotInstrumented: int(n.Cases.NotInstrumented),
		ToReview:        int(n.Cases.ToReview),
		ToFix:           int(n.Cases.ToFix),
		Reviewed:        int(n.Cases.Reviewed),
	}
	out.LastActivity = n.LastActivity
	return out
}

func toAPISummary(s catalogue.CaseSummary) openapi.Case {
	out := toAPICase(s.Case)
	out.Captures = &openapi.CaptureCounts{
		Total:     int(s.Captures.Total),
		Validated: int(s.Captures.Validated),
		Commented: int(s.Captures.Commented),
		ToJudge:   int(s.Captures.ToJudge),
	}
	out.LastEdition = s.LastEdition
	return out
}

// ListAxes returns the project's rendering axes, in the order they read in.
func (s *Server) ListAxes(ctx context.Context, request openapi.ListAxesRequestObject) (openapi.ListAxesResponseObject, error) {
	project, err := s.catalogue.ProjectBySlug(ctx, request.Slug)
	if errors.Is(err, app.ErrNotFound) {
		return openapi.ListAxes404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("project"),
		}, nil
	}
	if err != nil {
		return nil, err
	}

	axes, err := s.catalogue.Axes(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	return openapi.ListAxes200JSONResponse(toAPIAxes(axes)), nil
}

// OrderAxes declares the order axes read in, and relabels what already exists.
func (s *Server) OrderAxes(ctx context.Context, request openapi.OrderAxesRequestObject) (openapi.OrderAxesResponseObject, error) {
	project, err := s.catalogue.ProjectBySlug(ctx, request.Slug)
	if errors.Is(err, app.ErrNotFound) {
		return openapi.OrderAxes404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("project"),
		}, nil
	}
	if err != nil {
		return nil, err
	}

	axes, err := s.catalogue.OrderAxes(ctx, project.ID, request.Body.Order)
	if err != nil {
		return openapi.OrderAxes400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: openapi.BadRequestApplicationProblemPlusJSONResponse(
				problem("invalid-axis-order", "That order cannot be applied", http.StatusBadRequest, err.Error()),
			),
		}, nil
	}
	return openapi.OrderAxes200JSONResponse(toAPIAxes(axes)), nil
}

func toAPIAxes(axes []catalogue.Axis) []openapi.Axis {
	out := make([]openapi.Axis, 0, len(axes))
	for _, a := range axes {
		out = append(out, openapi.Axis{Name: a.Name, Position: int(a.Position)})
	}
	return out
}
