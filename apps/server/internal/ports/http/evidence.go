package http

import (
	"context"
	"errors"

	app "github.com/haribo/ozalid/apps/server/internal/app/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/app/evidence"
	"net/http"

	"github.com/haribo/ozalid/apps/server/internal/domain/access"
	"github.com/haribo/ozalid/apps/server/internal/ports/http/openapi"
	"github.com/haribo/ozalid/internal/contract"
)

// GetCaseCaptures returns the grid a case is judged from.
func (s *Server) GetCaseCaptures(ctx context.Context, request openapi.GetCaseCapturesRequestObject) (openapi.GetCaseCapturesResponseObject, error) {
	if why, no := s.mayNot(ctx, request.Slug, access.ReadProject); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.GetCaseCaptures401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.GetCaseCaptures403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}
	grid, err := s.evidence.Grid(ctx, request.Slug, request.CaseId, request.Params.EditionId)
	if errors.Is(err, app.ErrNotFound) {
		return openapi.GetCaseCaptures404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("case"),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return openapi.GetCaseCaptures200JSONResponse(toAPIGrid(grid)), nil
}

// GetReviewQueue returns the captures awaiting the reviewer under a category,
// or under the whole project (product.md §3.6).
func (s *Server) GetReviewQueue(ctx context.Context, request openapi.GetReviewQueueRequestObject) (openapi.GetReviewQueueResponseObject, error) {
	if why, no := s.mayNot(ctx, request.Slug, access.ReadProject); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.GetReviewQueue401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.GetReviewQueue403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}
	entries, err := s.evidence.Queue(ctx, request.Slug, request.Params.CategoryId)
	if errors.Is(err, app.ErrNotFound) {
		return openapi.GetReviewQueue404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("project"),
		}, nil
	}
	if err != nil {
		return nil, err
	}

	out := openapi.ReviewQueue{Entries: make([]openapi.QueueEntry, 0, len(entries))}
	for _, e := range entries {
		out.Entries = append(out.Entries, openapi.QueueEntry{
			CaseId: e.CaseID, CaseTitle: e.CaseTitle, CategoryId: e.CategoryID,
			StepId: e.StepID, StepName: e.StepName, StepPosition: e.StepPos,
			Variant: openapi.GridVariant{
				Id: e.Variant.ID, Label: e.Variant.Label, Values: e.Variant.Values,
			},
			Capture: toAPICapture(e.Capture),
		})
	}
	return openapi.GetReviewQueue200JSONResponse(out), nil
}

func toAPIGrid(g evidence.Grid) openapi.Grid {
	out := openapi.Grid{
		CaseId:     g.CaseID,
		Variants:   make([]openapi.GridVariant, 0, len(g.Variants)),
		Steps:      make([]openapi.GridStep, 0, len(g.Steps)),
		Recordings: make([]openapi.GridRecording, 0, len(g.Recordings)),
	}
	if g.EditionID != "" {
		out.EditionId = &g.EditionID
		takenAt := g.TakenAt
		out.TakenAt = &takenAt
	}
	if g.Revision != "" {
		out.Revision = &g.Revision
	}

	for _, v := range g.Variants {
		out.Variants = append(out.Variants, openapi.GridVariant{
			Id: v.ID, Label: v.Label, Values: v.Values,
		})
	}
	for _, st := range g.Steps {
		step := openapi.GridStep{
			Id: st.ID, Name: st.Name, Position: st.Position,
			Captures: make([]openapi.GridCapture, 0, len(st.Captures)),
		}
		for _, c := range st.Captures {
			step.Captures = append(step.Captures, toAPICapture(c))
		}
		out.Steps = append(out.Steps, step)
	}
	for _, r := range g.Recordings {
		out.Recordings = append(out.Recordings, openapi.GridRecording{
			Id: r.ID, VariantId: r.VariantID, Hash: r.Hash,
			Status:  openapi.GridRecordingStatus(r.Status),
			Refusal: nonEmptyPtr(r.Refusal),
		})
	}
	return out
}

// toAPICapture renders one capture, wherever it is read from: the grid and the
// queue show the same thing and must not drift into two renderings of it.
func toAPICapture(c evidence.Capture) openapi.GridCapture {
	return openapi.GridCapture{
		Id:          c.ID,
		VariantId:   c.VariantID,
		Hash:        c.Hash,
		Status:      openapi.GridCaptureStatus(c.Status),
		MovedPixels: c.MovedPixels,
		Provenance:  toAPIProvenance(c.Provenance),
	}
}

// toAPIProvenance omits the whole object when nothing was recorded, rather than
// serialising five empty strings.
func toAPIProvenance(p contract.Provenance) *openapi.Provenance {
	if p == (contract.Provenance{}) {
		return nil
	}
	return &openapi.Provenance{
		Os:             optionalString(p.OS),
		Browser:        optionalString(p.Browser),
		BrowserVersion: optionalString(p.BrowserVersion),
		Resolution:     optionalString(p.Resolution),
		EnvironmentId:  optionalString(p.EnvironmentID),
	}
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
